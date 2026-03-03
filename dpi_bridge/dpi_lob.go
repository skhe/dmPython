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
typedef dhandle         dhbfile;
*/
import "C"
import (
	"sync"
	"unsafe"
)

// lobHandle represents a LOB locator.
type lobHandle struct {
	mu      sync.Mutex
	conn    *connHandle
	data    []byte // LOB data buffer
	lobType int    // DSQL_BLOB or DSQL_CLOB
	lastErr *diagInfo
}

//export dpi_alloc_lob_locator
func dpi_alloc_lob_locator(hstmt C.dhstmt, plob *C.dhloblctr) C.DPIRETURN {
	if plob == nil {
		return DSQL_ERROR
	}
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob := &lobHandle{conn: stmt.conn, lobType: DSQL_BLOB}
	id := allocHandle(lob)
	*plob = C.dhloblctr(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_alloc_lob_locator2
func dpi_alloc_lob_locator2(hcon C.dhcon, plob *C.dhloblctr) C.DPIRETURN {
	if plob == nil {
		return DSQL_ERROR
	}
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob := &lobHandle{conn: conn, lobType: DSQL_BLOB}
	id := allocHandle(lob)
	*plob = C.dhloblctr(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_free_lob_locator
func dpi_free_lob_locator(hlob C.dhloblctr) C.DPIRETURN {
	id := ptrToHandle(unsafe.Pointer(hlob))
	freeHandle(id)
	return DSQL_SUCCESS
}

//export dpi_lob_get_length
func dpi_lob_get_length(hlob C.dhloblctr, length *C.slength) C.DPIRETURN {
	lob, err := getLobHandle(hlob)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob.mu.Lock()
	defer lob.mu.Unlock()

	if length != nil {
		*length = C.slength(len(lob.data))
	}
	return DSQL_SUCCESS
}

//export dpi_lob_get_length2
func dpi_lob_get_length2(hlob C.dhloblctr, length *C.sdint8) C.DPIRETURN {
	lob, err := getLobHandle(hlob)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob.mu.Lock()
	defer lob.mu.Unlock()

	if length != nil {
		*length = C.sdint8(len(lob.data))
	}
	return DSQL_SUCCESS
}

//export dpi_lob_read
func dpi_lob_read(hlob C.dhloblctr, startPos C.ulength, ctype C.sdint2,
	dataToRead C.slength, valBuf C.dpointer, bufLen C.slength, dataGet *C.slength) C.DPIRETURN {

	lob, err := getLobHandle(hlob)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob.mu.Lock()
	defer lob.mu.Unlock()

	start := int(startPos) - 1 // 1-based to 0-based
	if start < 0 {
		start = 0
	}
	if start >= len(lob.data) {
		if dataGet != nil {
			*dataGet = 0
		}
		return DSQL_NO_DATA
	}

	toRead := int(dataToRead)
	available := len(lob.data) - start
	if toRead > available {
		toRead = available
	}
	if toRead > int(bufLen) {
		toRead = int(bufLen)
	}

	if toRead > 0 && valBuf != nil {
		C.memcpy(unsafe.Pointer(valBuf), unsafe.Pointer(&lob.data[start]), C.size_t(toRead))
	}

	if dataGet != nil {
		*dataGet = C.slength(toRead)
	}

	return DSQL_SUCCESS
}

//export dpi_lob_read2
func dpi_lob_read2(hlob C.dhloblctr, startPos C.udint8, ctype C.sdint2,
	dataToRead C.slength, valBuf C.dpointer, bufLen C.slength, dataGet *C.slength) C.DPIRETURN {
	return dpi_lob_read(hlob, C.ulength(startPos), ctype, dataToRead, valBuf, bufLen, dataGet)
}

//export dpi_lob_read3
func dpi_lob_read3(hlob C.dhloblctr, startPos C.udint8, ctype C.sdint2,
	dataToRead C.slength, valBuf C.dpointer, bufLen C.slength, dataGet *C.slength, dataGetBytes *C.slength) C.DPIRETURN {
	rt := dpi_lob_read(hlob, C.ulength(startPos), ctype, dataToRead, valBuf, bufLen, dataGet)
	if dataGetBytes != nil && dataGet != nil {
		*dataGetBytes = *dataGet
	}
	return rt
}

//export dpi_lob_write
func dpi_lob_write(hlob C.dhloblctr, startPos C.ulength, ctype C.sdint2,
	val C.dpointer, bytesToWrite C.ulength, dataWrited *C.ulength) C.DPIRETURN {

	lob, err := getLobHandle(hlob)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob.mu.Lock()
	defer lob.mu.Unlock()

	start := int(startPos) - 1
	if start < 0 {
		start = 0
	}
	toWrite := int(bytesToWrite)

	newData := C.GoBytes(unsafe.Pointer(val), C.int(toWrite))

	// Ensure buffer is large enough
	needed := start + toWrite
	if needed > len(lob.data) {
		grown := make([]byte, needed)
		copy(grown, lob.data)
		lob.data = grown
	}
	copy(lob.data[start:], newData)

	if dataWrited != nil {
		*dataWrited = C.ulength(toWrite)
	}

	return DSQL_SUCCESS
}

//export dpi_lob_write2
func dpi_lob_write2(hlob C.dhloblctr, startPos C.udint8, ctype C.sdint2,
	val C.dpointer, bytesToWrite C.ulength, dataWrited *C.ulength) C.DPIRETURN {
	return dpi_lob_write(hlob, C.ulength(startPos), ctype, val, bytesToWrite, dataWrited)
}

//export dpi_lob_truncate
func dpi_lob_truncate(hlob C.dhloblctr, length C.ulength, dataLen *C.ulength) C.DPIRETURN {
	lob, err := getLobHandle(hlob)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob.mu.Lock()
	defer lob.mu.Unlock()

	newLen := int(length)
	if newLen < len(lob.data) {
		lob.data = lob.data[:newLen]
	}

	if dataLen != nil {
		*dataLen = C.ulength(len(lob.data))
	}

	return DSQL_SUCCESS
}

//export dpi_lob_truncate2
func dpi_lob_truncate2(hlob C.dhloblctr, length C.udint8, dataLen *C.udint8) C.DPIRETURN {
	lob, err := getLobHandle(hlob)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	lob.mu.Lock()
	defer lob.mu.Unlock()

	newLen := int(length)
	if newLen < len(lob.data) {
		lob.data = lob.data[:newLen]
	}

	if dataLen != nil {
		*dataLen = C.udint8(len(lob.data))
	}

	return DSQL_SUCCESS
}

// LOB W (wide) variants
//export dpi_lob_readW
func dpi_lob_readW(hlob C.dhloblctr, startPos C.ulength, ctype C.sdint2,
	dataToRead C.slength, valBuf C.dpointer, bufLen C.slength, dataGet *C.slength) C.DPIRETURN {
	return dpi_lob_read(hlob, startPos, ctype, dataToRead, valBuf, bufLen, dataGet)
}

//export dpi_lob_readW2
func dpi_lob_readW2(hlob C.dhloblctr, startPos C.udint8, ctype C.sdint2,
	dataToRead C.slength, valBuf C.dpointer, bufLen C.slength, dataGet *C.slength) C.DPIRETURN {
	return dpi_lob_read2(hlob, startPos, ctype, dataToRead, valBuf, bufLen, dataGet)
}

//export dpi_lob_readW3
func dpi_lob_readW3(hlob C.dhloblctr, startPos C.udint8, ctype C.sdint2,
	dataToRead C.slength, valBuf C.dpointer, bufLen C.slength, dataGet *C.slength, dataGetBytes *C.slength) C.DPIRETURN {
	return dpi_lob_read3(hlob, startPos, ctype, dataToRead, valBuf, bufLen, dataGet, dataGetBytes)
}

//export dpi_lob_writeW
func dpi_lob_writeW(hlob C.dhloblctr, startPos C.ulength, ctype C.sdint2,
	val C.dpointer, bytesToWrite C.ulength, dataWrited *C.ulength) C.DPIRETURN {
	return dpi_lob_write(hlob, startPos, ctype, val, bytesToWrite, dataWrited)
}

//export dpi_lob_writeW2
func dpi_lob_writeW2(hlob C.dhloblctr, startPos C.udint8, ctype C.sdint2,
	val C.dpointer, bytesToWrite C.ulength, dataWrited *C.ulength) C.DPIRETURN {
	return dpi_lob_write2(hlob, startPos, ctype, val, bytesToWrite, dataWrited)
}
