package main

/*
#include <stdint.h>
#include <string.h>
typedef signed char     sdbyte;
typedef unsigned char   udbyte;
typedef signed short    sdint2;
typedef unsigned short  udint2;
typedef signed int      sdint4;
typedef unsigned int    udint4;
typedef long long int   sdint8;
typedef unsigned long long int udint8;
typedef sdint8          slength;
typedef udint8          ulength;
typedef void*           dpointer;
typedef sdint2          DPIRETURN;
typedef void*           dhandle;
typedef dhandle         dhenv;
typedef dhandle         dhcon;
typedef dhandle         dhstmt;
typedef dhandle         dhdesc;
*/
import "C"
import (
	"database/sql"
	"fmt"
	"sync"
	"unsafe"
)

// columnInfo holds metadata about a result set column.
type columnInfo struct {
	name        string
	sqlType     int16
	precision   uint64
	scale       int16
	nullable    int16
	displaySize int64
	tableName   string
}

// paramInfo holds metadata about a statement parameter.
type paramInfo struct {
	name      string
	sqlType   int16
	precision uint64
	scale     int16
	nullable  int16
	paramType int16 // DSQL_PARAM_INPUT, etc.
}

// bindColInfo holds info about a column binding.
type bindColInfo struct {
	cType     int16
	dataPtr   unsafe.Pointer
	bufLen    int64
	indPtr    *C.slength
	actLenPtr *C.slength
}

// bindParamInfo holds info about a parameter binding.
type bindParamInfo struct {
	paramType int16
	cType     int16
	sqlType   int16
	precision uint64
	scale     int16
	dataPtr   unsafe.Pointer
	bufLen    int64
	indPtr    *C.slength
	actLenPtr *C.slength
}

// stmtHandle represents a DPI statement.
type stmtHandle struct {
	mu   sync.Mutex
	conn *connHandle

	// Prepared statement
	prepared *sql.Stmt
	sql      string

	// Result set
	rows        *sql.Rows
	columns     []columnInfo
	columnCount int16

	// Cached result rows (pre-fetched for accurate row count)
	cachedRows  [][]interface{} // all rows, each row is []interface{}
	fetchPos    int             // current position in cachedRows for dpi_fetch

	// Parameters
	params     []paramInfo
	paramCount uint16

	// Bindings
	colBindings   map[int]bindColInfo   // 1-based column index
	paramBindings map[int]bindParamInfo // 1-based parameter index

	// Current row data (from last fetch)
	currentRow  []interface{}
	rowsFetched int64

	// Statement attributes
	cursorType     uint32
	rowArraySize   uint64
	paramsetSize   uint64
	rowStatusPtr   unsafe.Pointer
	rowsFetchedPtr unsafe.Pointer

	// Descriptors (implicit)
	impRowDesc   *descHandle
	impParamDesc *descHandle

	// DML result
	rowsAffected int64

	// Diagnostics
	lastErr *diagInfo
}

// descHandle represents a DPI descriptor.
type descHandle struct {
	mu       sync.Mutex
	stmt     *stmtHandle
	descType int // 0 = row, 1 = param
	lastErr  *diagInfo
}

func newStmtHandle(conn *connHandle) *stmtHandle {
	st := &stmtHandle{
		conn:          conn,
		colBindings:   make(map[int]bindColInfo),
		paramBindings: make(map[int]bindParamInfo),
		rowArraySize:  1,
		paramsetSize:  1,
		cursorType:    DSQL_CURSOR_FORWARD_ONLY,
	}
	// Create implicit descriptors
	st.impRowDesc = &descHandle{stmt: st, descType: 0}
	st.impParamDesc = &descHandle{stmt: st, descType: 1}
	return st
}

//export dpi_alloc_stmt
func dpi_alloc_stmt(hcon C.dhcon, pstmt *C.dhstmt) C.DPIRETURN {
	if pstmt == nil {
		return DSQL_ERROR
	}
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt := newStmtHandle(conn)
	stmtID := allocHandle(stmt)
	// Also allocate handles for the implicit descriptors
	allocHandle(stmt.impRowDesc)
	allocHandle(stmt.impParamDesc)
	*pstmt = C.dhstmt(handleToPtr(stmtID))
	return DSQL_SUCCESS
}

