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

typedef struct {
    sdint2  year;
    udint2  month;
    udint2  day;
    udint2  hour;
    udint2  minute;
    udint2  second;
    udint4  fraction;
} dpi_timestamp_t;

typedef struct {
    sdint2  year;
    udint2  month;
    udint2  day;
} dpi_date_t;

typedef struct {
    udint2  hour;
    udint2  minute;
    udint2  second;
} dpi_time_t;

#define DPI_MAX_NUMERIC_LEN 16
typedef struct {
    udbyte  precision;
    signed char scale;
    udbyte  sign;
    udbyte  val[DPI_MAX_NUMERIC_LEN];
} dpi_numeric_t;
*/
import "C"
import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

//export dpi_number_columns
func dpi_number_columns(hstmt C.dhstmt, colCnt *C.sdint2) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	if colCnt != nil {
		*colCnt = C.sdint2(stmt.columnCount)
	}
	return DSQL_SUCCESS
}

//export dpi_desc_column
func dpi_desc_column(hstmt C.dhstmt, icol C.sdint2, name *C.sdbyte, bufLen C.sdint2,
	nameLen *C.sdint2, sqltype *C.sdint2, colSz *C.ulength,
	decDigits *C.sdint2, nullable *C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	idx := int(icol) - 1 // icol is 1-based
	if idx < 0 || idx >= len(stmt.columns) {
		stmt.lastErr = &diagInfo{errorCode: -1, message: fmt.Sprintf("Column index %d out of range", icol)}
		return DSQL_ERROR
	}

	col := &stmt.columns[idx]

	if name != nil && bufLen > 0 {
		n := cStringLen(name, int(bufLen), col.name)
		if nameLen != nil {
			*nameLen = C.sdint2(n)
		}
	} else if nameLen != nil {
		*nameLen = C.sdint2(len(col.name))
	}

	if sqltype != nil {
		*sqltype = C.sdint2(col.sqlType)
	}
	if colSz != nil {
		*colSz = C.ulength(col.precision)
	}
	if decDigits != nil {
		*decDigits = C.sdint2(col.scale)
	}
	if nullable != nil {
		*nullable = C.sdint2(col.nullable)
	}

	return DSQL_SUCCESS
}

//export dpi_desc_columnW
func dpi_desc_columnW(hstmt C.dhstmt, icol C.sdint2, name *C.sdbyte, bufLen C.sdint2,
	nameLen *C.sdint2, sqltype *C.sdint2, colSz *C.ulength,
	decDigits *C.sdint2, nullable *C.sdint2) C.DPIRETURN {
	return dpi_desc_column(hstmt, icol, name, bufLen, nameLen, sqltype, colSz, decDigits, nullable)
}

//export dpi_col_attr
func dpi_col_attr(hstmt C.dhstmt, icol C.udint2, fldID C.udint2,
	chrAttr C.dpointer, bufLen C.sdint2, chrAttrLen *C.sdint2,
	numAttr *C.slength) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	idx := int(icol) - 1
	if idx < 0 || idx >= len(stmt.columns) {
		if int16(fldID) == DSQL_COLUMN_COUNT {
			if numAttr != nil {
				*numAttr = C.slength(stmt.columnCount)
			}
			return DSQL_SUCCESS
		}
		return DSQL_ERROR
	}

	col := &stmt.columns[idx]
	field := int16(fldID)

	switch field {
	case DSQL_COLUMN_COUNT:
		if numAttr != nil {
			*numAttr = C.slength(stmt.columnCount)
		}
	case DSQL_COLUMN_NAME:
		if chrAttr != nil {
			n := cStringLen((*C.sdbyte)(chrAttr), int(bufLen), col.name)
			if chrAttrLen != nil {
				*chrAttrLen = C.sdint2(n)
			}
		}
	case DSQL_COLUMN_TYPE:
		if numAttr != nil {
			*numAttr = C.slength(col.sqlType)
		}
	case DSQL_COLUMN_LENGTH:
		if numAttr != nil {
			*numAttr = C.slength(col.precision)
		}
	case DSQL_COLUMN_PRECISION:
		if numAttr != nil {
			*numAttr = C.slength(col.precision)
		}
	case DSQL_COLUMN_SCALE:
		if numAttr != nil {
			*numAttr = C.slength(col.scale)
		}
	case DSQL_COLUMN_DISPLAY_SIZE:
		if numAttr != nil {
			*numAttr = C.slength(col.displaySize)
		}
	case DSQL_COLUMN_NULLABLE:
		if numAttr != nil {
			*numAttr = C.slength(col.nullable)
		}
	case DSQL_COLUMN_TABLE_NAME:
		if chrAttr != nil {
			n := cStringLen((*C.sdbyte)(chrAttr), int(bufLen), col.tableName)
			if chrAttrLen != nil {
				*chrAttrLen = C.sdint2(n)
			}
		}
	default:
		if numAttr != nil {
			*numAttr = 0
		}
	}

	return DSQL_SUCCESS
}

