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
	"unsafe"
)

//export dpi_get_desc_field
func dpi_get_desc_field(hdesc C.dhdesc, recNum C.udint2, field C.sdint2,
	val C.dpointer, valLen C.sdint4, strLen *C.sdint4) C.DPIRETURN {

	desc, err := getDescHandle(hdesc)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}

	if desc.stmt == nil {
		return DSQL_ERROR
	}

	desc.stmt.mu.Lock()
	defer desc.stmt.mu.Unlock()

	fieldID := int16(field)
	idx := int(recNum) - 1 // 1-based

	if desc.descType == 0 {
		// Row descriptor
		return getRowDescField(desc.stmt, idx, fieldID, val, valLen, strLen)
	}
	// Param descriptor
	return getParamDescField(desc.stmt, idx, fieldID, val, valLen, strLen)
}

//export dpi_get_desc_fieldW
func dpi_get_desc_fieldW(hdesc C.dhdesc, recNum C.udint2, field C.sdint2,
	val C.dpointer, valLen C.sdint4, strLen *C.sdint4) C.DPIRETURN {
	return dpi_get_desc_field(hdesc, recNum, field, val, valLen, strLen)
}

//export dpi_set_desc_field
func dpi_set_desc_field(hdesc C.dhdesc, recNum C.udint2, field C.sdint2,
	val C.dpointer, valLen C.sdint4) C.DPIRETURN {
	// Accept but do nothing for now
	return DSQL_SUCCESS
}

//export dpi_set_desc_fieldW
func dpi_set_desc_fieldW(hdesc C.dhdesc, recNum C.udint2, field C.sdint2,
	val C.dpointer, valLen C.sdint4) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_get_desc_rec
func dpi_get_desc_rec(hdesc C.dhdesc, recNum C.udint2,
	nameBuf *C.sdbyte, nameBufLen C.sdint2, nameLen *C.sdint2,
	typePtr *C.sdint2, subType *C.sdint2, length *C.slength,
	prec *C.sdint2, scale *C.sdint2, nullable *C.sdint2) C.DPIRETURN {

	desc, err := getDescHandle(hdesc)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	if desc.stmt == nil {
		return DSQL_ERROR
	}

	desc.stmt.mu.Lock()
	defer desc.stmt.mu.Unlock()

	idx := int(recNum) - 1

	if desc.descType == 0 {
		// Row descriptor
		if idx < 0 || idx >= len(desc.stmt.columns) {
			return DSQL_ERROR
		}
		col := &desc.stmt.columns[idx]
		if nameBuf != nil {
			n := cStringLen(nameBuf, int(nameBufLen), col.name)
			if nameLen != nil {
				*nameLen = C.sdint2(n)
			}
		}
		if typePtr != nil {
			*typePtr = C.sdint2(col.sqlType)
		}
		if subType != nil {
			*subType = 0
		}
		if length != nil {
			*length = C.slength(col.precision)
		}
		if prec != nil {
			*prec = C.sdint2(col.precision)
		}
		if scale != nil {
			*scale = C.sdint2(col.scale)
		}
		if nullable != nil {
			*nullable = C.sdint2(col.nullable)
		}
	} else {
		// Param descriptor
		if idx < 0 || idx >= len(desc.stmt.params) {
			return DSQL_ERROR
		}
		p := &desc.stmt.params[idx]
		if nameBuf != nil {
			n := cStringLen(nameBuf, int(nameBufLen), p.name)
			if nameLen != nil {
				*nameLen = C.sdint2(n)
			}
		}
		if typePtr != nil {
			*typePtr = C.sdint2(p.sqlType)
		}
		if subType != nil {
			*subType = 0
		}
		if length != nil {
			*length = C.slength(p.precision)
		}
		if prec != nil {
			*prec = C.sdint2(p.precision)
		}
		if scale != nil {
			*scale = C.sdint2(p.scale)
		}
		if nullable != nil {
			*nullable = C.sdint2(p.nullable)
		}
	}

	return DSQL_SUCCESS
}

//export dpi_get_desc_recW
func dpi_get_desc_recW(hdesc C.dhdesc, recNum C.udint2,
	nameBuf *C.sdbyte, nameBufLen C.sdint2, nameLen *C.sdint2,
	typePtr *C.sdint2, subType *C.sdint2, length *C.slength,
	prec *C.sdint2, scale *C.sdint2, nullable *C.sdint2) C.DPIRETURN {
	return dpi_get_desc_rec(hdesc, recNum, nameBuf, nameBufLen, nameLen, typePtr, subType, length, prec, scale, nullable)
}

//export dpi_set_desc_rec
func dpi_set_desc_rec(hdesc C.dhdesc, recNum C.udint2, typeVal C.sdint2, subType C.sdint2,
	length C.slength, prec C.sdint2, scale C.sdint2,
	dataPtr C.dpointer, strLen *C.slength, indPtr *C.slength) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_copy_desc
func dpi_copy_desc(srcDesc C.dhdesc, targetDesc C.dhdesc) C.DPIRETURN {
	return DSQL_SUCCESS
}

