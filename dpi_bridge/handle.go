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
typedef dhandle         dhobjdesc;
typedef dhandle         dhobj;
typedef dhandle         dhbfile;
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

// Handle pool: maps uintptr IDs to Go objects.
// C code holds these IDs as void* pointers.
// This avoids passing actual Go pointers to C (which the GC could move).

var (
	handleMu   sync.RWMutex
	handles    = make(map[uintptr]interface{})
	nextHandle uintptr
)

// allocHandle stores a Go object and returns a handle ID (as uintptr).
func allocHandle(obj interface{}) uintptr {
	handleMu.Lock()
	defer handleMu.Unlock()
	nextHandle++
	handles[nextHandle] = obj
	return nextHandle
}

// getHandle retrieves a Go object by handle ID.
func getHandle(id uintptr) (interface{}, bool) {
	handleMu.RLock()
	defer handleMu.RUnlock()
	obj, ok := handles[id]
	return obj, ok
}

// freeHandle removes a handle from the pool.
func freeHandle(id uintptr) bool {
	handleMu.Lock()
	defer handleMu.Unlock()
	if _, ok := handles[id]; ok {
		delete(handles, id)
		return true
	}
	return false
}

// handleToPtr converts a uintptr handle ID to a C void* pointer.
func handleToPtr(id uintptr) unsafe.Pointer {
	return unsafe.Pointer(id)
}

// ptrToHandle converts a C void* pointer back to a uintptr handle ID.
func ptrToHandle(p unsafe.Pointer) uintptr {
	return uintptr(p)
}

// getTypedHandle retrieves a handle and checks its type.
func getEnvHandle(h C.dhenv) (*envHandle, error) {
	id := ptrToHandle(unsafe.Pointer(h))
	obj, ok := getHandle(id)
	if !ok {
		return nil, fmt.Errorf("invalid env handle")
	}
	env, ok := obj.(*envHandle)
	if !ok {
		return nil, fmt.Errorf("handle is not an env handle")
	}
	return env, nil
}

func getConnHandle(h C.dhcon) (*connHandle, error) {
	id := ptrToHandle(unsafe.Pointer(h))
	obj, ok := getHandle(id)
	if !ok {
		return nil, fmt.Errorf("invalid connection handle")
	}
	conn, ok := obj.(*connHandle)
	if !ok {
		return nil, fmt.Errorf("handle is not a connection handle")
	}
	return conn, nil
}

func getStmtHandle(h C.dhstmt) (*stmtHandle, error) {
	id := ptrToHandle(unsafe.Pointer(h))
	obj, ok := getHandle(id)
	if !ok {
		return nil, fmt.Errorf("invalid statement handle")
	}
	stmt, ok := obj.(*stmtHandle)
	if !ok {
		return nil, fmt.Errorf("handle is not a statement handle")
	}
	return stmt, nil
}

func getDescHandle(h C.dhdesc) (*descHandle, error) {
	id := ptrToHandle(unsafe.Pointer(h))
	obj, ok := getHandle(id)
	if !ok {
		return nil, fmt.Errorf("invalid descriptor handle")
	}
	desc, ok := obj.(*descHandle)
	if !ok {
		return nil, fmt.Errorf("handle is not a descriptor handle")
	}
	return desc, nil
}

func getLobHandle(h C.dhloblctr) (*lobHandle, error) {
	id := ptrToHandle(unsafe.Pointer(h))
	obj, ok := getHandle(id)
	if !ok {
		return nil, fmt.Errorf("invalid LOB handle")
	}
	lob, ok := obj.(*lobHandle)
	if !ok {
		return nil, fmt.Errorf("handle is not a LOB handle")
	}
	return lob, nil
}
