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
typedef dhandle         dhloblctr;
*/
import "C"
import (
	"sync"
	"unsafe"
)

// envHandle represents a DPI environment.
type envHandle struct {
	mu        sync.Mutex
	localCode int32 // PG_UTF8 etc.
	langID    int32 // LANGUAGE_CN etc.
	lastErr   *diagInfo
}

// diagInfo stores diagnostic information for error reporting.
type diagInfo struct {
	errorCode int32
	message   string
}

func newEnvHandle() *envHandle {
	return &envHandle{
		localCode: PG_UTF8,
		langID:    LANGUAGE_EN,
	}
}

//export dpi_module_init
func dpi_module_init() C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_module_deinit
func dpi_module_deinit() C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_alloc_env
func dpi_alloc_env(penv *C.dhenv) C.DPIRETURN {
	if penv == nil {
		return DSQL_ERROR
	}
	env := newEnvHandle()
	id := allocHandle(env)
	*penv = C.dhenv(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_free_env
func dpi_free_env(henv C.dhenv) C.DPIRETURN {
	id := ptrToHandle(unsafe.Pointer(henv))
	if freeHandle(id) {
		return DSQL_SUCCESS
	}
	return DSQL_INVALID_HANDLE
}

//export dpi_set_env_attr
func dpi_set_env_attr(henv C.dhenv, attrID C.sdint4, val C.dpointer, valLen C.sdint4) C.DPIRETURN {
	env, err := getEnvHandle(henv)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	env.mu.Lock()
	defer env.mu.Unlock()

	intVal := int32(uintptr(val))

	switch int32(attrID) {
	case DSQL_ATTR_LOCAL_CODE:
		env.localCode = intVal
	case DSQL_ATTR_LANG_ID:
		env.langID = intVal
	default:
		// Unknown attribute — ignore silently
	}
	return DSQL_SUCCESS
}

//export dpi_get_env_attr
func dpi_get_env_attr(henv C.dhenv, attrID C.sdint4, val C.dpointer, bufLen C.sdint4, valLen *C.sdint4) C.DPIRETURN {
	env, err := getEnvHandle(henv)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	env.mu.Lock()
	defer env.mu.Unlock()

	switch int32(attrID) {
	case DSQL_ATTR_LOCAL_CODE:
		*(*C.sdint4)(val) = C.sdint4(env.localCode)
		if valLen != nil {
			*valLen = C.sdint4(unsafe.Sizeof(C.sdint4(0)))
		}
	case DSQL_ATTR_LANG_ID:
		*(*C.sdint4)(val) = C.sdint4(env.langID)
		if valLen != nil {
			*valLen = C.sdint4(unsafe.Sizeof(C.sdint4(0)))
		}
	default:
		return DSQL_ERROR
	}
	return DSQL_SUCCESS
}
