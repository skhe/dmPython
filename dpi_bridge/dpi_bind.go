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

//export dpi_bind_param
func dpi_bind_param(hstmt C.dhstmt, iparam C.udint2, paramType C.sdint2,
	ctype C.sdint2, dtype C.sdint2, precision C.ulength,
	scale C.sdint2, buf C.dpointer, bufLen C.slength, indPtr *C.slength) C.DPIRETURN {

	return dpi_bind_param2(hstmt, iparam, paramType, ctype, dtype, precision, scale, buf, bufLen, indPtr, nil)
}

//export dpi_bind_param2
func dpi_bind_param2(hstmt C.dhstmt, iparam C.udint2, paramType C.sdint2,
	ctype C.sdint2, dtype C.sdint2, precision C.ulength,
	scale C.sdint2, buf C.dpointer, bufLen C.slength,
	indPtr *C.slength, actLenPtr *C.slength) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	idx := int(iparam)
	if idx < 1 {
		stmt.lastErr = &diagInfo{errorCode: -1, message: fmt.Sprintf("Invalid parameter index: %d", iparam)}
		return DSQL_ERROR
	}

	stmt.paramBindings[idx] = bindParamInfo{
		paramType: int16(paramType),
		cType:     int16(ctype),
		sqlType:   int16(dtype),
		precision: uint64(precision),
		scale:     int16(scale),
		dataPtr:   unsafe.Pointer(buf),
		bufLen:    int64(bufLen),
		indPtr:    indPtr,
		actLenPtr: actLenPtr,
	}

	return DSQL_SUCCESS
}

//export dpi_number_params
func dpi_number_params(hstmt C.dhstmt, paramCnt *C.udint2) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	if paramCnt != nil {
		*paramCnt = C.udint2(stmt.paramCount)
	}
	return DSQL_SUCCESS
}

//export dpi_desc_param
func dpi_desc_param(hstmt C.dhstmt, iparam C.udint2,
	sqlType *C.sdint2, prec *C.ulength, scale *C.sdint2, nullable *C.sdint2) C.DPIRETURN {

	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	idx := int(iparam) - 1
	if idx < 0 || idx >= len(stmt.params) {
		stmt.lastErr = &diagInfo{errorCode: -1, message: fmt.Sprintf("Parameter index %d out of range", iparam)}
		return DSQL_ERROR
	}

	p := &stmt.params[idx]
	if sqlType != nil {
		*sqlType = C.sdint2(p.sqlType)
	}
	if prec != nil {
		*prec = C.ulength(p.precision)
	}
	if scale != nil {
		*scale = C.sdint2(p.scale)
	}
	if nullable != nil {
		*nullable = C.sdint2(p.nullable)
	}

	return DSQL_SUCCESS
}

//export dpi_unbind_params
func dpi_unbind_params(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	stmt.paramBindings = make(map[int]bindParamInfo)
	return DSQL_SUCCESS
}

//export dpi_put_data
func dpi_put_data(hstmt C.dhstmt, val C.dpointer, valLen C.slength) C.DPIRETURN {
	// data-at-exec: not fully implemented yet
	return DSQL_SUCCESS
}

//export dpi_param_data
func dpi_param_data(hstmt C.dhstmt, valPtr *C.dpointer) C.DPIRETURN {
	// data-at-exec: not fully implemented yet
	return DSQL_NO_DATA
}

//export dpi_exec_add_batch
func dpi_exec_add_batch(hstmt C.dhstmt) C.DPIRETURN {
	// Batch execution: placeholder
	return DSQL_SUCCESS
}

//export dpi_exec_batch
func dpi_exec_batch(hstmt C.dhstmt) C.DPIRETURN {
	// Batch execution: placeholder
	return DSQL_SUCCESS
}
