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
typedef dhandle         dhobj;
typedef dhandle         dhobjdesc;
typedef dhandle         dhbfile;
*/
import "C"
import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	dm "gitee.com/chunanyong/dm"
)

// connHandle represents a DPI connection.
type connHandle struct {
	mu   sync.Mutex
	env  *envHandle
	conn *dm.DmConnection // actual Go driver connection
	db   *sql.DB          // holds the sql.DB for lifecycle management
	tx   driver.Tx        // active transaction (nil if none)

	// Connection parameters (set before login)
	host       string
	port       int
	user       string
	password   string
	schema     string
	autocommit bool
	loginTimeout int
	connTimeout  int
	txnIsolation int

	// Post-login info
	serverVersion string
	serverCode    int32

	// Diagnostics
	lastErr *diagInfo
}

func newConnHandle(env *envHandle) *connHandle {
	return &connHandle{
		env:        env,
		port:       DSQL_DEAFAULT_TCPIP_PORT,
		autocommit: false,
		serverCode: PG_UTF8,
	}
}

//export dpi_alloc_con
func dpi_alloc_con(henv C.dhenv, pcon *C.dhcon) C.DPIRETURN {
	if pcon == nil {
		return DSQL_ERROR
	}
	env, err := getEnvHandle(henv)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn := newConnHandle(env)
	id := allocHandle(conn)
	*pcon = C.dhcon(handleToPtr(id))
	return DSQL_SUCCESS
}

//export dpi_free_con
func dpi_free_con(hcon C.dhcon) C.DPIRETURN {
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn.mu.Lock()
	if conn.db != nil {
		conn.db.Close()
		conn.db = nil
		conn.conn = nil
	}
	conn.mu.Unlock()

	id := ptrToHandle(unsafe.Pointer(hcon))
	freeHandle(id)
	return DSQL_SUCCESS
}

//export dpi_set_con_attr
func dpi_set_con_attr(hcon C.dhcon, attrID C.sdint4, val C.dpointer, valLen C.sdint4) C.DPIRETURN {
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()

	attr := int32(attrID)
	intVal := int(uintptr(val))

	switch attr {
	case DSQL_ATTR_LOGIN_PORT:
		conn.port = intVal
	case DSQL_ATTR_AUTOCOMMIT:
		conn.autocommit = (intVal != 0)
		// If already connected, apply autocommit
		if conn.conn != nil {
			conn.conn.Exec("SET TRANSACTION AUTOCOMMIT " + map[bool]string{true: "ON", false: "OFF"}[conn.autocommit], nil)
		}
	case DSQL_ATTR_LOGIN_TIMEOUT:
		conn.loginTimeout = intVal
	case DSQL_ATTR_CONNECTION_TIMEOUT:
		conn.connTimeout = intVal
	case DSQL_ATTR_TXN_ISOLATION:
		conn.txnIsolation = intVal
	case DSQL_ATTR_LOGIN_SERVER:
		if valLen > 0 {
			conn.host = C.GoStringN((*C.char)(val), C.int(valLen))
		} else {
			conn.host = C.GoString((*C.char)(val))
		}
	case DSQL_ATTR_LOGIN_USER:
		if valLen > 0 {
			conn.user = C.GoStringN((*C.char)(val), C.int(valLen))
		} else {
			conn.user = C.GoString((*C.char)(val))
		}
	case DSQL_ATTR_CURRENT_SCHEMA:
		if valLen > 0 {
			conn.schema = C.GoStringN((*C.char)(val), C.int(valLen))
		} else {
			conn.schema = C.GoString((*C.char)(val))
		}
	case DSQL_ATTR_APP_NAME,
		DSQL_ATTR_SSL_PATH, DSQL_ATTR_SSL_PWD,
		DSQL_ATTR_UKEY_NAME, DSQL_ATTR_UKEY_PIN,
		DSQL_ATTR_COMPRESS_MSG, DSQL_ATTR_USE_STMT_POOL,
		DSQL_ATTR_MPP_LOGIN, DSQL_ATTR_RWSEPARATE,
		DSQL_ATTR_RWSEPARATE_PERCENT, DSQL_ATTR_CURSOR_ROLLBACK_BEHAVIOR,
		DSQL_ATTR_OSAUTH_TYPE, DSQL_ATTR_DDL_AUTOCOMMIT,
		DSQL_ATTR_COMPATIBLE_MODE, DSQL_ATTR_SHAKE_CRYPTO,
		DSQL_ATTR_NLS_NUMERIC_CHARACTERS, DSQL_ATTR_DM_SVC_PATH,
		DSQL_ATTR_ACCESS_MODE, DSQL_ATTR_PACKET_SIZE,
		DSQL_ATTR_CURRENT_CATALOG:
		// Accept but ignore these for now
	default:
		// Unknown attribute — ignore silently
	}
	return DSQL_SUCCESS
}

