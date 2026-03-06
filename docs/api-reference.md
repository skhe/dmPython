# API 参考

本页根据扩展源码 `src/native/py_Dameng.c`、`src/native/Connection.c`、`src/native/Cursor.c` 汇总公开接口。

## 模块级对象

### DB-API 元信息

- `dmPython.apilevel = "2.0"`
- `dmPython.threadsafety = 1`
- `dmPython.paramstyle = "qmark"`
- `dmPython.version`
- `dmPython.buildtime`

### 连接入口

- `dmPython.connect(...)`
- `dmPython.Connect(...)`

两者均为 `Connection` 类型构造入口。

### `connect()` 参数

```python
dmPython.connect(
    user=None,
    password=None,
    dsn=None,
    host=None,
    server=None,
    port=None,
    access_mode=None,
    autoCommit=None,
    connection_timeout=None,
    login_timeout=None,
    txn_isolation=None,
    app_name=None,
    compress_msg=None,
    use_stmt_pool=None,
    ssl_path=None,
    ssl_pwd=None,
    mpp_login=None,
    ukey_name=None,
    ukey_pin=None,
    rwseparate=None,
    rwseparate_percent=None,
    cursor_rollback_behavior=None,
    lang_id=None,
    local_code=None,
    cursorclass=None,
    schema=None,
    shake_crypto=None,
    catalog=None,
    dmsvc_path=None,
    parse_type=None,
)
```

说明：

- `host` 与 `server` 互斥（只允许设置一个）。
- `user` 支持 `user/password@server:port[/schema][?catalog=...]` 形式。
- 常量参数建议使用模块常量（如 `DSQL_AUTOCOMMIT_ON`、`ISO_LEVEL_READ_COMMITTED`）。

### 模块函数

- `DateFromTicks(ticks)`
- `TimeFromTicks(ticks)`
- `TimestampFromTicks(ticks)`
- `StringFromBytes(bs)`

### 日期时间类型别名

- `Date`
- `Time`
- `Timestamp`
- `DATETIME`

### 游标类型常量

- `TupleCursor`
- `DictCursor`

用于 `connect(cursorclass=...)`。

## Connection

### 方法

- `cursor()`
- `commit()`
- `rollback()`
- `close()`
- `disconnect()`（`close()` 别名）
- `debug(debug_type=dmPython.DEBUG_OPEN)`
- `shutdown(shutdown_type=dmPython.SHUTDOWN_DEFAULT)`
- `explain(statement)`
- `ping(reconnect=0)`
- `__enter__()`
- `__exit__(exc_type, exc_value, exc_traceback)`

### 成员属性（只读）

- `dsn`
- `server_status`
- `warning`

### 计算属性（含可写项）

可读写：

- `access_mode`
- `async_enable`
- `auto_ipd`
- `local_code`
- `lang_id`
- `app_name`
- `txn_isolation`
- `compress_msg`
- `rwseparate`
- `rwseparate_percent`
- `use_stmt_pool`
- `ssl_path`
- `mpp_login`
- `autoCommit`
- `autocommit`
- `connection_dead`
- `connection_timeout`
- `login_timeout`
- `packet_size`
- `port`

只读：

- `server_code`
- `current_schema`
- `str_case_sensitive`
- `max_row_size`
- `current_catalog`
- `trx_state`
- `server_version`
- `cursor_rollback_behavior`
- `user`
- `server`
- `inst_name`
- `version`
- `max_identifier_length`
- `outputtypehandler`
- `stmtcachesize`

以上属性多数存在同名 `DSQL_ATTR_*` 别名，例如：

- `connection.autoCommit` <=> `connection.DSQL_ATTR_AUTOCOMMIT`
- `connection.port` <=> `connection.DSQL_ATTR_LOGIN_PORT`

## Cursor

### 方法

