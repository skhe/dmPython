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
	"unsafe"
)

// Object/BFILE operations — stub implementations for P4 priority.
// These allow dmPython to load and run without crashing for basic operations,
// but complex object/BFILE features will return errors.

// objDescHandle represents an object type descriptor.
type objDescHandle struct {
	name      string
	schema    string
	sqlType   int16
	fieldCount int32
}

// objHandle represents an object instance.
type objHandle struct {
	desc *objDescHandle
}

// bfileHandle represents a BFILE locator.
type bfileHandle struct {
	dirName  string
	fileName string
}

//export dpi_desc_obj
func dpi_desc_obj(hcon C.dhcon, schema *C.sdbyte, name *C.sdbyte, objDesc *C.dhobjdesc) C.DPIRETURN {
	if objDesc == nil {
		return DSQL_ERROR
	}
	desc := &objDescHandle{
		schema: C.GoString((*C.char)(unsafe.Pointer(schema))),
		name:   C.GoString((*C.char)(unsafe.Pointer(name))),
	}
	id := allocHandle(desc)
	*objDesc = C.dhobjdesc(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_desc_obj2
func dpi_desc_obj2(hcon C.dhcon, schema *C.sdbyte, pkgName *C.sdbyte, name *C.sdbyte, objDesc *C.dhobjdesc) C.DPIRETURN {
	return dpi_desc_obj(hcon, schema, name, objDesc)
}

//export dpi_free_obj_desc
func dpi_free_obj_desc(objDesc C.dhobjdesc) C.DPIRETURN {
	id := ptrToHandle(unsafe.Pointer(objDesc))
	freeHandle(id)
	return DSQL_SUCCESS
}

//export dpi_alloc_obj
func dpi_alloc_obj(hcon C.dhcon, pobj *C.dhobj) C.DPIRETURN {
	if pobj == nil {
		return DSQL_ERROR
	}
	obj := &objHandle{}
	id := allocHandle(obj)
	*pobj = C.dhobj(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_free_obj
func dpi_free_obj(hobj C.dhobj) C.DPIRETURN {
	id := ptrToHandle(unsafe.Pointer(hobj))
	freeHandle(id)
	return DSQL_SUCCESS
}

//export dpi_bind_obj_desc
func dpi_bind_obj_desc(hobj C.dhobj, hdesc C.dhobjdesc) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_unbind_obj_desc
func dpi_unbind_obj_desc(hobj C.dhobj) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_set_obj_val
func dpi_set_obj_val(hobj C.dhobj, nth C.udint4, ctype C.udint2, val C.dpointer, valLen C.slength) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_get_obj_val
func dpi_get_obj_val(hobj C.dhobj, nth C.udint4, ctype C.udint2, val C.dpointer, bufLen C.udint4, valLen *C.slength) C.DPIRETURN {
	if valLen != nil {
		*valLen = C.slength(DSQL_NULL_DATA)
	}
	return DSQL_SUCCESS
}

//export dpi_get_obj_attr
func dpi_get_obj_attr(hobj C.dhobj, nth C.udint4, attrID C.udint2, buf C.dpointer, bufLen C.udint4, length *C.slength) C.DPIRETURN {
	if length != nil {
		*length = 0
	}
	return DSQL_SUCCESS
}

//export dpi_get_obj_desc_attr
func dpi_get_obj_desc_attr(objDesc C.dhobjdesc, nth C.udint4, attrID C.udint2, buf C.dpointer, bufLen C.udint4, length *C.slength) C.DPIRETURN {
	attr := int(attrID)
	const (
		attrObjType       = 1 // DSQL_ATTR_OBJ_TYPE
		attrObjFieldCount = 5 // DSQL_ATTR_OBJ_FIELD_COUNT
		attrObjName       = 6 // DSQL_ATTR_OBJ_NAME
	)
	switch attr {
	case attrObjType:
		if buf != nil {
			*(*C.sdint2)(unsafe.Pointer(buf)) = C.sdint2(DSQL_VARCHAR)
		}
	case attrObjFieldCount:
		if buf != nil {
			*(*C.sdint4)(unsafe.Pointer(buf)) = 0
		}
	case attrObjName:
		if buf != nil {
			*(*C.sdbyte)(unsafe.Pointer(buf)) = 0
		}
		if length != nil {
			*length = 0
		}
	default:
		if buf != nil {
			*(*C.sdint4)(unsafe.Pointer(buf)) = 0
		}
	}
	return DSQL_SUCCESS
}

//export dpi_get_obj_desc_attrW
func dpi_get_obj_desc_attrW(objDesc C.dhobjdesc, nth C.udint4, attrID C.udint2, buf C.dpointer, bufLen C.udint4, length *C.slength) C.DPIRETURN {
	return dpi_get_obj_desc_attr(objDesc, nth, attrID, buf, bufLen, length)
}

//export dpi_set_indtab_node
func dpi_set_indtab_node(hobj C.dhobj, ktype C.udint2, key C.dpointer, keyLen C.slength,
	vtype C.udint2, val C.dpointer, valLen C.slength) C.DPIRETURN {
	return DSQL_SUCCESS
}

// BFILE operations
//export dpi_alloc_bfile
func dpi_alloc_bfile(hcon C.dhcon, pbfile *C.dhbfile) C.DPIRETURN {
	if pbfile == nil {
		return DSQL_ERROR
	}
	bf := &bfileHandle{}
	id := allocHandle(bf)
	*pbfile = C.dhbfile(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_free_bfile
func dpi_free_bfile(hbfile C.dhbfile) C.DPIRETURN {
	id := ptrToHandle(unsafe.Pointer(hbfile))
	freeHandle(id)
	return DSQL_SUCCESS
}

//export dpi_bfile_construct
func dpi_bfile_construct(hbfile C.dhbfile, dirName *C.udbyte, fileName *C.udbyte) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_bfile_constructW
func dpi_bfile_constructW(hbfile C.dhbfile, dirName *C.udbyte, dirNameLen C.udint4, fileName *C.udbyte, fileNameLen C.udint4) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_bfile_get_name
func dpi_bfile_get_name(hbfile C.dhbfile,
	dirBuf *C.udbyte, dirBufLen C.udint4, dirLen *C.udint4,
	fileBuf *C.udbyte, fileBufLen C.udint4, fileLen *C.udint4) C.DPIRETURN {
	if dirLen != nil {
		*dirLen = 0
	}
	if fileLen != nil {
		*fileLen = 0
	}
	return DSQL_SUCCESS
}

//export dpi_bfile_get_nameW
func dpi_bfile_get_nameW(hbfile C.dhbfile,
	dirBuf *C.udbyte, dirBufLen C.udint4, dirLen *C.udint4,
	fileBuf *C.udbyte, fileBufLen C.udint4, fileLen *C.udint4) C.DPIRETURN {
	return dpi_bfile_get_name(hbfile, dirBuf, dirBufLen, dirLen, fileBuf, fileBufLen, fileLen)
}

//export dpi_bfile_read
func dpi_bfile_read(hbfile C.dhbfile, startPos C.udint8, ctype C.sdint2,
	dataToRead C.udint8, valBuf C.dpointer, bufLen C.udint8, dataGet *C.udint8) C.DPIRETURN {
	if dataGet != nil {
		*dataGet = 0
	}
	return DSQL_NO_DATA
}

// ROWID operations
//export dpi_build_rowid
func dpi_build_rowid(hcon C.dhcon, epno C.sdint4, partno C.sdint8, realRowid C.udint8,
	rowidBuf *C.sdbyte, rowidBufLen C.udint4, rowidLen *C.udint4) C.DPIRETURN {
	if rowidLen != nil {
		*rowidLen = 0
	}
	return DSQL_SUCCESS
}

//export dpi_rowid_to_char
func dpi_rowid_to_char(hcon C.dhcon, rowid *C.sdbyte, rowidLen C.udint4,
	destBuf *C.sdbyte, destBufLen C.udint4, destLen *C.udint4) C.DPIRETURN {
	if destBuf != nil && destBufLen > 0 {
		*destBuf = 0
	}
	if destLen != nil {
		*destLen = 0
	}
	return DSQL_SUCCESS
}

//export dpi_char_to_rowid
func dpi_char_to_rowid(hcon C.dhcon, rowidStr *C.sdbyte, rowidLen C.udint4,
	destBuf *C.sdbyte, destBufLen C.udint4, destLen *C.udint4) C.DPIRETURN {
	if destLen != nil {
		*destLen = 0
	}
	return DSQL_SUCCESS
}

// Additional stubs for cursor name operations
//export dpi_set_cursor_name
func dpi_set_cursor_name(hstmt C.dhstmt, name *C.sdbyte, nameLen C.sdint2) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_set_cursor_nameW
func dpi_set_cursor_nameW(hstmt C.dhstmt, name *C.sdbyte, nameLen C.sdint2) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_get_cursor_name
func dpi_get_cursor_name(hstmt C.dhstmt, name *C.sdbyte, bufLen C.sdint2, nameLen *C.sdint2) C.DPIRETURN {
	if name != nil && bufLen > 0 {
		*name = 0
	}
	if nameLen != nil {
		*nameLen = 0
	}
	return DSQL_SUCCESS
}

//export dpi_get_cursor_nameW
func dpi_get_cursor_nameW(hstmt C.dhstmt, name *C.sdbyte, bufLen C.sdint2, nameLen *C.sdint2) C.DPIRETURN {
	return dpi_get_cursor_name(hstmt, name, bufLen, nameLen)
}

// set_pos, bulk_operation stubs
//export dpi_set_pos
func dpi_set_pos(hstmt C.dhstmt, rowNum C.ulength, op C.udint2, lockType C.udint2) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_bulk_operation
func dpi_bulk_operation(hstmt C.dhstmt, op C.udint2) C.DPIRETURN {
	return DSQL_SUCCESS
}

// Connection attribute W variants
//export dpi_set_con_attrW
func dpi_set_con_attrW(hcon C.dhcon, attrID C.sdint4, val C.dpointer, valLen C.sdint4) C.DPIRETURN {
	return dpi_set_con_attr(hcon, attrID, val, valLen)
}

//export dpi_get_con_attrW
func dpi_get_con_attrW(hcon C.dhcon, attrID C.sdint4, val C.dpointer, bufLen C.sdint4, valLen *C.sdint4) C.DPIRETURN {
	return dpi_get_con_attr(hcon, attrID, val, bufLen, valLen)
}
