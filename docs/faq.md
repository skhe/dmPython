# FAQ

## 1. `host` 和 `server` 有什么区别？

`connect()` 同时提供了 `host` 与 `server` 参数，两者语义相近，但不能同时设置。

## 2. `connect()` 最简参数是什么？

至少应提供可用凭据和目标地址，常见是：`user`、`password`、`server`、`port`。

## 3. 支持字典游标吗？

支持。连接时传 `cursorclass=dmPython.DictCursor`，查询结果按列名映射为字典。

## 4. 为什么 `Cursor.parse()` 报 `NotSupportedError`？

这是当前实现状态，不是调用方式问题。可使用 `prepare()` 或直接 `execute()`。

## 5. 如何查看连接是否失效？

可读 `connection.connection_dead` 或调用 `connection.ping(reconnect=1)`。

## 6. 如何调试服务端日志开关？

使用 `connection.debug()`，参数可选 `DEBUG_OPEN/DEBUG_CLOSE/DEBUG_SWITCH/DEBUG_SIMPLE`。
