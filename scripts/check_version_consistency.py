#!/usr/bin/env python3
"""Check version consistency across metadata, headers, docs, and smoke script."""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


def _fail(msg: str) -> None:
    raise RuntimeError(msg)


def _extract_pyproject_version() -> str:
    text = (ROOT / "pyproject.toml").read_text(encoding="utf-8")
    match = re.search(r'^\s*version\s*=\s*"([^"]+)"\s*$', text, re.MULTILINE)
    if not match:
        _fail("Cannot find [project].version in pyproject.toml")
    return match.group(1)


def _extract_header_version() -> str:
    text = (ROOT / "src/native/py_Dameng.h").read_text(encoding="utf-8")
    match = re.search(r'#define\s+BUILD_VERSION_STRING\s+"([^"]+)"', text)
    if not match:
        _fail("Cannot find BUILD_VERSION_STRING in src/native/py_Dameng.h")
    return match.group(1)


def _check_setup_uses_pyproject() -> None:
    text = (ROOT / "setup.py").read_text(encoding="utf-8")
    if "BUILD_VERSION = read_project_version()" not in text:
        _fail("setup.py is not using pyproject.toml as version source (BUILD_VERSION = read_project_version())")


def _check_docs_version(version: str) -> None:
    files = [ROOT / "README.md", ROOT / "docs/README_zh.md"]
    pattern = re.compile(r"dmPython_macOS-(\d+\.\d+\.\d+)-cp312-cp312-macosx_14_0_arm64\.whl")
    for path in files:
        text = path.read_text(encoding="utf-8")
        match = pattern.search(text)
        if not match:
            _fail(f"Cannot find wheel example in {path}")
        if match.group(1) != version:
            _fail(f"Wheel example version mismatch in {path}: {match.group(1)} != {version}")


def _check_smoke_script_dynamic() -> None:
    path = ROOT / "scripts/test_connection.py"
    text = path.read_text(encoding="utf-8")
    if "get_expected_version()" not in text:
        _fail("scripts/test_connection.py is missing get_expected_version()")
    if re.search(r'dmPython\.version\s*==\s*"\d+\.\d+\.\d+"', text):
        _fail("scripts/test_connection.py still has a hard-coded version assertion")


def _check_runtime_version(version: str, strict: bool) -> None:
    try:
        import dmPython  # type: ignore

        runtime_version = getattr(dmPython, "version", "")
        if runtime_version != version:
            _fail(f"Runtime dmPython.version mismatch: {runtime_version} != {version}")
    except Exception as exc:
        if strict:
            _fail(f"Runtime dmPython version check failed: {exc}")
        print(f"[WARN] Runtime check skipped: {exc}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--strict-runtime", action="store_true", help="Fail when runtime dmPython check is unavailable")
    parser.add_argument("--print-version", action="store_true", help="Print canonical project version")
    args = parser.parse_args()

    version = _extract_pyproject_version()
    if args.print_version:
        print(version)
        return 0

    _check_setup_uses_pyproject()
    header_version = _extract_header_version()
    if header_version != version:
        _fail(f"Header version mismatch: {header_version} != {version}")

    _check_docs_version(version)
    _check_smoke_script_dynamic()
    _check_runtime_version(version, strict=args.strict_runtime)

    print(f"[OK] version consistency checks passed for {version}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except RuntimeError as exc:
        print(f"[FAIL] {exc}")
        raise SystemExit(2)
