# dmPython-macOS 文档

这是一套面向 `dmPython-macOS` 的完整使用文档，覆盖：

- 安装与环境准备
- 快速开始
- 完整 API 参考（`connect()`、`Connection`、`Cursor`、异常、常量）
- 可独立运行示例（CRUD、批量、LOB、存储过程、连接池）
- 从官方 `dmPython` 迁移到 macOS 版本的指南

## 文档导航

- [安装指南](installation.md)
- [快速开始](quickstart.md)
- [API 参考](api-reference.md)
- 示例
  - [基本增删改查](examples/basic-crud.md)
  - [批量插入](examples/bulk-insert.md)
  - [LOB 大对象操作](examples/lob-handling.md)
  - [存储过程](examples/stored-proc.md)
  - [连接池](examples/connection-pool.md)
- [迁移指南](migration.md)
- [常见问题](faq.md)

## 约定

本文示例默认通过环境变量读取连接参数：

- `DM_HOST`
- `DM_PORT`
- `DM_USER`
- `DM_PASSWORD`

示例默认使用 `DM_PORT=5236`、`DM_USER=SYSDBA`、`DM_PASSWORD=SYSDBA001`。