//export dpi_free_stmt
func dpi_free_stmt(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	if stmt.rows != nil {
		stmt.rows.Close()
		stmt.rows = nil
	}
	if stmt.prepared != nil {
		stmt.prepared.Close()
		stmt.prepared = nil
	}
	stmt.mu.Unlock()

	id := ptrToHandle(unsafe.Pointer(hstmt))
	freeHandle(id)
	return DSQL_SUCCESS
}

//export dpi_alloc_handle
func dpi_alloc_handle(hndlType C.sdint2, hndlIn C.dhandle, hndlOut *C.dhandle) C.DPIRETURN {
	if hndlOut == nil {
		return DSQL_ERROR
	}
	switch int16(hndlType) {
	case DSQL_HANDLE_ENV:
		return dpi_alloc_env((*C.dhenv)(unsafe.Pointer(hndlOut)))
	case DSQL_HANDLE_DBC:
		return dpi_alloc_con(C.dhenv(unsafe.Pointer(hndlIn)), (*C.dhcon)(unsafe.Pointer(hndlOut)))
	case DSQL_HANDLE_STMT:
		return dpi_alloc_stmt(C.dhcon(unsafe.Pointer(hndlIn)), (*C.dhstmt)(unsafe.Pointer(hndlOut)))
	case DSQL_HANDLE_DESC:
		return dpi_alloc_desc(C.dhcon(unsafe.Pointer(hndlIn)), (*C.dhdesc)(unsafe.Pointer(hndlOut)))
	default:
		return DSQL_ERROR
	}
}

//export dpi_free_handle
func dpi_free_handle(hndlType C.sdint2, hndl C.dhandle) C.DPIRETURN {
	switch int16(hndlType) {
	case DSQL_HANDLE_ENV:
		return dpi_free_env(C.dhenv(unsafe.Pointer(hndl)))
	case DSQL_HANDLE_DBC:
		return dpi_free_con(C.dhcon(unsafe.Pointer(hndl)))
	case DSQL_HANDLE_STMT:
		return dpi_free_stmt(C.dhstmt(unsafe.Pointer(hndl)))
	default:
		id := ptrToHandle(unsafe.Pointer(hndl))
		freeHandle(id)
		return DSQL_SUCCESS
	}
}

//export dpi_set_stmt_attr
func dpi_set_stmt_attr(hstmt C.dhstmt, attrID C.sdint4, val C.dpointer, valLen C.sdint4) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	intVal := uint64(uintptr(val))

	switch int32(attrID) {
	case DSQL_ATTR_CURSOR_TYPE:
		stmt.cursorType = uint32(intVal)
	case DSQL_ATTR_ROW_ARRAY_SIZE:
		if intVal > 0 {
			stmt.rowArraySize = intVal
		}
	case DSQL_ATTR_PARAMSET_SIZE:
		if intVal > 0 {
			stmt.paramsetSize = intVal
		}
	case DSQL_ATTR_ROW_STATUS_PTR:
		stmt.rowStatusPtr = unsafe.Pointer(val)
	case DSQL_ATTR_ROWS_FETCHED_PTR:
		stmt.rowsFetchedPtr = unsafe.Pointer(val)
	case DSQL_ATTR_QUERY_TIMEOUT, DSQL_ATTR_MAX_ROWS,
		DSQL_ATTR_CURSOR_SCROLLABLE, DSQL_ATTR_SQL_CHARSET,
		DSQL_ATTR_IGN_BP_ERR:
		// Accept but ignore
	default:
		// Ignore unknown attributes
	}
	return DSQL_SUCCESS
}

//export dpi_get_stmt_attr
func dpi_get_stmt_attr(hstmt C.dhstmt, attrID C.sdint4, val C.dpointer, bufLen C.sdint4, valLen *C.sdint4) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	switch int32(attrID) {
	case DSQL_ATTR_IMP_ROW_DESC:
		// Return the descriptor handle for row description
		descID := findDescHandleID(stmt.impRowDesc)
		*(*C.dhdesc)(val) = C.dhdesc(handleToPtr(descID))
		if valLen != nil {
			*valLen = C.sdint4(unsafe.Sizeof(C.dhdesc(nil)))
		}
	case DSQL_ATTR_IMP_PARAM_DESC:
		descID := findDescHandleID(stmt.impParamDesc)
		*(*C.dhdesc)(val) = C.dhdesc(handleToPtr(descID))
		if valLen != nil {
			*valLen = C.sdint4(unsafe.Sizeof(C.dhdesc(nil)))
		}
	case DSQL_ATTR_CURSOR_TYPE:
		*(*C.udint4)(val) = C.udint4(stmt.cursorType)
		if valLen != nil {
			*valLen = 4
		}
	case DSQL_ATTR_ROW_ARRAY_SIZE:
		*(*C.ulength)(val) = C.ulength(stmt.rowArraySize)
		if valLen != nil {
			*valLen = C.sdint4(unsafe.Sizeof(C.ulength(0)))
		}
	default:
		*(*C.sdint4)(val) = 0
		if valLen != nil {
			*valLen = 4
		}
	}
	return DSQL_SUCCESS
}

