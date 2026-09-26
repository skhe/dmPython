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
	"database/sql"
	"fmt"
	"strings"
	"unsafe"

	dm "gitee.com/chunanyong/dm"
)

// objDescHandle represents an object type descriptor.
type objDescHandle struct {
	name    string
	schema  string
	oid     int64
	sqlType int16
	fields  []objFieldDesc
	lastErr *diagInfo
}

type objFieldDesc struct {
	name      string
	schema    string
	typeName  string
	sqlType   int16
	precision int16
	scale     int16
	nested    *objDescHandle
	nestedID  uintptr
}

// objHandle represents an object instance.
type objHandle struct {
	desc    *objDescHandle
	values  []interface{}
	lastErr *diagInfo
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
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	if name == nil {
		conn.lastErr = &diagInfo{errorCode: -1, message: "object type name is required"}
		return DSQL_ERROR
	}
	owner := strings.ToUpper(C.GoString((*C.char)(unsafe.Pointer(schema))))
	if owner == "" {
		owner = strings.ToUpper(conn.user)
	}
	typeName := strings.ToUpper(C.GoString((*C.char)(unsafe.Pointer(name))))
	desc, err := describeObjectType(conn.db, owner, typeName, make(map[string]bool))
	if err != nil {
		conn.lastErr = diagFromError(err)
		return DSQL_ERROR
	}
	id := allocHandle(desc)
	*objDesc = C.dhobjdesc(handleToPtr(id))
	return DSQL_SUCCESS
}

func describeObjectType(db *sql.DB, owner, typeName string, visiting map[string]bool) (*objDescHandle, error) {
	if db == nil {
		return nil, fmt.Errorf("not connected")
	}
	key := owner + "." + typeName
	if visiting[key] {
		return nil, fmt.Errorf("recursive object type %s is not supported", key)
	}
	visiting[key] = true
	defer delete(visiting, key)

	var typeCode string
	var oid int64
	err := db.QueryRow("SELECT TYPECODE, TYPE_OID FROM ALL_TYPES WHERE OWNER=? AND TYPE_NAME=?", owner, typeName).Scan(&typeCode, &oid)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("object type %s is not visible", key)
		}
		return nil, err
	}
	desc := &objDescHandle{name: typeName, schema: owner, oid: oid}
	switch strings.ToUpper(typeCode) {
	case "CLASS":
		desc.sqlType = DSQL_CLASS
	case "RECORD":
		desc.sqlType = DSQL_RECORD
	default:
		return nil, fmt.Errorf("object type %s has unsupported kind %s", key, typeCode)
	}

	rows, err := db.Query("SELECT ATTR_NAME, ATTR_TYPE_OWNER, ATTR_TYPE_NAME, LENGTH, PRECISION, SCALE FROM ALL_TYPE_ATTRS WHERE OWNER=? AND TYPE_NAME=? ORDER BY ATTR_NO", owner, typeName)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var fieldName, fieldType string
		var fieldOwner sql.NullString
		var length, precision, scale sql.NullInt64
		if err = rows.Scan(&fieldName, &fieldOwner, &fieldType, &length, &precision, &scale); err != nil {
			break
		}
		field := objFieldDesc{name: fieldName, typeName: fieldType, precision: int16(length.Int64), scale: int16(scale.Int64)}
		if fieldOwner.Valid && fieldOwner.String != "" && !strings.EqualFold(fieldOwner.String, "NULL") {
			field.schema = fieldOwner.String
		} else {
			field.sqlType = objectScalarType(fieldType)
			if field.sqlType == 0 {
				err = fmt.Errorf("object field %s.%s has unsupported type %s", key, fieldName, fieldType)
				break
			}
			if precision.Valid && precision.Int64 > 0 {
				field.precision = int16(precision.Int64)
			}
		}
		desc.fields = append(desc.fields, field)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range desc.fields {
		field := &desc.fields[i]
		if field.schema == "" {
			continue
		}
		// Some DM8 versions report the referenced type name in ATTR_TYPE_OWNER.
		// Use the containing type's owner when that value is not an actual owner.
		var referencedOID int64
		err = db.QueryRow("SELECT TYPE_OID FROM ALL_TYPES WHERE OWNER=? AND TYPE_NAME=?", field.schema, field.typeName).Scan(&referencedOID)
		if err == sql.ErrNoRows && field.schema != owner {
			field.schema = owner
			err = nil
		}
		if err != nil {
			return nil, err
		}
		var nested *objDescHandle
		nested, err = describeObjectType(db, field.schema, field.typeName, visiting)
		if err != nil {
			return nil, err
		}
		field.nested = nested
		field.sqlType = nested.sqlType
	}
	return desc, nil
}