//export dpi_col_attrW
func dpi_col_attrW(hstmt C.dhstmt, icol C.udint2, fldID C.udint2,
	chrAttr C.dpointer, bufLen C.sdint2, chrAttrLen *C.sdint2,
	numAttr *C.slength) C.DPIRETURN {
	return dpi_col_attr(hstmt, icol, fldID, chrAttr, bufLen, chrAttrLen, numAttr)
}

//export dpi_bind_col
func dpi_bind_col(hstmt C.dhstmt, icol C.udint2, ctype C.sdint2,
	val C.dpointer, bufLen C.slength, ind *C.slength) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	stmt.colBindings[int(icol)] = bindColInfo{
		cType:   int16(ctype),
		dataPtr: unsafe.Pointer(val),
		bufLen:  int64(bufLen),
		indPtr:  ind,
	}
	return DSQL_SUCCESS
}

//export dpi_bind_col2
func dpi_bind_col2(hstmt C.dhstmt, icol C.udint2, ctype C.sdint2,
	val C.dpointer, bufLen C.slength, ind *C.slength, actLen *C.slength) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	stmt.colBindings[int(icol)] = bindColInfo{
		cType:     int16(ctype),
		dataPtr:   unsafe.Pointer(val),
		bufLen:    int64(bufLen),
		indPtr:    ind,
		actLenPtr: actLen,
	}
	return DSQL_SUCCESS
}

//export dpi_unbind_columns
func dpi_unbind_columns(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()
	stmt.colBindings = make(map[int]bindColInfo)
	return DSQL_SUCCESS
}

//export dpi_fetch
func dpi_fetch(hstmt C.dhstmt, rowNum *C.ulength) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	if stmt.cachedRows == nil || stmt.fetchPos >= len(stmt.cachedRows) {
		if rowNum != nil {
			*rowNum = 0
		}
		return DSQL_NO_DATA
	}

	// Calculate per-element sizes for each bound column
	type colElemInfo struct {
		elemSize  uintptr // size of one element in the buffer
		indStride uintptr // stride for indicator array (sizeof(slength))
	}
	colElems := make(map[int]colElemInfo)
	for colIdx, bind := range stmt.colBindings {
		elemSz := cTypeSize(bind.cType)
		if elemSz == 0 {
			// For variable-length types, bufLen is already the per-element size
			// (var->bufferSize from dmPython, total buffer = allocatedElements * bufferSize)
			elemSz = uintptr(bind.bufLen)
		}
		colElems[colIdx] = colElemInfo{
			elemSize:  elemSz,
			indStride: unsafe.Sizeof(C.slength(0)),
		}
	}

	// Fetch up to rowArraySize rows from cache
	fetched := uint64(0)
	numCols := int(stmt.columnCount)
	for fetched < stmt.rowArraySize && stmt.fetchPos < len(stmt.cachedRows) {
		row := stmt.cachedRows[stmt.fetchPos]
		stmt.currentRow = row

		// Write values to bound column buffers at the correct array offset
		for colIdx := 1; colIdx <= numCols; colIdx++ {
			bind, ok := stmt.colBindings[colIdx]
			if !ok {
				continue
			}
			ei := colElems[colIdx]
			// Create an offset binding for this row
			rowBind := bind
			if fetched > 0 && ei.elemSize > 0 {
				rowBind.dataPtr = unsafe.Pointer(uintptr(bind.dataPtr) + uintptr(fetched)*ei.elemSize)
				if bind.indPtr != nil {
					rowBind.indPtr = (*C.slength)(unsafe.Pointer(uintptr(unsafe.Pointer(bind.indPtr)) + uintptr(fetched)*ei.indStride))
				}
				if bind.actLenPtr != nil {
					rowBind.actLenPtr = (*C.slength)(unsafe.Pointer(uintptr(unsafe.Pointer(bind.actLenPtr)) + uintptr(fetched)*ei.indStride))
				}
			}
			writeValueToBinding(row[colIdx-1], rowBind, stmt.columns[colIdx-1].sqlType)
		}

		stmt.fetchPos++
		fetched++

		// For array fetch, set row status
		if stmt.rowStatusPtr != nil {
			statusArr := (*[1 << 20]C.udint2)(stmt.rowStatusPtr)
			statusArr[fetched-1] = 0 // DSQL_ROW_SUCCESS
		}
	}

	stmt.rowsFetched = int64(fetched)

	// Set rows fetched pointer
	if stmt.rowsFetchedPtr != nil {
		*(*C.ulength)(stmt.rowsFetchedPtr) = C.ulength(fetched)
	}

	if rowNum != nil {
		*rowNum = C.ulength(fetched)
	}

	if fetched == 0 {
		return DSQL_NO_DATA
	}

	return DSQL_SUCCESS
}