// findDescHandleID finds the handle ID for a descHandle, allocating one if needed.
func findDescHandleID(desc *descHandle) uintptr {
	handleMu.RLock()
	for id, obj := range handles {
		if obj == desc {
			handleMu.RUnlock()
			return id
		}
	}
	handleMu.RUnlock()
	// Not found — allocate
	return allocHandle(desc)
}

//export dpi_alloc_desc
func dpi_alloc_desc(hcon C.dhcon, pdesc *C.dhdesc) C.DPIRETURN {
	if pdesc == nil {
		return DSQL_ERROR
	}
	desc := &descHandle{}
	id := allocHandle(desc)
	*pdesc = C.dhdesc(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_free_desc
func dpi_free_desc(hdesc C.dhdesc) C.DPIRETURN {
	id := ptrToHandle(unsafe.Pointer(hdesc))
	freeHandle(id)
	return DSQL_SUCCESS
}

//export dpi_prepare
func dpi_prepare(hstmt C.dhstmt, sqlTxt *C.sdbyte) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sqlStr := C.GoString((*C.char)(unsafe.Pointer(sqlTxt)))
	stmt.sql = sqlStr

	// Close any previous resources
	if stmt.rows != nil {
		stmt.rows.Close()
		stmt.rows = nil
	}
	if stmt.prepared != nil {
		stmt.prepared.Close()
		stmt.prepared = nil
	}

	// Reset state
	stmt.columns = nil
	stmt.columnCount = 0
	stmt.paramCount = 0
	stmt.params = nil
	stmt.currentRow = nil
	stmt.rowsAffected = 0
	stmt.cachedRows = nil
	stmt.fetchPos = 0
	stmt.colBindings = make(map[int]bindColInfo)

	// Prepare the statement
	if stmt.conn.db == nil {
		stmt.lastErr = &diagInfo{errorCode: -1, message: "Not connected"}
		return DSQL_ERROR
	}

	prepared, dbErr := stmt.conn.db.Prepare(sqlStr)
	if dbErr != nil {
		stmt.lastErr = &diagInfo{
			errorCode: -1,
			message:   fmt.Sprintf("Prepare failed: %v", dbErr),
		}
		return DSQL_ERROR
	}
	stmt.prepared = prepared

	// Count parameters (count '?' in SQL)
	paramCount := uint16(0)
	inStr := false
	for _, ch := range sqlStr {
		if ch == '\'' {
			inStr = !inStr
		} else if ch == '?' && !inStr {
			paramCount++
		}
	}
	stmt.paramCount = paramCount
	stmt.params = make([]paramInfo, paramCount)
	for i := range stmt.params {
		stmt.params[i].paramType = DSQL_PARAM_INPUT
		stmt.params[i].sqlType = DSQL_VARCHAR
		stmt.params[i].nullable = DSQL_NULLABLE_UNKNOWN
	}

	return DSQL_SUCCESS
}

//export dpi_prepareW
func dpi_prepareW(hstmt C.dhstmt, sqlTxt *C.sdbyte, sqlLen C.sdint4) C.DPIRETURN {
	return dpi_prepare(hstmt, sqlTxt)
}

//export dpi_exec
func dpi_exec(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	if stmt.prepared == nil {
		stmt.lastErr = &diagInfo{errorCode: -1, message: "Statement not prepared"}
		return DSQL_ERROR
	}

	// Build args from parameter bindings
	args := buildExecArgs(stmt)

	// Determine if this is a query or exec
	if isQuery(stmt.sql) {
		rows, dbErr := stmt.prepared.Query(args...)
		if dbErr != nil {
			stmt.lastErr = diagFromError(dbErr)
			return DSQL_ERROR
		}
		stmt.rows = rows
		stmt.rowsAffected = 0
		populateColumnInfo(stmt)
		if dbErr := cacheAllRows(stmt); dbErr != nil {
			stmt.lastErr = diagFromError(dbErr)
			return DSQL_ERROR
		}
	} else {
		result, dbErr := stmt.prepared.Exec(args...)
		if dbErr != nil {
			stmt.lastErr = diagFromError(dbErr)
			return DSQL_ERROR
		}
		affected, _ := result.RowsAffected()
		stmt.rowsAffected = affected
		stmt.rows = nil
		stmt.columnCount = 0
	}

	return DSQL_SUCCESS
}

//export dpi_exec_direct
func dpi_exec_direct(hstmt C.dhstmt, sqlTxt *C.sdbyte) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sqlStr := C.GoString((*C.char)(unsafe.Pointer(sqlTxt)))
	stmt.sql = sqlStr

	// Close any previous resources
	if stmt.rows != nil {
		stmt.rows.Close()
		stmt.rows = nil
	}
	if stmt.prepared != nil {
		stmt.prepared.Close()
		stmt.prepared = nil
	}
	stmt.columns = nil
	stmt.currentRow = nil

	if stmt.conn.db == nil {
		stmt.lastErr = &diagInfo{errorCode: -1, message: "Not connected"}
		return DSQL_ERROR
	}

	stmt.cachedRows = nil
	stmt.fetchPos = 0

	if isQuery(sqlStr) {
		rows, dbErr := stmt.conn.db.Query(sqlStr)
		if dbErr != nil {
			stmt.lastErr = diagFromError(dbErr)
			return DSQL_ERROR
		}
		stmt.rows = rows
		populateColumnInfo(stmt)
		if dbErr := cacheAllRows(stmt); dbErr != nil {
			stmt.lastErr = diagFromError(dbErr)
			return DSQL_ERROR
		}
	} else {
		result, dbErr := stmt.conn.db.Exec(sqlStr)
		if dbErr != nil {
			stmt.lastErr = diagFromError(dbErr)
			return DSQL_ERROR
		}
		affected, _ := result.RowsAffected()
		stmt.rowsAffected = affected
		stmt.rows = nil
		stmt.columnCount = 0
	}

	return DSQL_SUCCESS
}

