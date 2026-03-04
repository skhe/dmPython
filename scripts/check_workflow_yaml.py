#!/usr/bin/env python3
"""Validate GitHub workflow YAML syntax."""
from __future__ import annotations

from pathlib import Path
import sys

import yaml


def main() -> int:
    workflow_dir = Path(__file__).resolve().parents[1] / ".github" / "workflows"
    files = sorted(workflow_dir.glob("*.yml")) + sorted(workflow_dir.glob("*.yaml"))
    if not files:
        print("No workflow files found.")
        return 1

    ok = True
    for wf in files:
        try:
            yaml.safe_load(wf.read_text(encoding="utf-8"))
            print(f"[OK] {wf}")
        except Exception as exc:
            ok = False
            print(f"[FAIL] {wf}: {exc}")

    return 0 if ok else 2


if __name__ == "__main__":
    sys.exit(main())