// cTypeSize returns the fixed size of a C type, or 0 for variable-length types.
func cTypeSize(cType int16) uintptr {
	switch cType {
	case DSQL_C_STINYINT, DSQL_C_UTINYINT, DSQL_C_BIT:
		return 1
	case DSQL_C_SSHORT, DSQL_C_USHORT:
		return 2
	case DSQL_C_SLONG, DSQL_C_ULONG:
		return 4
	case DSQL_C_SBIGINT, DSQL_C_UBIGINT:
		return 8
	case DSQL_C_FLOAT:
		return 4
	case DSQL_C_DOUBLE:
		return 8
	case DSQL_C_TIMESTAMP:
		return unsafe.Sizeof(C.dpi_timestamp_t{})
	case DSQL_C_DATE:
		return unsafe.Sizeof(C.dpi_date_t{})
	case DSQL_C_TIME:
		return unsafe.Sizeof(C.dpi_time_t{})
	case DSQL_C_NUMERIC:
		return 19 // DPI_MAX_NUMERIC_LEN(16) + precision + scale + sign
	default:
		return 0 // variable-length (string, binary, etc.)
	}
}

//export dpi_fetch_scroll
func dpi_fetch_scroll(hstmt C.dhstmt, orient C.sdint2, offset C.slength, rowNum *C.ulength) C.DPIRETURN {
	// For now, only support FETCH_NEXT
	return dpi_fetch(hstmt, rowNum)
}

//export dpi_get_data
func dpi_get_data(hstmt C.dhstmt, icol C.udint2, ctype C.sdint2,
	val C.dpointer, bufLen C.slength, valLen *C.slength) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	idx := int(icol) - 1
	if idx < 0 || idx >= len(stmt.currentRow) {
		return DSQL_ERROR
	}

	rawVal := stmt.currentRow[idx]
	if rawVal == nil {
		if valLen != nil {
			*valLen = C.slength(DSQL_NULL_DATA)
		}
		return DSQL_SUCCESS
	}

	bind := bindColInfo{
		cType:   int16(ctype),
		dataPtr: unsafe.Pointer(val),
		bufLen:  int64(bufLen),
		indPtr:  valLen,
	}

	sqlType := int16(DSQL_VARCHAR)
	if idx < len(stmt.columns) {
		sqlType = stmt.columns[idx].sqlType
	}

	writeValueToBinding(rawVal, bind, sqlType)
	return DSQL_SUCCESS
}