//export dpi_exec_directW
func dpi_exec_directW(hstmt C.dhstmt, sqlTxt *C.sdbyte, sqlLen C.sdint4) C.DPIRETURN {
	return dpi_exec_direct(hstmt, sqlTxt)
}

//export dpi_cancel
func dpi_cancel(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()
	if stmt.rows != nil {
		stmt.rows.Close()
		stmt.rows = nil
	}
	return DSQL_SUCCESS
}

// isQuery checks if a SQL string is a query that returns rows.
func isQuery(sqlStr string) bool {
	s := sqlStr
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n' || s[0] == '\r' || s[0] == '(') {
		s = s[1:]
	}
	if len(s) < 4 {
		return false
	}
	// Case-insensitive prefix check
	upper := make([]byte, 0, 7)
	for i := 0; i < len(s) && len(upper) < 7; i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		upper = append(upper, c)
	}
	prefix := string(upper)
	return len(prefix) >= 6 && prefix[:6] == "SELECT" ||
		len(prefix) >= 4 && prefix[:4] == "WITH" ||
		len(prefix) >= 4 && prefix[:4] == "SHOW" ||
		len(prefix) >= 6 && prefix[:6] == "EXPLAI"
}

// populateColumnInfo fills in column metadata from sql.Rows.
func populateColumnInfo(stmt *stmtHandle) {
	if stmt.rows == nil {
		stmt.columnCount = 0
		stmt.columns = nil
		return
	}

	colTypes, err := stmt.rows.ColumnTypes()
	if err != nil {
		colNames, _ := stmt.rows.Columns()
		stmt.columnCount = int16(len(colNames))
		stmt.columns = make([]columnInfo, len(colNames))
		for i, name := range colNames {
			stmt.columns[i] = columnInfo{
				name:        name,
				sqlType:     DSQL_VARCHAR,
				precision:   256,
				nullable:    DSQL_NULLABLE_UNKNOWN,
				displaySize: 256,
			}
		}
		return
	}

	stmt.columnCount = int16(len(colTypes))
	stmt.columns = make([]columnInfo, len(colTypes))
	for i, ct := range colTypes {
		col := columnInfo{
			name:     ct.Name(),
			nullable: DSQL_NULLABLE_UNKNOWN,
		}

		col.sqlType, col.precision, col.scale, col.displaySize = mapGoTypeToDPI(ct)

		nullable, ok := ct.Nullable()
		if ok {
			if nullable {
				col.nullable = DSQL_NULLABLE
			} else {
				col.nullable = DSQL_NO_NULLS
			}
		}

		stmt.columns[i] = col
	}
}

