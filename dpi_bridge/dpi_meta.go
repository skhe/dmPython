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
	"fmt"
	"unsafe"
)

// Catalog functions that execute SQL queries and return result sets.

func goStringFromUdbyte(s *C.udbyte, length C.sdint2) string {
	if s == nil || length == 0 {
		return ""
	}
	if length < 0 {
		return C.GoString((*C.char)(unsafe.Pointer(s)))
	}
	return C.GoStringN((*C.char)(unsafe.Pointer(s)), C.int(length))
}

//export dpi_tables
func dpi_tables(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2,
	tabletype *C.udbyte, namelength4 C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	schema := goStringFromUdbyte(schemaname, namelength2)
	table := goStringFromUdbyte(tablename, namelength3)
	ttype := goStringFromUdbyte(tabletype, namelength4)

	sql := "SELECT NULL AS TABLE_CAT, OWNER AS TABLE_SCHEM, TABLE_NAME, TABLE_TYPE, NULL AS REMARKS FROM ALL_TABLES WHERE 1=1"
	if schema != "" && schema != "%" {
		sql += fmt.Sprintf(" AND OWNER = '%s'", schema)
	}
	if table != "" && table != "%" {
		sql += fmt.Sprintf(" AND TABLE_NAME LIKE '%s'", table)
	}
	if ttype != "" && ttype != "%" {
		sql += fmt.Sprintf(" AND TABLE_TYPE = '%s'", ttype)
	}

	return execMetaQuery(stmt, sql)
}

//export dpi_columns
func dpi_columns(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2,
	columnname *C.udbyte, namelength4 C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	schema := goStringFromUdbyte(schemaname, namelength2)
	table := goStringFromUdbyte(tablename, namelength3)
	column := goStringFromUdbyte(columnname, namelength4)

	sql := `SELECT NULL AS TABLE_CAT, OWNER AS TABLE_SCHEM, TABLE_NAME, COLUMN_NAME,
		DATA_TYPE AS DATA_TYPE_CODE, DATA_TYPE AS TYPE_NAME,
		DATA_LENGTH AS COLUMN_SIZE, DATA_LENGTH AS BUFFER_LENGTH,
		DATA_SCALE AS DECIMAL_DIGITS, 10 AS NUM_PREC_RADIX,
		CASE NULLABLE WHEN 'Y' THEN 1 ELSE 0 END AS NULLABLE,
		NULL AS REMARKS, DATA_DEFAULT AS COLUMN_DEF,
		DATA_TYPE AS SQL_DATA_TYPE, NULL AS SQL_DATETIME_SUB,
		DATA_LENGTH AS CHAR_OCTET_LENGTH, COLUMN_ID AS ORDINAL_POSITION,
		NULLABLE AS IS_NULLABLE
		FROM ALL_TAB_COLUMNS WHERE 1=1`

	if schema != "" && schema != "%" {
		sql += fmt.Sprintf(" AND OWNER = '%s'", schema)
	}
	if table != "" && table != "%" {
		sql += fmt.Sprintf(" AND TABLE_NAME LIKE '%s'", table)
	}
	if column != "" && column != "%" {
		sql += fmt.Sprintf(" AND COLUMN_NAME LIKE '%s'", column)
	}
	sql += " ORDER BY TABLE_SCHEM, TABLE_NAME, ORDINAL_POSITION"

	return execMetaQuery(stmt, sql)
}

//export dpi_columns2
func dpi_columns2(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2,
	columnname *C.udbyte, namelength4 C.sdint2) C.DPIRETURN {
	return dpi_columns(hstmt, catalogname, namelength1, schemaname, namelength2, tablename, namelength3, columnname, namelength4)
}

//export dpi_primarykeys
func dpi_primarykeys(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	schema := goStringFromUdbyte(schemaname, namelength2)
	table := goStringFromUdbyte(tablename, namelength3)

	sql := `SELECT NULL AS TABLE_CAT, CON.OWNER AS TABLE_SCHEM, CON.TABLE_NAME,
		COL.COLUMN_NAME, COL.POSITION AS KEY_SEQ, CON.CONSTRAINT_NAME AS PK_NAME
		FROM ALL_CONSTRAINTS CON
		JOIN ALL_CONS_COLUMNS COL ON CON.CONSTRAINT_NAME = COL.CONSTRAINT_NAME AND CON.OWNER = COL.OWNER
		WHERE CON.CONSTRAINT_TYPE = 'P'`

	if schema != "" && schema != "%" {
		sql += fmt.Sprintf(" AND CON.OWNER = '%s'", schema)
	}
	if table != "" && table != "%" {
		sql += fmt.Sprintf(" AND CON.TABLE_NAME = '%s'", table)
	}
	sql += " ORDER BY COL.POSITION"

	return execMetaQuery(stmt, sql)
}

//export dpi_foreignkeys
func dpi_foreignkeys(hstmt C.dhstmt,
	pkCatalog *C.udbyte, pkCatalogLen C.sdint2,
	pkSchema *C.udbyte, pkSchemaLen C.sdint2,
	pkTable *C.udbyte, pkTableLen C.sdint2,
	fkCatalog *C.udbyte, fkCatalogLen C.sdint2,
	fkSchema *C.udbyte, fkSchemaLen C.sdint2,
	fkTable *C.udbyte, fkTableLen C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sql := `SELECT NULL AS PKTABLE_CAT, '' AS PKTABLE_SCHEM, '' AS PKTABLE_NAME,
		'' AS PKCOLUMN_NAME, NULL AS FKTABLE_CAT, '' AS FKTABLE_SCHEM,
		'' AS FKTABLE_NAME, '' AS FKCOLUMN_NAME, 0 AS KEY_SEQ,
		0 AS UPDATE_RULE, 0 AS DELETE_RULE, '' AS FK_NAME, '' AS PK_NAME
		WHERE 1=0` // Stub — returns empty result

	return execMetaQuery(stmt, sql)
}

