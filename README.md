[![CI](https://github.com/skhe/dmPython/actions/workflows/build-wheels.yml/badge.svg)](https://github.com/skhe/dmPython/actions/workflows/build-wheels.yml)
[![License: MulanPSL-2.0](https://img.shields.io/badge/license-MulanPSL--2.0-blue.svg)](http://license.coscl.org.cn/MulanPSL2)
[![Python versions](https://img.shields.io/badge/python-3.9--3.13-blue.svg)]()
[![macOS ARM64](https://img.shields.io/badge/platform-macOS%20ARM64-lightgrey.svg)]()

# dmPython-macOS

A Python DB-API 2.0 driver for the [Dameng (DM8)](https://www.dameng.com/) database — **macOS ARM64 edition** with a built-in Go bridge.

This is a community fork of the [official dmPython](https://github.com/DamengDB/dmPython) driver. The upstream project relies on a proprietary C library (`libdmdpi`) that is not available for macOS. This fork replaces it with a Go-based DPI bridge (`dpi_bridge/`), enabling native macOS ARM64 support without requiring a full Dameng installation.

## Installation

Download a pre-built wheel from [GitHub Releases](https://github.com/skhe/dmPython/releases):

```bash
pip install dmPython_macOS-2.5.31-cp312-cp312-macosx_14_0_arm64.whl
```

## Quick Start

```python
import dmPython

conn = dmPython.connect(
    user="SYSDBA",
    password="SYSDBA001",
    server="localhost",
    port=5236,
)

cursor = conn.cursor()
cursor.execute("SELECT * FROM SYSOBJECTS WHERE ROWNUM <= 5")

for row in cursor.fetchall():
    print(row)

cursor.close()
conn.close()
```

## Building from Source

**Prerequisites:**

- Go 1.21+ (to compile the DPI bridge)
- Python 3.9 – 3.13
- DPI header files — place them in `./dpi_include/` or set `DM_HOME`

```bash
# Clone the repository
git clone https://github.com/skhe/dmPython.git
cd dmPython

# Build the wheel
python -m build --wheel

# Or build the extension in-place for development
python setup.py build_ext --inplace
```

To skip the Go build step (if you already have `libdmdpi.dylib`):

```bash
DMPYTHON_SKIP_GO_BUILD=1 python -m build --wheel
```

## Project Structure

```
dmPython/
├── setup.py              # Build script
├── pyproject.toml        # Project metadata
├── docs/                 # Project docs (zh README, technical notes)
├── scripts/              # Local/ops scripts
├── src/native/           # C extension sources and headers
├── dpi_bridge/           # Go-based DPI bridge (replaces proprietary libdmdpi)
│   ├── main.go
│   ├── go.mod / go.sum
│   └── ...
├── dpi_include/          # DPI header files (not distributed, see README)
├── src/native/py_Dameng.c/h  # Module entry, type registration, exception hierarchy
├── src/native/strct.h        # Core struct definitions (Environment, Connection, Cursor)
├── src/native/Connection.c   # Connection management
├── src/native/Cursor.c       # Cursor operations, SQL execution
├── src/native/var.c          # Variable management core
├── src/native/v*.c           # Type-specific variable handlers
├── src/native/ex*.c          # External object interfaces (LOB, BFILE, Object)
└── .github/workflows/    # CI: builds macOS ARM64 wheels for Python 3.9–3.13
```

## License

Licensed under [Mulan PSL v2](http://license.coscl.org.cn/MulanPSL2).

---

[中文文档 (docs/README_zh.md)](docs/README_zh.md)
