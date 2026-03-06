# 安装指南

## 支持矩阵

- Python: 3.9 - 3.13
- 平台（本仓库发布目标）: macOS ARM64

> 说明：本项目是官方 `dmPython` 的 macOS ARM64 社区 fork。Linux/Windows 生产环境请优先评估官方发布版本。

## 方式一：安装预编译 wheel（推荐）

从 GitHub Releases 下载后安装：

```bash
pip install dmPython_macOS-<version>-cp312-cp312-macosx_14_0_arm64.whl
```

## 方式二：从源码构建

前置条件：

- Go 1.21+
- Python 3.9+
- DPI 头文件（放在 `dpi_include/` 或设置 `DM_HOME`）

```bash
git clone https://github.com/skhe/dmPython.git
cd dmPython
python -m build --wheel
```

本地开发构建扩展：

```bash
python setup.py build_ext --inplace
```

如果本地已有 `libdmdpi.dylib`，可跳过 Go 构建：

```bash
DMPYTHON_SKIP_GO_BUILD=1 python -m build --wheel
```

## 安装验证

```bash
python - <<'PY'
import dmPython
print("version:", dmPython.version)
print("buildtime:", dmPython.buildtime)
PY
```