func getRowDescField(stmt *stmtHandle, idx int, fieldID int16, val C.dpointer, valLen C.sdint4, strLen *C.sdint4) C.DPIRETURN {
	switch fieldID {
	case DSQL_DESC_COUNT:
		*(*C.sdint2)(val) = C.sdint2(stmt.columnCount)
		return DSQL_SUCCESS

	case DSQL_DESC_DISPLAY_SIZE:
		if idx < 0 || idx >= len(stmt.columns) {
			return DSQL_ERROR
		}
		*(*C.slength)(val) = C.slength(stmt.columns[idx].displaySize)
		return DSQL_SUCCESS

	case DSQL_DESC_TYPE:
		if idx < 0 || idx >= len(stmt.columns) {
			return DSQL_ERROR
		}
		*(*C.sdint2)(val) = C.sdint2(stmt.columns[idx].sqlType)
		return DSQL_SUCCESS

	case DSQL_DESC_LENGTH:
		if idx < 0 || idx >= len(stmt.columns) {
			return DSQL_ERROR
		}
		*(*C.ulength)(val) = C.ulength(stmt.columns[idx].precision)
		return DSQL_SUCCESS

	case DSQL_DESC_PRECISION:
		if idx < 0 || idx >= len(stmt.columns) {
			return DSQL_ERROR
		}
		*(*C.sdint2)(val) = C.sdint2(stmt.columns[idx].precision)
		return DSQL_SUCCESS

	case DSQL_DESC_SCALE:
		if idx < 0 || idx >= len(stmt.columns) {
			return DSQL_ERROR
		}
		*(*C.sdint2)(val) = C.sdint2(stmt.columns[idx].scale)
		return DSQL_SUCCESS

	case DSQL_DESC_NULLABLE:
		if idx < 0 || idx >= len(stmt.columns) {
			return DSQL_ERROR
		}
		*(*C.sdint2)(val) = C.sdint2(stmt.columns[idx].nullable)
		return DSQL_SUCCESS

	case DSQL_DESC_NAME:
		if idx < 0 || idx >= len(stmt.columns) {
			return DSQL_ERROR
		}
		n := cStringLen((*C.sdbyte)(val), int(valLen), stmt.columns[idx].name)
		if strLen != nil {
			*strLen = C.sdint4(n)
		}
		return DSQL_SUCCESS

	case DSQL_DESC_OBJ_DESCRIPTOR:
		// Return nil object descriptor
		*(*C.dpointer)(val) = nil
		return DSQL_SUCCESS

	default:
		// Return zero for unknown fields
		*(*C.sdint4)(val) = 0
		if strLen != nil {
			*strLen = C.sdint4(unsafe.Sizeof(C.sdint4(0)))
		}
		return DSQL_SUCCESS
	}
}

func getParamDescField(stmt *stmtHandle, idx int, fieldID int16, val C.dpointer, valLen C.sdint4, strLen *C.sdint4) C.DPIRETURN {
	switch fieldID {
	case DSQL_DESC_COUNT:
		*(*C.sdint2)(val) = C.sdint2(stmt.paramCount)
		return DSQL_SUCCESS

	case DSQL_DESC_PARAMETER_TYPE:
		if idx < 0 || idx >= len(stmt.params) {
			// Default to INPUT
			*(*C.sdint2)(val) = C.sdint2(DSQL_PARAM_INPUT)
			return DSQL_SUCCESS
		}
		*(*C.sdint2)(val) = C.sdint2(stmt.params[idx].paramType)
		return DSQL_SUCCESS

	case DSQL_DESC_TYPE:
		if idx < 0 || idx >= len(stmt.params) {
			*(*C.sdint2)(val) = C.sdint2(DSQL_VARCHAR)
			return DSQL_SUCCESS
		}
		*(*C.sdint2)(val) = C.sdint2(stmt.params[idx].sqlType)
		return DSQL_SUCCESS

	case DSQL_DESC_NAME:
		if idx < 0 || idx >= len(stmt.params) {
			if val != nil {
				*(*C.sdbyte)(val) = 0
			}
			if strLen != nil {
				*strLen = 0
			}
			return DSQL_SUCCESS
		}
		n := cStringLen((*C.sdbyte)(val), int(valLen), stmt.params[idx].name)
		if strLen != nil {
			*strLen = C.sdint4(n)
		}
		return DSQL_SUCCESS

	case DSQL_DESC_BIND_PARAMETER_TYPE:
		// Default to INPUT
		*(*C.sdint2)(val) = C.sdint2(DSQL_PARAM_INPUT)
		if idx >= 0 && idx < len(stmt.params) {
			*(*C.sdint2)(val) = C.sdint2(stmt.params[idx].paramType)
		}
		return DSQL_SUCCESS

	case DSQL_DESC_OBJ_DESCRIPTOR:
		*(*C.dpointer)(val) = nil
		return DSQL_SUCCESS

	default:
		*(*C.sdint4)(val) = 0
		return DSQL_SUCCESS
	}
}