//export dpi_get_data2
func dpi_get_data2(hstmt C.dhstmt, icol C.udint2, ctype C.sdint2,
	val C.dpointer, bufLen C.slength, valLen *C.slength, actLen *C.slength) C.DPIRETURN {
	return dpi_get_data(hstmt, icol, ctype, val, bufLen, valLen)
}

//export dpi_get_dataW
func dpi_get_dataW(hstmt C.dhstmt, icol C.udint2, ctype C.sdint2,
	val C.dpointer, bufLen C.slength, valLen *C.slength) C.DPIRETURN {
	return dpi_get_data(hstmt, icol, ctype, val, bufLen, valLen)
}

//export dpi_close_cursor
func dpi_close_cursor(hstmt C.dhstmt) C.DPIRETURN {
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
	stmt.cachedRows = nil
	stmt.fetchPos = 0
	stmt.currentRow = nil
	stmt.columns = nil
	stmt.columnCount = 0
	return DSQL_SUCCESS
}

//export dpi_more_results
func dpi_more_results(hstmt C.dhstmt) C.DPIRETURN {
	// Go's database/sql doesn't easily support multiple result sets
	return DSQL_NO_DATA
}

//export dpi_row_count
func dpi_row_count(hstmt C.dhstmt, rowNum *C.sdint8) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	if rowNum != nil {
		if stmt.cachedRows != nil {
			// Return the actual cached row count for SELECT queries.
			*rowNum = C.sdint8(len(stmt.cachedRows))
		} else {
			*rowNum = C.sdint8(stmt.rowsAffected)
		}
	}
	return DSQL_SUCCESS
}

// writeValueToBinding writes a Go value into a C buffer according to the binding info.
func writeValueToBinding(val interface{}, bind bindColInfo, sqlType int16) {
	if val == nil {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}

	if bind.dataPtr == nil {
		// Just report the length
		s := fmt.Sprintf("%v", val)
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(len(s))
		}
		return
	}

	cType := bind.cType

	switch cType {
	case DSQL_C_NCHAR, DSQL_C_CHAR, DSQL_C_WCHAR:
		writeStringValue(val, bind)
	case DSQL_C_SLONG:
		writeInt32Value(val, bind)
	case DSQL_C_ULONG:
		writeUint32Value(val, bind)
	case DSQL_C_SSHORT:
		writeInt16Value(val, bind)
	case DSQL_C_USHORT:
		writeUint16Value(val, bind)
	case DSQL_C_SBIGINT:
		writeInt64Value(val, bind)
	case DSQL_C_UBIGINT:
		writeUint64Value(val, bind)
	case DSQL_C_FLOAT:
		writeFloat32Value(val, bind)
	case DSQL_C_DOUBLE:
		writeFloat64Value(val, bind)
	case DSQL_C_STINYINT:
		writeInt8Value(val, bind)
	case DSQL_C_UTINYINT:
		writeUint8Value(val, bind)
	case DSQL_C_BIT:
		writeBitValue(val, bind)
	case DSQL_C_BINARY:
		writeBinaryValue(val, bind)
	case DSQL_C_TIMESTAMP:
		writeTimestampValue(val, bind)
	case DSQL_C_DATE:
		writeDateValue(val, bind)
	case DSQL_C_TIME:
		writeTimeValue(val, bind)
	case DSQL_C_NUMERIC:
		writeNumericValue(val, bind)
	default:
		// Default: treat as string
		writeStringValue(val, bind)
	}
}

