package main

/*
#include <stdint.h>
#include <stddef.h>
#include <string.h>

// DPI primitive types matching DPItypes.h
typedef signed char     sdbyte;
typedef unsigned char   udbyte;
typedef signed short    sdint2;
typedef unsigned short  udint2;
typedef signed int      sdint4;
typedef unsigned int    udint4;
typedef long long int   sdint8;
typedef unsigned long long int udint8;
typedef float           dfloat;
typedef double          ddouble;
typedef void*           dpointer;

// 64-bit platform length types
typedef sdint8          slength;
typedef udint8          ulength;

// Function return type
typedef sdint2          DPIRETURN;

// Handle types — all void*
typedef void*           dhandle;
typedef dhandle         dhenv;
typedef dhandle         dhcon;
typedef dhandle         dhstmt;
typedef dhandle         dhdesc;
typedef dhandle         dhloblctr;
typedef dhandle         dhobjdesc;
typedef dhandle         dhobj;
typedef dhandle         dhbfile;

// Numeric struct
#define DPI_MAX_NUMERIC_LEN 16
typedef struct {
    udbyte  precision;
    signed char scale;
    udbyte  sign;
    udbyte  val[DPI_MAX_NUMERIC_LEN];
} dpi_numeric_t;

// Timestamp struct
typedef struct {
    sdint2  year;
    udint2  month;
    udint2  day;
    udint2  hour;
    udint2  minute;
    udint2  second;
    udint4  fraction;
} dpi_timestamp_t;

// Date struct
typedef struct {
    sdint2  year;
    udint2  month;
    udint2  day;
} dpi_date_t;

// Time struct
typedef struct {
    udint2  hour;
    udint2  minute;
    udint2  second;
} dpi_time_t;
*/
import "C"
import "unsafe"

// DPI return codes
const (
	DSQL_SUCCESS           = C.DPIRETURN(0)
	DSQL_SUCCESS_WITH_INFO = C.DPIRETURN(1)
	DSQL_ERROR             = C.DPIRETURN(-1)
	DSQL_INVALID_HANDLE    = C.DPIRETURN(-2)
	DSQL_NO_DATA           = C.DPIRETURN(100)
	DSQL_NEED_DATA         = C.DPIRETURN(99)
	DSQL_STILL_EXECUTING   = C.DPIRETURN(2)
)

// Handle type identifiers
const (
	DSQL_HANDLE_ENV  = 1
	DSQL_HANDLE_DBC  = 2
	DSQL_HANDLE_STMT = 3
	DSQL_HANDLE_DESC = 4
)

// Special length/indicator values
const (
	DSQL_NULL_DATA    = C.slength(-1)
	DSQL_DATA_AT_EXEC = C.slength(-2)
	DSQL_NTS          = C.slength(-3)
)

// Transaction operations
const (
	DSQL_COMMIT   = 0
	DSQL_ROLLBACK = 1
)

// Parameter binding types
const (
	DSQL_PARAM_INPUT         = 1
	DSQL_PARAM_INPUT_OUTPUT  = 2
	DSQL_PARAM_OUTPUT        = 4
	DSQL_PARAM_OUTPUT_STREAM = 16
)

// Fetch orientations
const (
	DSQL_FETCH_NEXT = 1
)

// C type codes
const (
	DSQL_C_NCHAR     = 0
	DSQL_C_SSHORT    = 1
	DSQL_C_USHORT    = 2
	DSQL_C_SLONG     = 3
	DSQL_C_ULONG     = 4
	DSQL_C_FLOAT     = 5
	DSQL_C_DOUBLE    = 6
	DSQL_C_BIT       = 7
	DSQL_C_STINYINT  = 8
	DSQL_C_UTINYINT  = 9
	DSQL_C_SBIGINT   = 10
	DSQL_C_UBIGINT   = 11
	DSQL_C_BINARY    = 12
	DSQL_C_DATE      = 13
	DSQL_C_TIME      = 14
	DSQL_C_TIMESTAMP = 15
	DSQL_C_NUMERIC   = 16
	DSQL_C_DEFAULT   = 30
	DSQL_C_CLASS     = 31
	DSQL_C_RECORD    = 32
	DSQL_C_ARRAY     = 33
	DSQL_C_SARRAY    = 34

	DSQL_C_INTERVAL_YEAR             = 17
	DSQL_C_INTERVAL_MONTH            = 18
	DSQL_C_INTERVAL_DAY              = 19
	DSQL_C_INTERVAL_HOUR             = 20
	DSQL_C_INTERVAL_MINUTE           = 21
	DSQL_C_INTERVAL_SECOND           = 22
	DSQL_C_INTERVAL_YEAR_TO_MONTH    = 23
	DSQL_C_INTERVAL_DAY_TO_HOUR      = 24
	DSQL_C_INTERVAL_DAY_TO_MINUTE    = 25
	DSQL_C_INTERVAL_DAY_TO_SECOND    = 26
	DSQL_C_INTERVAL_HOUR_TO_MINUTE   = 27
	DSQL_C_INTERVAL_HOUR_TO_SECOND   = 28
	DSQL_C_INTERVAL_MINUTE_TO_SECOND = 29

	DSQL_C_LOB_HANDLE = 999
	DSQL_C_RSET       = 1000
	DSQL_C_WCHAR      = 1001
	DSQL_C_BFILE      = 1002
	DSQL_C_CHAR       = 1003
)