//export dpi_get_con_attr
func dpi_get_con_attr(hcon C.dhcon, attrID C.sdint4, val C.dpointer, bufLen C.sdint4, valLen *C.sdint4) C.DPIRETURN {
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()

	attr := int32(attrID)

	switch attr {
	case DSQL_ATTR_LOCAL_CODE:
		*(*C.sdint4)(val) = C.sdint4(conn.env.localCode)
		if valLen != nil {
			*valLen = 4
		}
	case DSQL_ATTR_LANG_ID:
		*(*C.sdint4)(val) = C.sdint4(conn.env.langID)
		if valLen != nil {
			*valLen = 4
		}
	case DSQL_ATTR_SERVER_CODE:
		*(*C.sdint4)(val) = C.sdint4(conn.serverCode)
		if valLen != nil {
			*valLen = 4
		}
	case DSQL_ATTR_AUTOCOMMIT:
		v := C.udint4(0)
		if conn.autocommit {
			v = 1
		}
		*(*C.udint4)(val) = v
		if valLen != nil {
			*valLen = 4
		}
	case DSQL_ATTR_TXN_ISOLATION:
		*(*C.sdint4)(val) = C.sdint4(conn.txnIsolation)
		if valLen != nil {
			*valLen = 4
		}
	case DSQL_ATTR_CONNECTION_DEAD:
		dead := C.sdint4(0) // DSQL_CD_FALSE
		if conn.conn == nil {
			dead = 1 // DSQL_CD_TRUE
		} else if conn.db != nil {
			if err := conn.db.Ping(); err != nil {
				dead = 1
			}
		}
		*(*C.sdint4)(val) = dead
		if valLen != nil {
			*valLen = 4
		}
	case DSQL_ATTR_SERVER_VERSION:
		ver := conn.serverVersion
		if ver == "" {
			ver = "DM Database Server 64 V8"
		}
		n := cStringLen((*C.sdbyte)(val), int(bufLen), ver)
		if valLen != nil {
			*valLen = C.sdint4(n)
		}
	case DSQL_ATTR_INSTANCE_NAME:
		name := "DAMENG"
		n := cStringLen((*C.sdbyte)(val), int(bufLen), name)
		if valLen != nil {
			*valLen = C.sdint4(n)
		}
	case DSQL_ATTR_CURRENT_SCHEMA:
		schema := conn.schema
		if schema == "" {
			schema = strings.ToUpper(conn.user)
		}
		n := cStringLen((*C.sdbyte)(val), int(bufLen), schema)
		if valLen != nil {
			*valLen = C.sdint4(n)
		}
	case DSQL_ATTR_TRX_STATE:
		// 0 = complete, 1 = active
		state := C.sdint4(0)
		if conn.tx != nil {
			state = 1
		}
		*(*C.sdint4)(val) = state
		if valLen != nil {
			*valLen = 4
		}
	default:
		// Return zero/empty for unknown attributes
		*(*C.sdint4)(val) = 0
		if valLen != nil {
			*valLen = 4
		}
	}
	return DSQL_SUCCESS
}

