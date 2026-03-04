#!/usr/bin/env python3
"""Validate local patched DM driver contract and patch docs."""
from __future__ import annotations

import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
GO_MOD = ROOT / "dpi_bridge/go.mod"
PATCH_DOC = ROOT / "dpi_bridge/third_party/chunanyong_dm/PATCHES.md"
PATCH_FILE = ROOT / "dpi_bridge/third_party/chunanyong_dm/a.go"


def fail(msg: str) -> None:
    raise RuntimeError(msg)


def require_contains(path: Path, needle: str) -> None:
    text = path.read_text(encoding="utf-8")
    if needle not in text:
        fail(f"{path} missing expected content: {needle}")


def main() -> int:
    if not GO_MOD.exists():
        fail(f"missing {GO_MOD}")
    if not PATCH_DOC.exists():
        fail(f"missing {PATCH_DOC}")
    if not PATCH_FILE.exists():
        fail(f"missing {PATCH_FILE}")

    require_contains(GO_MOD, "replace gitee.com/chunanyong/dm => ./third_party/chunanyong_dm")

    # Patch documentation should describe the actual patch point and its tests.
    require_contains(PATCH_DOC, "File: `a.go`")
    require_contains(PATCH_DOC, "Function: `dm_build_610`")
    require_contains(PATCH_DOC, "test_clob_unicode_problem_patterns_roundtrip")
    require_contains(PATCH_DOC, "test_clob_unicode_problem_patterns_subprocess_no_crash")

    # Patch implementation invariants for unicode chunk-boundary fix.
    require_contains(PATCH_FILE, "dm_build_610(")
    require_contains(PATCH_FILE, "var dm_build_615 bytes.Buffer")
    require_contains(PATCH_FILE, "Avoid splitting a UTF-8 sequence at chunk boundaries.")

    print("[OK] third-party patch consistency checks passed")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except RuntimeError as exc:
        print(f"[FAIL] {exc}")
        raise SystemExit(2)