func writeStringValue(val interface{}, bind bindColInfo) {
	var s string
	switch v := val.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case time.Time:
		s = v.Format("2006-01-02 15:04:05.000000")
	case sql.NullString:
		if !v.Valid {
			if bind.indPtr != nil {
				*bind.indPtr = C.slength(DSQL_NULL_DATA)
			}
			return
		}
		s = v.String
	default:
		s = fmt.Sprintf("%v", v)
	}

	b := []byte(s)
	n := len(b)
	bufLen := int(bind.bufLen)

	if bind.cType == DSQL_C_NCHAR || bind.cType == DSQL_C_WCHAR {
		// NTS: include null terminator space
		if n >= bufLen {
			n = bufLen - 1
		}
	} else if bind.cType == DSQL_C_CHAR {
		// No null terminator
		if n > bufLen {
			n = bufLen
		}
	}

	if n > 0 {
		C.memcpy(bind.dataPtr, unsafe.Pointer(&b[0]), C.size_t(n))
	}

	// Null terminate for NCHAR/WCHAR
	if bind.cType == DSQL_C_NCHAR || bind.cType == DSQL_C_WCHAR {
		*(*C.sdbyte)(unsafe.Pointer(uintptr(bind.dataPtr) + uintptr(n))) = 0
	}

	if bind.indPtr != nil {
		*bind.indPtr = C.slength(len(b))
	}
	if bind.actLenPtr != nil {
		*bind.actLenPtr = C.slength(n)
	}
}

func toInt64(val interface{}) (int64, bool) {
	switch v := val.(type) {
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case int:
		return int64(v), true
	case int16:
		return int64(v), true
	case int8:
		return int64(v), true
	case uint64:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint8:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n, true
		}
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return int64(f), true
		}
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case []byte:
		return toInt64(string(v))
	}
	return 0, false
}

func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case int:
		return float64(v), true
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f, true
		}
	case []byte:
		return toFloat64(string(v))
	}
	return 0, false
}

func writeInt32Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.sdint4)(bind.dataPtr) = C.sdint4(n)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.sdint4(0)))
	}
	if bind.actLenPtr != nil {
		*bind.actLenPtr = C.slength(unsafe.Sizeof(C.sdint4(0)))
	}
}

func writeUint32Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.udint4)(bind.dataPtr) = C.udint4(n)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.udint4(0)))
	}
}

func writeInt16Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.sdint2)(bind.dataPtr) = C.sdint2(n)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.sdint2(0)))
	}
}

func writeUint16Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.udint2)(bind.dataPtr) = C.udint2(n)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.udint2(0)))
	}
}

func writeInt64Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.sdint8)(bind.dataPtr) = C.sdint8(n)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.sdint8(0)))
	}
	if bind.actLenPtr != nil {
		*bind.actLenPtr = C.slength(unsafe.Sizeof(C.sdint8(0)))
	}
}

func writeUint64Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.udint8)(bind.dataPtr) = C.udint8(n)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.udint8(0)))
	}
}

func writeFloat32Value(val interface{}, bind bindColInfo) {
	f, ok := toFloat64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.float)(bind.dataPtr) = C.float(f)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.float(0)))
	}
}

func writeFloat64Value(val interface{}, bind bindColInfo) {
	f, ok := toFloat64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.double)(bind.dataPtr) = C.double(f)
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.double(0)))
	}
	if bind.actLenPtr != nil {
		*bind.actLenPtr = C.slength(unsafe.Sizeof(C.double(0)))
	}
}

func writeInt8Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.sdbyte)(bind.dataPtr) = C.sdbyte(n)
	if bind.indPtr != nil {
		*bind.indPtr = 1
	}
}

func writeUint8Value(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	*(*C.udbyte)(bind.dataPtr) = C.udbyte(n)
	if bind.indPtr != nil {
		*bind.indPtr = 1
	}
}

func writeBitValue(val interface{}, bind bindColInfo) {
	n, ok := toInt64(val)
	if !ok {
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}
	if n != 0 {
		*(*C.sdint4)(bind.dataPtr) = 1
	} else {
		*(*C.sdint4)(bind.dataPtr) = 0
	}
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.sdint4(0)))
	}
}

func writeBinaryValue(val interface{}, bind bindColInfo) {
	var b []byte
	switch v := val.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		b = []byte(fmt.Sprintf("%v", v))
	}

	n := len(b)
	if n > int(bind.bufLen) {
		n = int(bind.bufLen)
	}
	if n > 0 {
		C.memcpy(bind.dataPtr, unsafe.Pointer(&b[0]), C.size_t(n))
	}
	if bind.indPtr != nil {
		*bind.indPtr = C.slength(len(b))
	}
	if bind.actLenPtr != nil {
		*bind.actLenPtr = C.slength(n)
	}
}

