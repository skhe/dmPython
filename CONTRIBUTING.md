# Contributing to dmPython-macOS

Thank you for your interest in contributing! This is a **macOS-focused community fork** of the [official dmPython](https://github.com/DamengDB/dmPython) driver. Contributions that improve macOS support, the Go DPI bridge, CI, and documentation are especially welcome.

## Scope

This fork focuses on:

- macOS ARM64 support via the Go-based DPI bridge
- CI/CD for building and releasing macOS wheels
- Documentation and packaging improvements

For issues with the core dmPython C code or other platforms, please consider reporting them to the [upstream project](https://github.com/DamengDB/dmPython).

## Development Setup

### Prerequisites

- **Go 1.21+** — for building the DPI bridge (`dpi_bridge/`)
- **Python 3.9 – 3.13** — any supported version
- **DPI header files** — obtain from a Dameng database installation and place in `./dpi_include/`

### Build

```bash
# Clone your fork
git clone https://github.com/<your-username>/dmPython.git
cd dmPython

# Build the Go bridge + C extension
python setup.py build_ext --inplace

# Or build a wheel
pip install build
python -m build --wheel
```

### Test

```bash
# Verify the extension loads
python -c "import dmPython; print(dmPython.version)"
```

## Pull Request Process

1. Fork the repository and create a feature branch from `main`.
2. Make your changes with clear, focused commits.
3. Ensure the project builds successfully.
4. Open a pull request against `main` with a clear description of the change.

## Code Style

- **C code**: Follow the existing style in the repository. No reformatting of unchanged code.
- **Go code** (`dpi_bridge/`): Use `gofmt`.
- **Python code**: Follow PEP 8.

## Reporting Issues

When reporting a bug, please include:

- macOS version and architecture (`uname -m`)
- Python version (`python --version`)
- dmPython version (`python -c "import dmPython; print(dmPython.version)"`)
- Dameng database version (if applicable)
- Steps to reproduce the issue

## License

By contributing, you agree that your contributions will be licensed under the [Mulan PSL v2](http://license.coscl.org.cn/MulanPSL2).