- `execute(statement, params=None, **kwargs)`
- `executedirect(statement)`
- `fetchall()`
- `fetchone()`
- `fetchmany(rows=arraysize)`
- `prepare(statement)`
- `parse(statement)`（当前实现返回 `NotSupportedError`）
- `setinputsizes(*args, **kwargs)`
- `executemany(statement, seq_of_params)`
- `callproc(name, params=None)`
- `callfunc(name, params=None)`
- `setoutputsize(size, column=-1)`
- `var(typ, size=0, arraysize=cursor.arraysize, inconverter=None, outconverter=None, typename=None, encoding_errors=None, bypass_decode=False, encodingErrors=None)`
- `arrayvar(...)`（当前实现返回 `NotSupportedError`）
- `bindnames()`（当前实现返回 `NotSupportedError`）
- `close()`
- `next()`
- `nextset()`
- `__enter__()`
- `__exit__(exc_type, exc_value, exc_traceback)`

### 成员属性

- `arraysize`（可写）
- `bindarraysize`（可写）
- `rowcount`（只读）
- `rownumber`（只读）
- `with_rows`（只读）
- `statement`（只读）
- `connection`（只读）
- `column_names`（只读）
- `lastrowid`（只读）
- `execid`（只读）
- `_isClosed`（内部）
- `_statement`（内部）
- `output_stream`（可写）
- `description`（只读计算属性）

## 异常层次

- `Warning`
- `Error`
  - `InterfaceError`
  - `DatabaseError`
    - `DataError`
    - `OperationalError`
    - `IntegrityError`
    - `InternalError`
    - `ProgrammingError`
    - `NotSupportedError`

此外还提供 `DmError` 对象（包含 `code`、`offset`、`message`、`context`）。

## 常量

### 调试与关库

- `DEBUG_CLOSE`
- `DEBUG_OPEN`
- `DEBUG_SWITCH`
- `DEBUG_SIMPLE`
- `SHUTDOWN_DEFAULT`
- `SHUTDOWN_ABORT`
- `SHUTDOWN_IMMEDIATE`
- `SHUTDOWN_TRANSACTIONAL`
- `SHUTDOWN_NORMAL`

### 事务与访问模式

- `ISO_LEVEL_READ_DEFAULT`
- `ISO_LEVEL_READ_UNCOMMITTED`
- `ISO_LEVEL_READ_COMMITTED`
- `ISO_LEVEL_REPEATABLE_READ`
- `ISO_LEVEL_SERIALIZABLE`
- `DSQL_MODE_READ_ONLY`
- `DSQL_MODE_READ_WRITE`
- `DSQL_AUTOCOMMIT_ON`
- `DSQL_AUTOCOMMIT_OFF`

### 编码与语言

- `PG_UTF8`
- `PG_GBK`
- `PG_BIG5`
- `PG_ISO_8859_9`
- `PG_EUC_JP`
- `PG_EUC_KR`
- `PG_KOI8R`
- `PG_ISO_8859_1`
- `PG_SQL_ASCII`
- `PG_GB18030`
- `PG_ISO_8859_11`
- `LANGUAGE_CN`
- `LANGUAGE_EN`
- `LANGUAGE_CNT_HK`（条件编译）

### 其他连接行为

- `DSQL_TRUE`
- `DSQL_FALSE`
- `DSQL_RWSEPARATE_ON`
- `DSQL_RWSEPARATE_OFF`
- `DSQL_TRX_ACTIVE`
- `DSQL_TRX_COMPLETE`
- `DSQL_MPP_LOGIN_GLOBAL`
- `DSQL_MPP_LOGIN_LOCAL`
- `DSQL_CB_CLOSE`
- `DSQL_CB_PRESERVE`

### 数据类型对象

模块还导出一组数据类型对象，可用于绑定/类型判断：

- `INTERVAL`, `YEAR_MONTH_INTERVAL`
- `BLOB`, `CLOB`, `LOB`
- `BFILE`, `exBFILE`
- `LONG_BINARY`, `LONG_STRING`
- `DATE`, `TIME`, `TIMESTAMP`
- `CURSOR`
- `STRING`, `FIXED_STRING`, `BINARY`, `FIXED_BINARY`
- `OBJECTVAR`, `objectvar`
- `NUMBER`, `DOUBLE`, `REAL`, `BOOLEAN`, `DECIMAL`
- `TIME_WITH_TIMEZONE`, `TIMESTAMP_WITH_TIMEZONE`
- `BIGINT`, `ROWID`