func (conn *connHandle) columnObjectDesc(typeName string) (*objDescHandle, uintptr, error) {
	typeName = strings.ToUpper(typeName)
	if typeName == "" || objectScalarType(typeName) != 0 {
		return nil, 0, fmt.Errorf("not an object type")
	}
	owner := strings.ToUpper(conn.user)
	var oid int64
	if err := conn.db.QueryRow("SELECT TYPE_OID FROM ALL_TYPES WHERE OWNER=? AND TYPE_NAME=?", owner, typeName).Scan(&oid); err != nil {
		return nil, 0, err
	}
	key := fmt.Sprintf("%s.%s:%d", owner, typeName, oid)
	conn.mu.Lock()
	if desc := conn.objectDescs[key]; desc != nil {
		id := conn.objectDescIDs[key]
		conn.mu.Unlock()
		return desc, id, nil
	}
	conn.mu.Unlock()
	desc, err := describeObjectType(conn.db, owner, typeName, make(map[string]bool))
	if err != nil {
		return nil, 0, err
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if existing := conn.objectDescs[key]; existing != nil {
		return existing, conn.objectDescIDs[key], nil
	}
	id := allocHandle(desc)
	conn.objectDescs[key] = desc
	conn.objectDescIDs[key] = id
	return desc, id, nil
}

func objectScalarType(name string) int16 {
	switch strings.ToUpper(name) {
	case "INTEGER", "INT":
		return DSQL_INT
	case "BIGINT":
		return DSQL_BIGINT
	case "SMALLINT":
		return DSQL_SMALLINT
	case "DECIMAL", "DEC", "NUMBER", "NUMERIC":
		return DSQL_DEC
	case "VARCHAR", "VARCHAR2":
		return DSQL_VARCHAR
	case "CHAR":
		return DSQL_CHAR
	case "DATE":
		return DSQL_DATE
	case "TIME":
		return DSQL_TIME
	case "TIMESTAMP", "DATETIME":
		return DSQL_TIMESTAMP
	case "BLOB":
		return DSQL_BLOB
	case "CLOB":
		return DSQL_CLOB
	default:
		return 0
	}
}

//export dpi_desc_obj2
func dpi_desc_obj2(hcon C.dhcon, schema *C.sdbyte, pkgName *C.sdbyte, name *C.sdbyte, objDesc *C.dhobjdesc) C.DPIRETURN {
	if pkgName != nil && C.GoString((*C.char)(unsafe.Pointer(pkgName))) != "" {
		conn, err := getConnHandle(hcon)
		if err != nil {
			return DSQL_INVALID_HANDLE
		}
		conn.lastErr = &diagInfo{errorCode: -1, message: "package object types are not supported"}
		return DSQL_ERROR
	}
	return dpi_desc_obj(hcon, schema, name, objDesc)
}

//export dpi_free_obj_desc
func dpi_free_obj_desc(objDesc C.dhobjdesc) C.DPIRETURN {
	desc, ok := objectDescriptor(objDesc)
	if !ok {
		return DSQL_INVALID_HANDLE
	}
	for i := range desc.fields {
		if desc.fields[i].nestedID != 0 {
			freeObjectDescriptor(desc.fields[i].nestedID, desc.fields[i].nested)
		}
	}
	id := ptrToHandle(unsafe.Pointer(objDesc))
	freeHandle(id)
	return DSQL_SUCCESS
}

func freeObjectDescriptor(id uintptr, desc *objDescHandle) {
	for i := range desc.fields {
		if desc.fields[i].nestedID != 0 {
			freeObjectDescriptor(desc.fields[i].nestedID, desc.fields[i].nested)
		}
	}
	freeHandle(id)
}

func objectDescriptor(h C.dhobjdesc) (*objDescHandle, bool) {
	value, ok := getHandle(ptrToHandle(unsafe.Pointer(h)))
	desc, typed := value.(*objDescHandle)
	return desc, ok && typed
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
	value, ok := getHandle(ptrToHandle(unsafe.Pointer(hobj)))
	obj, typed := value.(*objHandle)
	desc, descOK := objectDescriptor(hdesc)
	if !ok || !typed || !descOK {
		return DSQL_INVALID_HANDLE
	}
	obj.desc = desc
	obj.values = make([]interface{}, len(desc.fields))
	return DSQL_SUCCESS
}

//export dpi_unbind_obj_desc
func dpi_unbind_obj_desc(hobj C.dhobj) C.DPIRETURN {
	value, ok := getHandle(ptrToHandle(unsafe.Pointer(hobj)))
	obj, typed := value.(*objHandle)
	if !ok || !typed {
		return DSQL_INVALID_HANDLE
	}
	obj.desc = nil
	return DSQL_SUCCESS
}

//export dpi_set_obj_val
func dpi_set_obj_val(hobj C.dhobj, nth C.udint4, ctype C.udint2, val C.dpointer, valLen C.slength) C.DPIRETURN {
	value, ok := getHandle(ptrToHandle(unsafe.Pointer(hobj)))
	obj, typed := value.(*objHandle)
	if !ok || !typed {
		return DSQL_INVALID_HANDLE
	}
	if obj.desc == nil || nth < 1 || int(nth) > len(obj.values) {
		obj.lastErr = &diagInfo{errorCode: -1, message: "object value position is invalid"}
		return DSQL_ERROR
	}
	var valueToSet interface{}
	if valLen != C.slength(DSQL_NULL_DATA) {
		switch int16(ctype) {
		case DSQL_C_CLASS, DSQL_C_RECORD:
			child, childOK := getHandle(ptrToHandle(unsafe.Pointer(val)))
			childObj, typed := child.(*objHandle)
			if !childOK || !typed || childObj.desc == nil {
				obj.lastErr = &diagInfo{errorCode: -1, message: "nested object handle is invalid"}
				return DSQL_ERROR
			}
			valueToSet = append([]interface{}(nil), childObj.values...)
		default:
			indicator := valLen
			valueToSet = extractBoundValue(bindParamInfo{cType: int16(ctype), dataPtr: unsafe.Pointer(val), bufLen: int64(valLen), indPtr: &indicator})
		}
	}
	obj.values[int(nth)-1] = valueToSet
	obj.lastErr = nil
	return DSQL_SUCCESS
}

func (obj *objHandle) driverValue() *dm.DmStruct {
	values := append([]interface{}(nil), obj.values...)
	return dm.NewDmStruct(obj.desc.schema+"."+obj.desc.name, values)
}

func fillObjectHandle(value interface{}, handle unsafe.Pointer, desc *objDescHandle) error {
	if desc == nil {
		return fmt.Errorf("object descriptor is unavailable")
	}
	stored, ok := getHandle(ptrToHandle(handle))
	obj, typed := stored.(*objHandle)
	if !ok || !typed {
		return fmt.Errorf("object handle is invalid")
	}
	var valueObject *dm.DmStruct
	switch v := value.(type) {
	case *dm.DmStruct:
		valueObject = v
	case dm.DmStruct:
		valueObject = &v
	default:
		return fmt.Errorf("expected DM object value, got %T", value)
	}
	values, err := valueObject.GetAttributes()
	if err != nil {
		return err
	}
	if len(values) != len(desc.fields) {
		return fmt.Errorf("object has %d values, descriptor has %d fields", len(values), len(desc.fields))
	}
	obj.desc = desc
	obj.values = values
	return nil
}

//export dpi_get_obj_val
func dpi_get_obj_val(hobj C.dhobj, nth C.udint4, ctype C.udint2, val C.dpointer, bufLen C.udint4, valLen *C.slength) C.DPIRETURN {
	value, ok := getHandle(ptrToHandle(unsafe.Pointer(hobj)))
	obj, typed := value.(*objHandle)
	if !ok || !typed {
		return DSQL_INVALID_HANDLE
	}
	if obj.desc == nil || nth < 1 || int(nth) > len(obj.values) {
		obj.lastErr = &diagInfo{errorCode: -1, message: "object value position is invalid"}
		return DSQL_ERROR
	}
	field := obj.desc.fields[int(nth)-1]
	bind := bindColInfo{cType: int16(ctype), dataPtr: unsafe.Pointer(val), bufLen: int64(bufLen), indPtr: valLen}
	if err := writeValueToBinding(obj.values[int(nth)-1], bind, field.sqlType, field.nested); err != nil {
		obj.lastErr = diagFromError(err)
		return DSQL_ERROR
	}
	obj.lastErr = nil
	return DSQL_SUCCESS
}

//export dpi_get_obj_attr
func dpi_get_obj_attr(hobj C.dhobj, nth C.udint4, attrID C.udint2, buf C.dpointer, bufLen C.udint4, length *C.slength) C.DPIRETURN {
	value, ok := getHandle(ptrToHandle(unsafe.Pointer(hobj)))
	obj, typed := value.(*objHandle)
	if !ok || !typed || obj.desc == nil || attrID != 1 || buf == nil {
		return DSQL_INVALID_HANDLE
	}
	*(*C.udint4)(unsafe.Pointer(buf)) = C.udint4(len(obj.desc.fields))
	if length != nil {
		*length = 4
	}
	return DSQL_SUCCESS
}

//export dpi_get_obj_desc_attr
func dpi_get_obj_desc_attr(objDesc C.dhobjdesc, nth C.udint4, attrID C.udint2, buf C.dpointer, bufLen C.udint4, length *C.slength) C.DPIRETURN {
	desc, ok := objectDescriptor(objDesc)
	if !ok || buf == nil || int(nth) > len(desc.fields) {
		return DSQL_INVALID_HANDLE
	}
	name, schema, sqlType, precision, scale, count := desc.name, desc.schema, desc.sqlType, int16(0), int16(0), len(desc.fields)
	var field *objFieldDesc
	if nth > 0 {
		field = &desc.fields[int(nth)-1]
		name, schema, sqlType, precision, scale, count = field.name, field.schema, field.sqlType, field.precision, field.scale, 0
		if field.nested != nil {
			count = len(field.nested.fields)
		}
	}
	switch int(attrID) {
	case 1: // DSQL_ATTR_OBJ_TYPE
		*(*C.sdint2)(unsafe.Pointer(buf)) = C.sdint2(sqlType)
		setObjectAttrLength(length, 2)
	case 2: // DSQL_ATTR_OBJ_PREC
		*(*C.sdint2)(unsafe.Pointer(buf)) = C.sdint2(precision)
		setObjectAttrLength(length, 2)
	case 3: // DSQL_ATTR_OBJ_SCALE
		*(*C.sdint2)(unsafe.Pointer(buf)) = C.sdint2(scale)
		setObjectAttrLength(length, 2)
	case 4: // DSQL_ATTR_OBJ_DESC
		if field == nil || field.nested == nil {
			return DSQL_ERROR
		}
		if field.nestedID == 0 {
			field.nestedID = allocHandle(field.nested)
		}
		*(*C.dhobjdesc)(unsafe.Pointer(buf)) = C.dhobjdesc(handleToPtr(field.nestedID))
		setObjectAttrLength(length, int(unsafe.Sizeof(uintptr(0))))
	case 5: // DSQL_ATTR_OBJ_FIELD_COUNT
		*(*C.udint4)(unsafe.Pointer(buf)) = C.udint4(count)
		setObjectAttrLength(length, 4)
	case 6: // DSQL_ATTR_OBJ_NAME
		setObjectAttrLength(length, int(cStringLen((*C.sdbyte)(unsafe.Pointer(buf)), int(bufLen), name)))
	case 7: // DSQL_ATTR_OBJ_SCHAME
		setObjectAttrLength(length, int(cStringLen((*C.sdbyte)(unsafe.Pointer(buf)), int(bufLen), schema)))
	default:
		return DSQL_ERROR
	}
	return DSQL_SUCCESS
}

func setObjectAttrLength(length *C.slength, value int) {
	if length != nil {
		*length = C.slength(value)
	}
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
//
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
//
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
//
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
//
//export dpi_set_pos
func dpi_set_pos(hstmt C.dhstmt, rowNum C.ulength, op C.udint2, lockType C.udint2) C.DPIRETURN {
	return DSQL_SUCCESS
}

//export dpi_bulk_operation
func dpi_bulk_operation(hstmt C.dhstmt, op C.udint2) C.DPIRETURN {
	return DSQL_SUCCESS
}

// Connection attribute W variants
//
//export dpi_set_con_attrW
func dpi_set_con_attrW(hcon C.dhcon, attrID C.sdint4, val C.dpointer, valLen C.sdint4) C.DPIRETURN {
	return dpi_set_con_attr(hcon, attrID, val, valLen)
}

//export dpi_get_con_attrW
func dpi_get_con_attrW(hcon C.dhcon, attrID C.sdint4, val C.dpointer, bufLen C.sdint4, valLen *C.sdint4) C.DPIRETURN {
	return dpi_get_con_attr(hcon, attrID, val, bufLen, valLen)
}