// SQL server type codes
const (
	DSQL_CHAR         = 1
	DSQL_VARCHAR      = 2
	DSQL_BIT          = 3
	DSQL_TINYINT      = 5
	DSQL_SMALLINT     = 6
	DSQL_INT          = 7
	DSQL_BIGINT       = 8
	DSQL_DEC          = 9
	DSQL_FLOAT        = 10
	DSQL_DOUBLE       = 11
	DSQL_BLOB         = 12
	DSQL_BOOLEAN      = 13
	DSQL_DATE         = 14
	DSQL_TIME         = 15
	DSQL_TIMESTAMP    = 16
	DSQL_BINARY       = 17
	DSQL_VARBINARY    = 18
	DSQL_CLOB         = 19
	DSQL_TIME_TZ      = 22
	DSQL_TIMESTAMP_TZ = 23
	DSQL_CLASS        = 24
	DSQL_RECORD       = 25
	DSQL_ARRAY        = 26
	DSQL_SARRAY       = 27
	DSQL_ROWID        = 28
	DSQL_RSET         = 119
	DSQL_BFILE        = 1000
)

// Environment/Connection attributes (from DPIext.h)
const (
	DSQL_ATTR_LOCAL_CODE      = 12345
	DSQL_ATTR_LANG_ID         = 12346
	DSQL_ATTR_TIME_ZONE       = 12348
	DSQL_ATTR_DEC2DOUB_CNVT   = 12350
	DSQL_ATTR_ACCESS_MODE     = 101
	DSQL_ATTR_AUTOCOMMIT      = 102
	DSQL_ATTR_LOGIN_TIMEOUT   = 103
	DSQL_ATTR_TXN_ISOLATION   = 108
	DSQL_ATTR_CURRENT_CATALOG = 109
	DSQL_ATTR_PACKET_SIZE     = 112
	DSQL_ATTR_CONNECTION_TIMEOUT = 113
	DSQL_ATTR_CONNECTION_DEAD = 1209

	DSQL_ATTR_LOGIN_PORT           = 12350
	DSQL_ATTR_LOGIN_USER           = 12352
	DSQL_ATTR_CURRENT_SCHEMA       = 12354
	DSQL_ATTR_INSTANCE_NAME        = 12355
	DSQL_ATTR_LOGIN_SERVER         = 12356
	DSQL_ATTR_SERVER_CODE          = 12349
	DSQL_ATTR_APP_NAME             = 12357
	DSQL_ATTR_COMPRESS_MSG         = 12358
	DSQL_ATTR_USE_STMT_POOL        = 12359
	DSQL_ATTR_SERVER_VERSION       = 12400
	DSQL_ATTR_SSL_PATH             = 12401
	DSQL_ATTR_SSL_PWD              = 12402
	DSQL_ATTR_MPP_LOGIN            = 12403
	DSQL_ATTR_TRX_STATE            = 12404
	DSQL_ATTR_UKEY_NAME            = 12405
	DSQL_ATTR_UKEY_PIN             = 12406
	DSQL_ATTR_RWSEPARATE           = 12408
	DSQL_ATTR_RWSEPARATE_PERCENT   = 12409
	DSQL_ATTR_CURSOR_ROLLBACK_BEHAVIOR = 12410
	DSQL_ATTR_OSAUTH_TYPE          = 12412
	DSQL_ATTR_DDL_AUTOCOMMIT       = 12414
	DSQL_ATTR_COMPATIBLE_MODE      = 12424
	DSQL_ATTR_SHAKE_CRYPTO         = 12426
	DSQL_ATTR_NLS_NUMERIC_CHARACTERS = 12430
	DSQL_ATTR_DM_SVC_PATH          = 12431
)

// Statement attributes
const (
	DSQL_ATTR_QUERY_TIMEOUT    = 0
	DSQL_ATTR_MAX_ROWS         = 1
	DSQL_ATTR_CURSOR_TYPE      = 6
	DSQL_ATTR_PARAMSET_SIZE    = 22
	DSQL_ATTR_ROW_STATUS_PTR   = 25
	DSQL_ATTR_ROWS_FETCHED_PTR = 26
	DSQL_ATTR_ROW_ARRAY_SIZE   = 27
	DSQL_ATTR_CURSOR_SCROLLABLE = -1
	DSQL_ATTR_APP_ROW_DESC     = 10010
	DSQL_ATTR_APP_PARAM_DESC   = 10011
	DSQL_ATTR_IMP_ROW_DESC     = 10012
	DSQL_ATTR_IMP_PARAM_DESC   = 10013
	DSQL_ATTR_SQL_CHARSET      = 20000
	DSQL_ATTR_IGN_BP_ERR       = 20001
)

// Cursor types
const (
	DSQL_CURSOR_FORWARD_ONLY = 0
	DSQL_CURSOR_STATIC       = 3
)

