# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [2.5.32] - 2026-03-04

### Fixed

- 修复了 GitHub Actions 中发布步骤与已存在 Release 冲突导致失败的问题：当 tag 已存在时改为 `gh release upload --clobber` 幂等上传。
- 修复了 release 任务偶发“发布成功但缺少 wheel 附件”的风险，新增发布后资产完整性校验并输出缺失清单。

### Changed

- 优化 CI 分层与门禁信号，确保 DM 集成测试按环境条件可控执行，避免误报。
- 统一版本链路文档示例到 `2.5.32`，保持发布元数据与使用说明一致。

## [2.5.31] - 2026-03-03

### Fixed

- 修复了大 CLOB 在特定 Unicode 模式（如 `稳态中文🚀X`）回读错位的问题，消除了 `U+FFFD` 替换字符导致的内容损坏。
- 修复了 CLOB 分块写入在 UTF-8 边界处切分导致的潜在字符错位问题。

### Added

- 新增 P0 CLOB Unicode 回归测试集，覆盖高风险模式、`executemany`、长度契约与子进程防崩溃场景。
- 新增并扩展 P0/P1/P2 集成测试分层与压力场景，提升稳定性回归覆盖。
- 新增 DM 集成测试 CI 工作流与覆盖率产物上传。
- 在 `dpi_bridge` 引入补丁化本地 DM Go 驱动依赖，并记录补丁说明文档。

## [2.5.30] - 2025-09-03

### Fixed

- 修复了多线程并发释放连接的问题

## [2.5.29] - 2025-07-31

### Added

- 增加了 parse_type 连接参数

## [2.5.28] - 2025-07-21

### Fixed

- 修复了 dmPython 查询空间数据的问题

## [2.5.27] - 2025-06-17

### Fixed

- 修复了 dmPython 没有对连接参数长度进行检查的问题

## [2.5.26] - 2025-04-23

### Fixed

- 修复了 dmPython 使用 executemany 函数插入数据时，当字符类型数据中存在当前编码无法识别字符时会发生中断的问题

## [2.5.25] - 2025-04-15

### Fixed

- 修复了 dmPython 查询大字段时，内存持续增长的问题

## [2.5.24] - 2025-04-11

### Fixed

- 修复了 dmPython.connect 函数没有对输入参数 user、password、server、schema 长度进行判断的问题

## [2.5.23] - 2025-04-11

### Fixed

- 修复了 Connection.shutdown 函数没有对输入参数长度进行判断的问题

## [2.5.22] - 2025-02-25

### Changed

- 兼容了 Oracle 在 Number 类型 scale 大于 0 并且为整数时，会输出其小数形式的处理方式

## [2.5.21] - 2025-02-19

### Added

- 增加了对 dmPython.CURSOR 类型绑定参数执行的支持

## [2.5.20] - 2025-01-20

### Fixed

- 修复了使用 IPv6 地址连接达梦数据库失败的问题
- 修复了当输入参数列中有大字段类型时，获取输出参数失败的问题

## [2.5.19] - 2025-01-06

### Fixed

- 修复了 bit 列值为 null 时，returning into 输出参数报错的问题

## [2.5.18] - 2024-12-31

### Added

- 增加了连接参数 dmsvc_path，指定 dm_svc.conf 路径

## [2.5.17] - 2024-12-26

### Changed

- 更改了密码策略，不允许使用默认密码

## [2.5.16] - 2024-11-22

### Fixed

- 修复了 returning into 输出参数类型为 blob 时，会导致程序崩溃的问题
- 修复了 dmPython 读取 bfile 有父目录引用时，报错不正常的问题

### Added

- 增加了 dmPython 安装时可以使用 drivers 目录作为 DM_HOME 目录的支持

## [2.5.15] - 2024-11-20

### Fixed

- 修复了 dmPython 删除不存在的 bfile 目录时，会导致程序崩溃的问题
- 修复了 dmPython 的 callproc 和 callfunc 函数中的 SQL 注入问题

### Changed

- 兼容了 DM7 版本的 DPI

## [2.5.14] - 2024-11-19

### Fixed

- 修复了当 update 和 delete 语句影响行数为 0 时，returning into 输出参数会导致程序崩溃的问题

## [2.5.13] - 2024-11-14

### Fixed

- 修复了 DM_HOME 的搜索逻辑，会优先在当前目录搜索需要的动态库，然后才会去父目录搜索

### Added

- 增加了在使用繁体中文时，使用不支持繁体中文编码时的报错

## [2.5.12] - 2024-11-13

### Fixed

- 修复了 dmPython 使用编码方式 PG_ISO_8859_11、PG_KOI8R、PG_SQL_ASCII 连接数据库报错的问题

## [2.5.11] - 2024-09-20

### Fixed

- 修复了绑定参数输入 blob 或 clob 数据时，程序崩溃的问题
- 消除了 Python 3.12 版本安装 dmPython 时的警告

## [2.5.10] - 2024-09-20

### Fixed

- 修复了 returning into 输出参数返回多行结果时，无法输出空数据的问题

## [2.5.9] - 2024-08-29

### Added

- 增加了对多租户连接参数的支持

### Fixed

- 修复了游标读取 bfile 数据后，退出程序时报错资源清理出错的问题

## [2.5.8] - 2024-07-03

### Added

- 增加了对 nls_numeric_characters 参数的支持，支持以字符串格式返回非标准时间类型

### Fixed

- 修复了多线程下更新 blob 和 clob 数据会发生阻塞的问题
- 修复了超长数据插入时的字符串截断问题

## [2.5.7] - 2024-04-15

### Added

- 增加了 returning into 输出参数支持返回多行结果的支持

### Changed

- 适配 DPI prepare 本地化的修复，调整了一些函数的使用顺序

## [2.5.6] - 2023-12-07

### Fixed

- 修复了获取变长字符串类型时，相关描述信息不准确的问题

## [2.5.5] - 2023-11-08

### Added

- 增加了对 Python 3.12 版本的支持

## [2.5.4] - 2023-10-25

### Fixed

- 修复了数据库推荐类型为 varchar，传入参数类型为 int，数据类型转换失败的错误
