[![CI](https://github.com/skhe/dmPython/actions/workflows/build-wheels.yml/badge.svg)](https://github.com/skhe/dmPython/actions/workflows/build-wheels.yml)
[![License: MulanPSL-2.0](https://img.shields.io/badge/license-MulanPSL--2.0-blue.svg)](http://license.coscl.org.cn/MulanPSL2)
[![Python versions](https://img.shields.io/badge/python-3.9--3.13-blue.svg)]()
[![macOS ARM64](https://img.shields.io/badge/platform-macOS%20ARM64-lightgrey.svg)]()

# dmPython-macOS

dmPython 是达梦数据库（DM8）的原生 Python 驱动程序，遵循 [Python DB API 2.0](https://www.python.org/dev/peps/pep-0249/) 规范。通过 C 扩展模块和 DPI（Dameng Programming Interface）库与达梦数据库通信。

本项目是 [官方 dmPython](https://github.com/DamengDB/dmPython) 的社区 fork。上游项目依赖的专有 C 库 `libdmdpi` 不提供 macOS 版本，本 fork 使用 Go 编写的 DPI 桥接库（`dpi_bridge/`）替代，实现原生 macOS ARM64 支持，无需安装完整的达梦数据库。

**当前版本：** 2.5.31

## 特性

- 遵循 Python DB API 2.0 规范
- 支持 Python 3.9 – 3.13
- 支持 SQLAlchemy 和 Django 框架
- 支持 BLOB/CLOB、BFILE、对象类型等丰富的数据类型
- 支持元组游标和字典游标两种模式
- macOS ARM64 原生支持（通过 Go DPI 桥接库）

## 安装

从 [GitHub Releases](https://github.com/skhe/dmPython/releases) 下载预编译 wheel：

```bash
pip install dmPython_macOS-2.5.31-cp312-cp312-macosx_14_0_arm64.whl
```

## 快速开始

```python
import dmPython

# 建立连接
conn = dmPython.connect(user='SYSDBA', password='SYSDBA001', server='localhost', port=5236)

# 创建游标并执行 SQL
cursor = conn.cursor()
cursor.execute("SELECT * FROM SYSOBJECTS WHERE ROWNUM <= 5")

# 获取结果
rows = cursor.fetchall()
for row in rows:
    print(row)

# 关闭连接
cursor.close()
conn.close()
```

## 从源码构建

**前置条件：**

- Go 1.21+（编译 DPI 桥接库）
- Python 3.9 – 3.13
- DPI 头文件 — 放入 `./dpi_include/` 目录或设置 `DM_HOME` 环境变量

```bash
# 克隆仓库
git clone https://github.com/skhe/dmPython.git
cd dmPython

# 构建 wheel
python -m build --wheel

# 或在本地构建扩展（开发用）
python setup.py build_ext --inplace
```

跳过 Go 构建步骤（已有 `libdmdpi.dylib` 时）：

```bash
DMPYTHON_SKIP_GO_BUILD=1 python -m build --wheel
```

### 调试追踪

在 `setup.py` 中取消以下注释，重新编译后会在当前目录生成 `dmPython_trace.log`：

```python
define_macros.append(('TRACE', None))
```

## 项目结构

```
dmPython/
├── setup.py              # 构建脚本
├── pyproject.toml        # 项目元数据
├── docs/                 # 项目文档（中文 README、技术报告）
├── scripts/              # 本地/运维脚本
├── src/native/           # C 扩展源码与头文件
├── dpi_bridge/           # Go DPI 桥接库（替代专有 libdmdpi）
│   ├── main.go
│   ├── go.mod / go.sum
│   └── ...
├── dpi_include/          # DPI 头文件（不随源码分发，见 README）
├── src/native/py_Dameng.c/h  # 模块入口，类型注册，异常层次
├── src/native/strct.h        # 核心结构定义 (Environment, Connection, Cursor)
├── src/native/Connection.c   # 连接管理
├── src/native/Cursor.c       # 游标操作，SQL 执行
├── src/native/var.c          # 变量管理核心
├── src/native/v*.c           # 各类型变量处理器
├── src/native/ex*.c          # 外部对象接口 (LOB, BFILE, Object)
└── .github/workflows/    # CI：构建 macOS ARM64 wheels (Python 3.9–3.13)
```

## 许可证

本项目采用 [木兰宽松许可证 第2版（Mulan PSL v2）](http://license.coscl.org.cn/MulanPSL2) 授权。

---

[English README (README.md)](../README.md)