func writeTimestampValue(val interface{}, bind bindColInfo) {
	var t time.Time
	switch v := val.(type) {
	case time.Time:
		t = v
	case string:
		// Try parsing various formats
		formats := []string{
			"2006-01-02 15:04:05.999999999",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, f := range formats {
			if parsed, err := time.Parse(f, v); err == nil {
				t = parsed
				break
			}
		}
	default:
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}

	ts := (*C.dpi_timestamp_t)(bind.dataPtr)
	ts.year = C.sdint2(t.Year())
	ts.month = C.udint2(t.Month())
	ts.day = C.udint2(t.Day())
	ts.hour = C.udint2(t.Hour())
	ts.minute = C.udint2(t.Minute())
	ts.second = C.udint2(t.Second())
	ts.fraction = C.udint4(t.Nanosecond())

	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.dpi_timestamp_t{}))
	}
	if bind.actLenPtr != nil {
		*bind.actLenPtr = C.slength(unsafe.Sizeof(C.dpi_timestamp_t{}))
	}
}

func writeDateValue(val interface{}, bind bindColInfo) {
	var t time.Time
	switch v := val.(type) {
	case time.Time:
		t = v
	case string:
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			t = parsed
		}
	default:
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}

	d := (*C.dpi_date_t)(bind.dataPtr)
	d.year = C.sdint2(t.Year())
	d.month = C.udint2(t.Month())
	d.day = C.udint2(t.Day())

	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.dpi_date_t{}))
	}
}

func writeTimeValue(val interface{}, bind bindColInfo) {
	var t time.Time
	switch v := val.(type) {
	case time.Time:
		t = v
	case string:
		if parsed, err := time.Parse("15:04:05", v); err == nil {
			t = parsed
		}
	default:
		if bind.indPtr != nil {
			*bind.indPtr = C.slength(DSQL_NULL_DATA)
		}
		return
	}

	tm := (*C.dpi_time_t)(bind.dataPtr)
	tm.hour = C.udint2(t.Hour())
	tm.minute = C.udint2(t.Minute())
	tm.second = C.udint2(t.Second())

	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.dpi_time_t{}))
	}
}

func writeNumericValue(val interface{}, bind bindColInfo) {
	// Convert value to string first, then to numeric struct
	var s string
	switch v := val.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case int64:
		s = strconv.FormatInt(v, 10)
	case float64:
		s = strconv.FormatFloat(v, 'f', -1, 64)
	default:
		s = fmt.Sprintf("%v", v)
	}

	num := (*C.dpi_numeric_t)(bind.dataPtr)

	// Parse sign
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}
	if negative {
		num.sign = 0
	} else {
		num.sign = 1
	}

	// Parse integer and decimal parts
	parts := strings.SplitN(s, ".", 2)
	intPart := parts[0]
	decPart := ""
	if len(parts) > 1 {
		decPart = parts[1]
	}

	// Set precision and scale
	num.precision = C.udbyte(len(intPart) + len(decPart))
	num.scale = C.sdbyte(len(decPart))

	// Convert to big integer representation
	fullDigits := intPart + decPart
	bigVal := new(big.Int)
	bigVal.SetString(fullDigits, 10)

	// Store bytes in little-endian format
	bBytes := bigVal.Bytes()
	for i := 0; i < 16; i++ {
		num.val[i] = 0
	}
	// Reverse to little-endian
	for i, j := 0, len(bBytes)-1; i <= j; i, j = i+1, j-1 {
		bBytes[i], bBytes[j] = bBytes[j], bBytes[i]
	}
	for i := 0; i < len(bBytes) && i < 16; i++ {
		num.val[i] = C.udbyte(bBytes[i])
	}

	if bind.indPtr != nil {
		*bind.indPtr = C.slength(unsafe.Sizeof(C.dpi_numeric_t{}))
	}
	if bind.actLenPtr != nil {
		*bind.actLenPtr = C.slength(unsafe.Sizeof(C.dpi_numeric_t{}))
	}
}

