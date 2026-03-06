#!/usr/bin/env python3
"""Check version consistency across metadata, headers, docs, and smoke script."""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path
from typing import Iterable


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


def _extract_names_from_c_array(text: str, array_name: str) -> list[str]:
    pattern = rf"static\s+Py(?:MethodDef|MemberDef|GetSetDef)\s+{re.escape(array_name)}\[\]\s*=\s*\{{(.*?)\n\}};"
    match = re.search(pattern, text, re.DOTALL)
    if not match:
        _fail(f"Cannot find C array: {array_name}")
    body = match.group(1)
    body = re.sub(r"/\*.*?\*/", "", body, flags=re.DOTALL)
    body = re.sub(r"//.*", "", body)
    names = re.findall(r'\{\s*"([^"]+)"', body)
    return [n for n in names if n != "NULL"]


def _extract_connect_keywords(connection_text: str) -> list[str]:
    match = re.search(
        r"static char \*keywordList\[\]\s*=\s*\{(.*?)NULL\s*\};",
        connection_text,
        re.DOTALL,
    )
    if not match:
        _fail("Cannot find connect() keywordList in Connection.c")
    return re.findall(r'"([^"]+)"', match.group(1))


def _missing_tokens(doc_text: str, tokens: Iterable[str]) -> list[str]:
    missing = [t for t in tokens if t not in doc_text]
    return sorted(set(missing))


def _check_docs_structure() -> None:
    required = [
        "docs/index.md",
        "docs/installation.md",
        "docs/quickstart.md",
        "docs/api-reference.md",
        "docs/examples/basic-crud.md",
        "docs/examples/bulk-insert.md",
        "docs/examples/lob-handling.md",
        "docs/examples/stored-proc.md",
        "docs/examples/connection-pool.md",
        "docs/migration.md",
        "docs/faq.md",
        "mkdocs.yml",
    ]
    missing = [path for path in required if not (ROOT / path).exists()]
    if missing:
        _fail(f"Missing required docs files: {missing}")


def _check_api_docs_coverage() -> None:
    api_doc = (ROOT / "docs/api-reference.md").read_text(encoding="utf-8")
    # Native C sources contain mixed legacy comments; decode losslessly.
    conn_c = (ROOT / "src/native/Connection.c").read_text(encoding="latin-1")
    cur_c = (ROOT / "src/native/Cursor.c").read_text(encoding="latin-1")
    mod_c = (ROOT / "src/native/py_Dameng.c").read_text(encoding="latin-1")

    connect_keywords = _extract_connect_keywords(conn_c)
    conn_methods = _extract_names_from_c_array(conn_c, "g_ConnectionMethods")
    conn_members = _extract_names_from_c_array(conn_c, "g_ConnectionMembers")
    conn_getset = _extract_names_from_c_array(conn_c, "g_ConnectionCalcMembers")
    cur_methods = _extract_names_from_c_array(cur_c, "g_CursorMethods")
    cur_members = _extract_names_from_c_array(cur_c, "g_CursorMembers")
    cur_getset = _extract_names_from_c_array(cur_c, "g_CursorCalcMembers")
    mod_methods = _extract_names_from_c_array(mod_c, "g_ModuleMethods")
    exceptions = re.findall(r'SetException\(module,\s*&g_\w+,\s*"([^"]+)"', mod_c)

    conn_getset = [n for n in conn_getset if not n.startswith("DSQL_ATTR_")]

    checks = {
        "connect keywords": connect_keywords,
        "Connection methods": conn_methods,
        "Connection members": conn_members,
        "Connection get/set attrs": conn_getset,
        "Cursor methods": cur_methods,
        "Cursor members": cur_members,
        "Cursor get/set attrs": cur_getset,
        "module methods": mod_methods,
        "exceptions": exceptions,
    }
    errors: list[str] = []
    for section, tokens in checks.items():
        missing = _missing_tokens(api_doc, tokens)
        if missing:
            errors.append(f"{section}: {missing}")
    if errors:
        _fail("API docs coverage missing entries:\n" + "\n".join(errors))


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
    _check_docs_structure()
    _check_api_docs_coverage()
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