//export dpi_statistics
func dpi_statistics(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2,
	unique C.udint2, reserved C.udint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	// Return empty result set
	sql := `SELECT NULL AS TABLE_CAT, '' AS TABLE_SCHEM, '' AS TABLE_NAME,
		0 AS NON_UNIQUE, '' AS INDEX_QUALIFIER, '' AS INDEX_NAME,
		0 AS TYPE, 0 AS ORDINAL_POSITION, '' AS COLUMN_NAME,
		'' AS ASC_OR_DESC, 0 AS CARDINALITY, 0 AS PAGES, '' AS FILTER_CONDITION
		WHERE 1=0`

	return execMetaQuery(stmt, sql)
}

//export dpi_specialcolumns
func dpi_specialcolumns(hstmt C.dhstmt, identifiertype C.udint2,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2,
	scope C.udint2, nullable C.udint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sql := `SELECT 0 AS SCOPE, '' AS COLUMN_NAME, 0 AS DATA_TYPE, '' AS TYPE_NAME,
		0 AS COLUMN_SIZE, 0 AS BUFFER_LENGTH, 0 AS DECIMAL_DIGITS, 0 AS PSEUDO_COLUMN
		WHERE 1=0`

	return execMetaQuery(stmt, sql)
}

//export dpi_specialcolumns2
func dpi_specialcolumns2(hstmt C.dhstmt, identifiertype C.udint2,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2,
	scope C.udint2, nullable C.udint2) C.DPIRETURN {
	return dpi_specialcolumns(hstmt, identifiertype, catalogname, namelength1, schemaname, namelength2, tablename, namelength3, scope, nullable)
}

//export dpi_tableprivileges
func dpi_tableprivileges(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sql := `SELECT NULL AS TABLE_CAT, '' AS TABLE_SCHEM, '' AS TABLE_NAME,
		'' AS GRANTOR, '' AS GRANTEE, '' AS PRIVILEGE, '' AS IS_GRANTABLE
		WHERE 1=0`
	return execMetaQuery(stmt, sql)
}

//export dpi_columnprivileges
func dpi_columnprivileges(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	tablename *C.udbyte, namelength3 C.sdint2,
	columnname *C.udbyte, namelength4 C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sql := `SELECT NULL AS TABLE_CAT, '' AS TABLE_SCHEM, '' AS TABLE_NAME,
		'' AS COLUMN_NAME, '' AS GRANTOR, '' AS GRANTEE, '' AS PRIVILEGE, '' AS IS_GRANTABLE
		WHERE 1=0`
	return execMetaQuery(stmt, sql)
}

//export dpi_procedures
func dpi_procedures(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	procname *C.udbyte, namelength3 C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sql := `SELECT NULL AS PROCEDURE_CAT, '' AS PROCEDURE_SCHEM, '' AS PROCEDURE_NAME,
		0 AS NUM_INPUT_PARAMS, 0 AS NUM_OUTPUT_PARAMS, 0 AS NUM_RESULT_SETS,
		'' AS REMARKS, 0 AS PROCEDURE_TYPE
		WHERE 1=0`
	return execMetaQuery(stmt, sql)
}

//export dpi_procedurecolumns
func dpi_procedurecolumns(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	procname *C.udbyte, namelength3 C.sdint2,
	columnname *C.udbyte, namelength4 C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	sql := `SELECT NULL AS PROCEDURE_CAT, '' AS PROCEDURE_SCHEM, '' AS PROCEDURE_NAME,
		'' AS COLUMN_NAME, 0 AS COLUMN_TYPE, 0 AS DATA_TYPE, '' AS TYPE_NAME,
		0 AS COLUMN_SIZE, 0 AS BUFFER_LENGTH, 0 AS DECIMAL_DIGITS, 0 AS NUM_PREC_RADIX,
		0 AS NULLABLE, '' AS REMARKS
		WHERE 1=0`
	return execMetaQuery(stmt, sql)
}

//export dpi_procedurecolumns2
func dpi_procedurecolumns2(hstmt C.dhstmt,
	catalogname *C.udbyte, namelength1 C.sdint2,
	schemaname *C.udbyte, namelength2 C.sdint2,
	procname *C.udbyte, namelength3 C.sdint2,
	columnname *C.udbyte, namelength4 C.sdint2) C.DPIRETURN {
	return dpi_procedurecolumns(hstmt, catalogname, namelength1, schemaname, namelength2, procname, namelength3, columnname, namelength4)
}

// execMetaQuery executes a metadata SQL query and sets up the result set.
// Caller must hold stmt.mu.Lock().
func execMetaQuery(stmt *stmtHandle, sql string) C.DPIRETURN {
	if stmt.conn.db == nil {
		stmt.lastErr = &diagInfo{errorCode: -1, message: "Not connected"}
		return DSQL_ERROR
	}

	// Close previous result set
	if stmt.rows != nil {
		stmt.rows.Close()
		stmt.rows = nil
	}

	rows, dbErr := stmt.conn.db.Query(sql)
	if dbErr != nil {
		stmt.lastErr = diagFromError(dbErr)
		return DSQL_ERROR
	}
	stmt.rows = rows
	stmt.sql = sql
	populateColumnInfo(stmt)
	if dbErr := cacheAllRows(stmt); dbErr != nil {
		stmt.lastErr = diagFromError(dbErr)
		return DSQL_ERROR
	}
	return DSQL_SUCCESS
}