// extractBoundValue reads a value from a parameter binding buffer.
func extractBoundValue(bind bindParamInfo) interface{} {
	if bind.indPtr != nil && *bind.indPtr == C.slength(DSQL_NULL_DATA) {
		return nil
	}
	if bind.dataPtr == nil {
		return nil
	}

	switch bind.cType {
	case DSQL_C_NCHAR, DSQL_C_CHAR, DSQL_C_WCHAR:
		if bind.indPtr != nil && *bind.indPtr >= 0 {
			return C.GoStringN((*C.char)(bind.dataPtr), C.int(*bind.indPtr))
		}
		return C.GoString((*C.char)(bind.dataPtr))
	case DSQL_C_SLONG:
		return int64(*(*C.sdint4)(bind.dataPtr))
	case DSQL_C_ULONG:
		return int64(*(*C.udint4)(bind.dataPtr))
	case DSQL_C_SSHORT:
		return int64(*(*C.sdint2)(bind.dataPtr))
	case DSQL_C_USHORT:
		return int64(*(*C.udint2)(bind.dataPtr))
	case DSQL_C_SBIGINT:
		return int64(*(*C.sdint8)(bind.dataPtr))
	case DSQL_C_UBIGINT:
		return int64(*(*C.udint8)(bind.dataPtr))
	case DSQL_C_FLOAT:
		return float64(*(*C.float)(bind.dataPtr))
	case DSQL_C_DOUBLE:
		return float64(*(*C.double)(bind.dataPtr))
	case DSQL_C_STINYINT:
		return int64(*(*C.sdbyte)(bind.dataPtr))
	case DSQL_C_UTINYINT:
		return int64(*(*C.udbyte)(bind.dataPtr))
	case DSQL_C_BIT:
		return int64(*(*C.sdint4)(bind.dataPtr))
	case DSQL_C_BINARY:
		length := int64(bind.bufLen)
		if bind.indPtr != nil && *bind.indPtr >= 0 {
			length = int64(*bind.indPtr)
		}
		return C.GoBytes(bind.dataPtr, C.int(length))
	case DSQL_C_TIMESTAMP:
		ts := (*C.dpi_timestamp_t)(bind.dataPtr)
		return time.Date(
			int(ts.year), time.Month(ts.month), int(ts.day),
			int(ts.hour), int(ts.minute), int(ts.second),
			int(ts.fraction), time.Local,
		)
	case DSQL_C_DATE:
		d := (*C.dpi_date_t)(bind.dataPtr)
		return time.Date(int(d.year), time.Month(d.month), int(d.day), 0, 0, 0, 0, time.Local)
	case DSQL_C_TIME:
		t := (*C.dpi_time_t)(bind.dataPtr)
		return time.Date(0, 1, 1, int(t.hour), int(t.minute), int(t.second), 0, time.Local)
	case DSQL_C_NUMERIC:
		num := (*C.dpi_numeric_t)(bind.dataPtr)
		return numericToString(num)
	default:
		// Treat as string
		if bind.indPtr != nil && *bind.indPtr >= 0 {
			return C.GoStringN((*C.char)(bind.dataPtr), C.int(*bind.indPtr))
		}
		return C.GoString((*C.char)(bind.dataPtr))
	}
}

func numericToString(num *C.dpi_numeric_t) string {
	// Read the val bytes into a big.Int (little-endian)
	b := make([]byte, 16)
	for i := 0; i < 16; i++ {
		b[i] = byte(num.val[i])
	}
	// Reverse to big-endian
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	// Trim leading zeros
	start := 0
	for start < len(b)-1 && b[start] == 0 {
		start++
	}
	b = b[start:]

	bigVal := new(big.Int)
	bigVal.SetBytes(b)

	s := bigVal.String()
	scale := int(num.scale)
	if scale > 0 {
		if len(s) <= scale {
			s = strings.Repeat("0", scale-len(s)+1) + s
		}
		s = s[:len(s)-scale] + "." + s[len(s)-scale:]
	}

	if num.sign == 0 {
		s = "-" + s
	}
	return s
}

// Unused but keeping for potential future use
var _ = binary.LittleEndian
var _ = math.Float64frombits