// cacheAllRows pre-fetches all rows from stmt.rows into stmt.cachedRows,
// then closes stmt.rows. This gives us an accurate row count for dpi_row_count.
func cacheAllRows(stmt *stmtHandle) error {
	if stmt.rows == nil {
		return nil
	}
	numCols := int(stmt.columnCount)
	stmt.cachedRows = nil
	stmt.fetchPos = 0

	for stmt.rows.Next() {
		scanDest := make([]interface{}, numCols)
		for i := 0; i < numCols; i++ {
			scanDest[i] = new(interface{})
		}
		if err := stmt.rows.Scan(scanDest...); err != nil {
			return err
		}
		row := make([]interface{}, numCols)
		for i := 0; i < numCols; i++ {
			row[i] = *(scanDest[i].(*interface{}))
		}
		stmt.cachedRows = append(stmt.cachedRows, row)
	}

	// Close the rows now that we've cached everything
	stmt.rows.Close()
	stmt.rows = nil
	return nil
}

// mapGoTypeToDPI maps a sql.ColumnType to DPI types.
func mapGoTypeToDPI(ct *sql.ColumnType) (sqlType int16, precision uint64, scale int16, displaySize int64) {
	dbTypeName := ct.DatabaseTypeName()

	switch dbTypeName {
	case "INT", "INTEGER":
		return DSQL_INT, 10, 0, 11
	case "BIGINT":
		return DSQL_BIGINT, 19, 0, 20
	case "SMALLINT":
		return DSQL_SMALLINT, 5, 0, 6
	case "TINYINT":
		return DSQL_TINYINT, 3, 0, 4
	case "FLOAT":
		return DSQL_FLOAT, 24, 0, 24
	case "DOUBLE", "DOUBLE PRECISION":
		return DSQL_DOUBLE, 53, 0, 53
	case "DECIMAL", "DEC", "NUMBER", "NUMERIC":
		p, s, _ := ct.DecimalSize()
		return DSQL_DEC, uint64(p), int16(s), int64(p) + 2
	case "CHAR":
		l, _ := ct.Length()
		return DSQL_CHAR, uint64(l), 0, int64(l)
	case "VARCHAR", "VARCHAR2":
		l, _ := ct.Length()
		if l == 0 {
			l = 256
		}
		return DSQL_VARCHAR, uint64(l), 0, int64(l)
	case "BLOB":
		return DSQL_BLOB, 0, 0, 0
	case "CLOB", "TEXT":
		return DSQL_CLOB, 0, 0, 0
	case "DATE":
		return DSQL_DATE, 10, 0, 10
	case "TIME":
		return DSQL_TIME, 8, 0, 8
	case "TIMESTAMP", "DATETIME":
		return DSQL_TIMESTAMP, 26, 6, 26
	case "TIMESTAMP WITH TIME ZONE":
		return DSQL_TIMESTAMP_TZ, 34, 6, 34
	case "BIT", "BOOL", "BOOLEAN":
		return DSQL_BIT, 1, 0, 1
	case "BINARY":
		l, _ := ct.Length()
		return DSQL_BINARY, uint64(l), 0, int64(l) * 2
	case "VARBINARY":
		l, _ := ct.Length()
		return DSQL_VARBINARY, uint64(l), 0, int64(l) * 2
	default:
		l, _ := ct.Length()
		if l == 0 {
			l = 256
		}
		return DSQL_VARCHAR, uint64(l), 0, int64(l)
	}
}

// buildExecArgs builds []interface{} from parameter bindings.
func buildExecArgs(stmt *stmtHandle) []interface{} {
	if len(stmt.paramBindings) == 0 {
		return nil
	}
	args := make([]interface{}, stmt.paramCount)
	for i := 0; i < int(stmt.paramCount); i++ {
		bind, ok := stmt.paramBindings[i+1] // 1-based
		if !ok {
			args[i] = nil
			continue
		}
		args[i] = extractBoundValue(bind)
	}
	return args
}

// diagFromError creates a diagInfo from a Go error.
func diagFromError(err error) *diagInfo {
	if err == nil {
		return nil
	}
	return &diagInfo{
		errorCode: -1,
		message:   err.Error(),
	}
}
