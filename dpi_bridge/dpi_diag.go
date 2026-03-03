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
	"strings"
	"unsafe"
)

// getDiagFromHandle retrieves diagnostic info from any handle type.
func getDiagFromHandle(hndlType int16, hndl C.dhandle) *diagInfo {
	id := ptrToHandle(unsafe.Pointer(hndl))
	obj, ok := getHandle(id)
	if !ok {
		return nil
	}

	switch h := obj.(type) {
	case *envHandle:
		return h.lastErr
	case *connHandle:
		return h.lastErr
	case *stmtHandle:
		return h.lastErr
	case *descHandle:
		return h.lastErr
	}
	return nil
}

//export dpi_get_diag_rec
func dpi_get_diag_rec(hndlType C.sdint2, hndl C.dhandle, recNum C.sdint2,
	errCode *C.sdint4, errMsg *C.sdbyte, bufSz C.sdint2, msgLen *C.sdint2) C.DPIRETURN {

	if recNum != 1 {
		return DSQL_NO_DATA
	}

	diag := getDiagFromHandle(int16(hndlType), hndl)
	if diag == nil {
		return DSQL_NO_DATA
	}

	if errCode != nil {
		*errCode = C.sdint4(diag.errorCode)
	}

	if errMsg != nil && bufSz > 0 {
		n := cStringLen(errMsg, int(bufSz), diag.message)
		if msgLen != nil {
			*msgLen = C.sdint2(n)
		}
	} else if msgLen != nil {
		*msgLen = C.sdint2(len(diag.message))
	}

	return DSQL_SUCCESS
}

//export dpi_get_diag_recW
func dpi_get_diag_recW(hndlType C.sdint2, hndl C.dhandle, recNum C.sdint2,
	errCode *C.sdint4, errMsg *C.sdbyte, bufSz C.sdint2, msgLen *C.sdint2) C.DPIRETURN {
	return dpi_get_diag_rec(hndlType, hndl, recNum, errCode, errMsg, bufSz, msgLen)
}