//export dpi_login
func dpi_login(hcon C.dhcon, svr *C.sdbyte, user *C.sdbyte, pwd *C.sdbyte) C.DPIRETURN {
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()

	server := C.GoString((*C.char)(unsafe.Pointer(svr)))
	username := C.GoString((*C.char)(unsafe.Pointer(user)))
	password := C.GoString((*C.char)(unsafe.Pointer(pwd)))

	// Parse server string: may be "host:port" or "host" or "host/catalog"
	host := server
	port := conn.port
	catalog := ""

	// Check for catalog (host/catalog format)
	if idx := strings.Index(host, "/"); idx >= 0 {
		catalog = host[idx+1:]
		host = host[:idx]
	}

	// Check for port in host
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		if p, err := strconv.Atoi(host[idx+1:]); err == nil {
			port = p
			host = host[:idx]
		}
	}

	if host == "" {
		host = "localhost"
	}

	conn.user = username
	conn.password = password
	conn.host = host
	conn.port = port

	// Build DSN for Go driver
	dsn := fmt.Sprintf("dm://%s:%s@%s:%d",
		username, password, host, port)
	if catalog != "" {
		dsn += "/" + catalog
	}

	// Set autoCommit in query params
	params := []string{}
	if conn.autocommit {
		params = append(params, "autoCommit=true")
	} else {
		params = append(params, "autoCommit=false")
	}
	if conn.loginTimeout > 0 {
		params = append(params, fmt.Sprintf("loginTimeout=%d", conn.loginTimeout))
	}
	if conn.connTimeout > 0 {
		params = append(params, fmt.Sprintf("socketTimeout=%d", conn.connTimeout*1000))
	}
	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}

	db, dbErr := sql.Open("dm", dsn)
	if dbErr != nil {
		conn.lastErr = &diagInfo{
			errorCode: -1,
			message:   fmt.Sprintf("Failed to open connection: %v", dbErr),
		}
		return DSQL_ERROR
	}

	// Force a real connection
	ctx := context.Background()
	rawConn, dbErr := db.Conn(ctx)
	if dbErr != nil {
		db.Close()
		conn.lastErr = &diagInfo{
			errorCode: -1,
			message:   fmt.Sprintf("Failed to connect: %v", dbErr),
		}
		return DSQL_ERROR
	}

	// Extract the underlying DmConnection
	var dmConn *dm.DmConnection
	dbErr = rawConn.Raw(func(driverConn interface{}) error {
		var ok bool
		dmConn, ok = driverConn.(*dm.DmConnection)
		if !ok {
			return fmt.Errorf("unexpected driver connection type: %T", driverConn)
		}
		return nil
	})
	rawConn.Close()
	if dbErr != nil {
		db.Close()
		conn.lastErr = &diagInfo{
			errorCode: -1,
			message:   fmt.Sprintf("Failed to get DM connection: %v", dbErr),
		}
		return DSQL_ERROR
	}

	conn.db = db
	conn.conn = dmConn

	// Try to get server version
	var version string
	row := db.QueryRow("SELECT BANNER FROM V$VERSION")
	if row.Scan(&version) == nil {
		conn.serverVersion = version
	}

	// Get server encoding
	var serverCode int32
	row = db.QueryRow("SELECT UNICODE")
	if row.Scan(&serverCode) == nil {
		if serverCode == 1 {
			conn.serverCode = PG_UTF8
		}
	}

	return DSQL_SUCCESS
}

//export dpi_loginW
func dpi_loginW(hcon C.dhcon, svr *C.sdbyte, user *C.sdbyte, pwd *C.sdbyte) C.DPIRETURN {
	// For now, treat W variant the same as non-W since we handle UTF-8 throughout
	return dpi_login(hcon, svr, user, pwd)
}

//export dpi_logout
func dpi_logout(hcon C.dhcon) C.DPIRETURN {
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.tx != nil {
		conn.tx.Rollback()
		conn.tx = nil
	}
	if conn.db != nil {
		conn.db.Close()
		conn.db = nil
		conn.conn = nil
	}
	return DSQL_SUCCESS
}

//export dpi_commit
func dpi_commit(hcon C.dhcon) C.DPIRETURN {
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.db == nil {
		conn.lastErr = &diagInfo{errorCode: -1, message: "Not connected"}
		return DSQL_ERROR
	}

	_, dbErr := conn.db.Exec("COMMIT")
	if dbErr != nil {
		conn.lastErr = &diagInfo{
			errorCode: -1,
			message:   fmt.Sprintf("Commit failed: %v", dbErr),
		}
		return DSQL_ERROR
	}
	conn.tx = nil
	return DSQL_SUCCESS
}

//export dpi_rollback
func dpi_rollback(hcon C.dhcon) C.DPIRETURN {
	conn, err := getConnHandle(hcon)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.db == nil {
		conn.lastErr = &diagInfo{errorCode: -1, message: "Not connected"}
		return DSQL_ERROR
	}

	_, dbErr := conn.db.Exec("ROLLBACK")
	if dbErr != nil {
		conn.lastErr = &diagInfo{
			errorCode: -1,
			message:   fmt.Sprintf("Rollback failed: %v", dbErr),
		}
		return DSQL_ERROR
	}
	conn.tx = nil
	return DSQL_SUCCESS
}

//export dpi_end_tran
func dpi_end_tran(hndlType C.sdint2, hndl C.dhandle, txnType C.sdint2) C.DPIRETURN {
	if int16(txnType) == DSQL_COMMIT {
		return dpi_commit(C.dhcon(hndl))
	}
	return dpi_rollback(C.dhcon(hndl))
}

// setConnDiag sets diagnostic info on a connection handle.
func setConnDiag(conn *connHandle, code int32, msg string) {
	conn.lastErr = &diagInfo{errorCode: code, message: msg}
}