// Autocommit values
const (
	DSQL_AUTOCOMMIT_OFF = 0
	DSQL_AUTOCOMMIT_ON  = 1
)

// Encoding constants (from DPIext.h)
const (
	PG_UTF8     = 1
	PG_GBK      = 2
	PG_GB18030  = 10
)

// Diagnostics field identifiers
const (
	DSQL_DIAG_NUMBER                  = 1
	DSQL_DIAG_DYNAMIC_FUNCTION_CODE   = 2
	DSQL_DIAG_ROW_COUNT               = 3
	DSQL_DIAG_EXPLAIN                 = 5
	DSQL_DIAG_ROWID                   = 6
	DSQL_DIAG_EXECID                  = 7
	DSQL_DIAG_SERVER_STAT             = 8
	DSQL_DIAG_MESSAGE_TEXT            = 102
	DSQL_DIAG_ERROR_CODE              = 103
)

// Statement function codes (DSQL_DIAG_DYNAMIC_FUNCTION_CODE values)
const (
	DSQL_DIAG_FUNC_CODE_INVALID   = 0
	DSQL_DIAG_FUNC_CODE_SELECT    = 1
	DSQL_DIAG_FUNC_CODE_INSERT    = 2
	DSQL_DIAG_FUNC_CODE_DELETE    = 3
	DSQL_DIAG_FUNC_CODE_UPDATE    = 4
	DSQL_DIAG_FUNC_CODE_CREATE_DB = 5
	DSQL_DIAG_FUNC_CODE_CREATE_TAB = 6
	DSQL_DIAG_FUNC_CODE_DROP_TAB  = 7
	DSQL_DIAG_FUNC_CODE_CALL      = 28
	DSQL_DIAG_FUNC_CODE_MERGE     = 69
	DSQL_DIAG_FUNC_CODE_SET_CURRENT_SCHEMA = 68
)

// Descriptor field identifiers
const (
	DSQL_DESC_COUNT          = 1001
	DSQL_DESC_TYPE           = 1002
	DSQL_DESC_LENGTH         = 1003
	DSQL_DESC_PRECISION      = 1005
	DSQL_DESC_SCALE          = 1006
	DSQL_DESC_NULLABLE       = 1008
	DSQL_DESC_INDICATOR_PTR  = 1009
	DSQL_DESC_DATA_PTR       = 1010
	DSQL_DESC_NAME           = 1011
	DSQL_DESC_OCTET_LENGTH   = 1013
	DSQL_DESC_DISPLAY_SIZE   = 6 // same as DSQL_COLUMN_DISPLAY_SIZE
	DSQL_DESC_PARAMETER_TYPE = 33
	DSQL_DESC_OBJ_DESCRIPTOR = 10001
	DSQL_DESC_BIND_PARAMETER_TYPE = 10003
)

// Column attribute identifiers
const (
	DSQL_COLUMN_COUNT        = 0
	DSQL_COLUMN_NAME         = 1
	DSQL_COLUMN_TYPE         = 2
	DSQL_COLUMN_LENGTH       = 3
	DSQL_COLUMN_PRECISION    = 4
	DSQL_COLUMN_SCALE        = 5
	DSQL_COLUMN_DISPLAY_SIZE = 6
	DSQL_COLUMN_NULLABLE     = 7
	DSQL_COLUMN_TABLE_NAME   = 15
)

// Nullable values
const (
	DSQL_NO_NULLS         = 0
	DSQL_NULLABLE         = 1
	DSQL_NULLABLE_UNKNOWN = 2
)

// Default TCP port
const DSQL_DEAFAULT_TCPIP_PORT = 5236

// ISO transaction isolation levels
const (
	ISO_LEVEL_READ_UNCOMMITTED = 0
	ISO_LEVEL_READ_COMMITTED   = 1
	ISO_LEVEL_REPEATABLE_READ  = 2
	ISO_LEVEL_SERIALIZABLE     = 3
)

// Language constants
const (
	LANGUAGE_CN     = 0
	LANGUAGE_EN     = 1
	LANGUAGE_CNT_HK = 2
)

// goString converts a C sdbyte* to a Go string. If length < 0, it's NTS.
func goString(s *C.sdbyte, length C.sdint4) string {
	if s == nil {
		return ""
	}
	if length < 0 {
		// null-terminated string
		return C.GoString((*C.char)(unsafe.Pointer(s)))
	}
	return C.GoStringN((*C.char)(unsafe.Pointer(s)), C.int(length))
}

// cStringLen writes a Go string into a C buffer and returns the length written.
func cStringLen(dst *C.sdbyte, bufLen int, src string) C.sdint2 {
	if dst == nil || bufLen <= 0 {
		return C.sdint2(len(src))
	}
	b := []byte(src)
	n := len(b)
	if n >= bufLen {
		n = bufLen - 1
	}
	if n > 0 {
		C.memcpy(unsafe.Pointer(dst), unsafe.Pointer(&b[0]), C.size_t(n))
	}
	// null terminate
	*(*C.sdbyte)(unsafe.Pointer(uintptr(unsafe.Pointer(dst)) + uintptr(n))) = 0
	return C.sdint2(n)
}