//export dpi_get_diag_field
func dpi_get_diag_field(hndlType C.sdint2, hndl C.dhandle, recNum C.sdint2,
	diagID C.sdint2, diagInfo2 C.dpointer, bufLen C.slength, infoLen *C.slength) C.DPIRETURN {
	field := int16(diagID)

	switch field {
	case DSQL_DIAG_DYNAMIC_FUNCTION_CODE:
		// Determine statement type from the SQL text
		funcCode := int32(0) // INVALID
		id := ptrToHandle(unsafe.Pointer(hndl))
		obj, ok := getHandle(id)
		if ok {
			if stmt, ok := obj.(*stmtHandle); ok {
				stmt.mu.Lock()
				funcCode = inferFuncCode(stmt.sql)
				stmt.mu.Unlock()
			}
		}
		if diagInfo2 != nil {
			*(*C.sdint4)(unsafe.Pointer(diagInfo2)) = C.sdint4(funcCode)
		}
		if infoLen != nil {
			*infoLen = C.slength(unsafe.Sizeof(C.sdint4(0)))
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_ROW_COUNT:
		// Get row count from statement handle
		id := ptrToHandle(unsafe.Pointer(hndl))
		obj, ok := getHandle(id)
		if !ok {
			return DSQL_INVALID_HANDLE
		}
		if stmt, ok := obj.(*stmtHandle); ok {
			stmt.mu.Lock()
			*(*C.sdint8)(diagInfo2) = C.sdint8(stmt.rowsAffected)
			stmt.mu.Unlock()
			if infoLen != nil {
				*infoLen = C.slength(unsafe.Sizeof(C.sdint8(0)))
			}
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_NUMBER:
		diag := getDiagFromHandle(int16(hndlType), hndl)
		count := C.sdint4(0)
		if diag != nil {
			count = 1
		}
		*(*C.sdint4)(diagInfo2) = count
		if infoLen != nil {
			*infoLen = C.slength(unsafe.Sizeof(C.sdint4(0)))
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_ERROR_CODE:
		if recNum < 1 {
			return DSQL_NO_DATA
		}
		diag := getDiagFromHandle(int16(hndlType), hndl)
		if diag == nil {
			return DSQL_NO_DATA
		}
		*(*C.sdint4)(diagInfo2) = C.sdint4(diag.errorCode)
		if infoLen != nil {
			*infoLen = C.slength(unsafe.Sizeof(C.sdint4(0)))
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_MESSAGE_TEXT:
		if recNum < 1 {
			return DSQL_NO_DATA
		}
		diag := getDiagFromHandle(int16(hndlType), hndl)
		if diag == nil {
			return DSQL_NO_DATA
		}
		n := cStringLen((*C.sdbyte)(diagInfo2), int(bufLen), diag.message)
		if infoLen != nil {
			*infoLen = C.slength(n)
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_ROWID:
		// Return empty ROWID
		if diagInfo2 != nil {
			*(*C.sdbyte)(diagInfo2) = 0
		}
		if infoLen != nil {
			*infoLen = 0
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_EXECID:
		// Return 0 exec ID (caller passes udint4*)
		if diagInfo2 != nil {
			*(*C.udint4)(unsafe.Pointer(diagInfo2)) = 0
		}
		if infoLen != nil {
			*infoLen = C.slength(unsafe.Sizeof(C.udint4(0)))
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_EXPLAIN:
		// Return empty explain
		if diagInfo2 != nil && bufLen > 0 {
			*(*C.sdbyte)(diagInfo2) = 0
		}
		if infoLen != nil {
			*infoLen = 0
		}
		return DSQL_SUCCESS

	case DSQL_DIAG_SERVER_STAT:
		if diagInfo2 != nil {
			*(*C.sdint4)(diagInfo2) = 0
		}
		if infoLen != nil {
			*infoLen = C.slength(unsafe.Sizeof(C.sdint4(0)))
		}
		return DSQL_SUCCESS

	default:
		return DSQL_ERROR
	}
}

//export dpi_get_diag_fieldW
func dpi_get_diag_fieldW(hndlType C.sdint2, hndl C.dhandle, recNum C.sdint2,
	diagID C.sdint2, diagInfo2 C.dpointer, bufLen C.slength, infoLen *C.slength) C.DPIRETURN {
	return dpi_get_diag_field(hndlType, hndl, recNum, diagID, diagInfo2, bufLen, infoLen)
}

// inferFuncCode determines the statement type code from the SQL text.
func inferFuncCode(sqlStr string) int32 {
	s := strings.TrimSpace(sqlStr)
	if len(s) == 0 {
		return DSQL_DIAG_FUNC_CODE_INVALID
	}
	// Find the first keyword
	upper := strings.ToUpper(s)
	// Remove leading parentheses
	for strings.HasPrefix(upper, "(") {
		upper = strings.TrimSpace(upper[1:])
	}

	switch {
	case strings.HasPrefix(upper, "SELECT"):
		return DSQL_DIAG_FUNC_CODE_SELECT
	case strings.HasPrefix(upper, "INSERT"):
		return DSQL_DIAG_FUNC_CODE_INSERT
	case strings.HasPrefix(upper, "UPDATE"):
		return DSQL_DIAG_FUNC_CODE_UPDATE
	case strings.HasPrefix(upper, "DELETE"):
		return DSQL_DIAG_FUNC_CODE_DELETE
	case strings.HasPrefix(upper, "MERGE"):
		return DSQL_DIAG_FUNC_CODE_MERGE
	case strings.HasPrefix(upper, "CALL") || strings.HasPrefix(upper, "EXEC"):
		return DSQL_DIAG_FUNC_CODE_CALL
	case strings.HasPrefix(upper, "CREATE TABLE"):
		return DSQL_DIAG_FUNC_CODE_CREATE_TAB
	case strings.HasPrefix(upper, "DROP TABLE"):
		return DSQL_DIAG_FUNC_CODE_DROP_TAB
	case strings.HasPrefix(upper, "SET SCHEMA"):
		return DSQL_DIAG_FUNC_CODE_SET_CURRENT_SCHEMA
	default:
		return DSQL_DIAG_FUNC_CODE_INVALID
	}
}
