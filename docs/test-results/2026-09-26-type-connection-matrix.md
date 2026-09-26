# 数据类型与连接选项行为矩阵（2026-09-26）

## 本次验证

在本机 OrbStack 的隔离达梦官方 ARM64 实例上，使用非管理员测试账号运行 `--require-dm -m requires_dm tests`。Python 3.9、3.10、3.11、3.12、3.13 均为 **104 passed、0 failed、0 errors、0 skipped**；每个版本另有 2 个非真实库用例不在本次选择中。结果由 JUnit 摘要的零跳过门槛复核。此前相同环境每版为 66 个真实库用例，本次新增 38 个 P1 用例。GitHub 托管的开发镜像需等本分支 CI 再次验证。

| 范围 | 已验证行为 |
| --- | --- |
| 整数 | `SMALLINT` 下界、`INTEGER` 上界、`BIGINT` 上界往返 |
| 数值 | `DECIMAL(18,6)`、`FLOAT`、`DOUBLE` 的有限值往返 |
| 文本 | `VARCHAR`/`NVARCHAR` 中文和 emoji；`CHAR`/`NCHAR` 定长补空格 |
| 二进制 | `BINARY` 补零、`VARBINARY` 包含 NUL 与非 ASCII 字节 |
| 日期时间 | 闰日 `DATE`、`TIME` 时分秒、带微秒 `TIMESTAMP` 往返 |
| 空值 | 整数、十进制、文本、二进制、日期、时间戳列的 `NULL` 往返 |
| 连接地址 | `server`、`host`、`dsn=host:port` 均可连接；`host` 与 `server` 同时传入会报错 |
| 连接行为 | 默认元组与 `DictCursor` 行形状、`schema`、`autoCommit` 开关及开启后无需显式提交的持久化、`txn_isolation` 读回 |
| 选项基础检查 | 非数字 `port` 报错；`connection_timeout`、`login_timeout`、`compress_msg`、`use_stmt_pool` 可建连并执行查询 |

`DECIMAL`、`FLOAT`、`DOUBLE`、`TIME`、`TIMESTAMP` 在当前默认读取路径通常返回字符串。矩阵断言数值或时间内容，同时保留未来改善 Python 返回类型的空间。

## 已发现的问题

1. **连接选项旧问题**：此前 `connection_timeout=5`、`login_timeout=5` 和 `app_name='dmpython_matrix'` 建连后，属性读回分别为 `0`、`0` 和空字符串；`login_timeout=1` 遇到无响应服务端时，4 秒内仍无法返回。本轮已修复并加入行为回归。测试账号没有 `SYS.V$SESSIONS` 查询权限，尚未从服务端会话视图独立核实 `app_name`。

本次修复了 `datetime.time` 绑定在亚洲/上海本地时区下偏移五分钟的问题：桥接层曾以公元 0 年构造无时区的 `TIME`，触发历史时区偏移；改用现代锚定日期后，上述五版用例均验证 `23:59:58` 原样写入和读回。

## 后续修复：高精度十进制写入

`DECIMAL(30,8)` 参数原先经过默认精度的二进制浮点转换，使 `12345678901234567890.12345678` 的小数位归零。现在直接解析十进制字符，并覆盖 `Decimal`、文本、负数和科学计数法四种参数形式。本机官方 DM8 上 Python 3.10 的完整真实库回归为 **108 passed、0 failed、0 skipped**；尚需以 CI 结果验证托管镜像。

## 后续修复：带时区时间

绑定带固定时区的 Python `datetime.time`、`datetime.datetime` 或带偏移量的文本时，保留偏移量；读取 `TIME WITH TIME ZONE` 和 `TIMESTAMP WITH TIME ZONE` 时也保留偏移量。回归以时间点相等为准，允许数据库把输入时区规范化为服务器时区。本机官方 DM8 上，Python 3.9、3.10、3.11、3.12、3.13 的完整真实库回归均为 **112 passed、0 failed**，另有 2 个非真实库用例未选入。

## 后续修复：区间与连接超时

`INTERVAL DAY TO SECOND` 现可与 Python `datetime.timedelta` 往返，包括负数、微秒、零值和负 10 万天；读取时描述类型为 `dmPython.INTERVAL`。`INTERVAL YEAR TO MONTH` 的文本参数与读取类型 `dmPython.YEAR_MONTH_INTERVAL` 也已验证，两个区间类型的 `NULL` 往返通过。旧实现对大负区间读取发生 32 位整数溢出，本轮已修复。

当时的实现将 `login_timeout=1` 解释为 1 秒，并把 `connection_timeout` 用于 TCP 拨号；这与达梦官方接口定义不符，已在后续修复中更正。本机官方 DM8 上，当时 Python 3.9、3.10、3.11、3.12、3.13 的完整真实库回归均为 **121 passed、0 failed**，另有 2 个非真实库用例未选入。

## 下一轮边界

需要继续覆盖其他十进制边界、时区边界、其他区间限定形式、复杂对象与数组、BFILE、不同编码，以及 SSL、UKey、MPP、读写分离、拨号超时和故障转移的实际效果。当前仅有一版官方 DM8 服务端和一版 GitHub CI 开发镜像的历史基线；不能据此推断跨达梦服务端版本兼容。

## 后续回归：特殊连接地址与服务配置

在本机官方 DM8 上验证了方括号形式的 IPv6 地址（`server`、`host`、`dsn`）、包含连接串分隔符的密码、显式 `dmsvc_path` 服务名解析，以及首个端点握手失败后切换到第二个端点。服务配置路径包含空格和 `&` 时也可连接。此前 `dmsvc_path` 被桥接层静默忽略，服务名会被当作普通 DNS 主机；现已修复。SSL、UKey、MPP 与读写分离仍需对应环境的实际效果验证。

## 后续回归：十进制与时区边界

`DECIMAL(30,8)` 的最大绝对值、正负最小非零值与零值单条写入通过；混合 `Decimal`、文本、科学计数法和 `NULL` 的 `executemany` 批量写入也保留精度。`TIME WITH TIME ZONE` 与 `TIMESTAMP WITH TIME ZONE` 在 `+14:00`、`-12:59` 偏移量和跨日附近的微秒值往返通过。本机官方 DM8 上 Python 3.9 至 3.13 各有 18 项针对性用例通过。
