#!/usr/bin/env python3
"""Validate roadmap phase status table consistency."""

from __future__ import annotations

import datetime as _dt
import re
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
ROADMAP = ROOT / "docs" / "ROADMAP.md"
ALLOWED_STATUS = {"NOT_STARTED", "IN_PROGRESS", "DONE", "BLOCKED"}
EXPECTED_PHASES = {"Phase 1", "Phase 2", "Phase 3", "Phase 4"}
TABLE_HEADER = [
    "Phase",
    "Window",
    "Status",
    "Owner",
    "Exit Criteria",
    "Last Updated",
    "Evidence Links",
]


def _fail(message: str) -> None:
    print(f"[FAIL] {message}")
    raise SystemExit(2)


def _parse_table_rows(text: str) -> list[dict[str, str]]:
    lines = text.splitlines()
    header_index = -1
    for idx, line in enumerate(lines):
        if line.strip().startswith("| Phase | Window | Status | Owner | Exit Criteria | Last Updated | Evidence Links |"):
            header_index = idx
            break
    if header_index < 0:
        _fail("Cannot find roadmap phase overview table header in docs/ROADMAP.md")

    rows: list[dict[str, str]] = []
    for line in lines[header_index + 1 :]:
        stripped = line.strip()
        if not stripped.startswith("|"):
            break
        parts = [p.strip() for p in stripped.split("|")[1:-1]]
        if not parts:
            continue
        if all(re.fullmatch(r"-+", p.replace(" ", "")) for p in parts):
            continue
        if len(parts) != len(TABLE_HEADER):
            _fail(f"Invalid table column count in row: {stripped}")
        row = dict(zip(TABLE_HEADER, parts))
        rows.append(row)
    return rows


def _validate_date(date_str: str, phase: str) -> None:
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", date_str):
        _fail(f"{phase} has invalid Last Updated date format: {date_str} (expected YYYY-MM-DD)")
    try:
        _dt.date.fromisoformat(date_str)
    except ValueError:
        _fail(f"{phase} has invalid Last Updated calendar date: {date_str}")


def _validate_evidence(status: str, evidence: str, phase: str) -> None:
    placeholder_values = {"", "-", "N/A", "n/a", "pending", "Pending"}
    if status == "DONE" and evidence.strip() in placeholder_values:
        _fail(f"{phase} is DONE but Evidence Links is empty/placeholder")


def main() -> None:
    if not ROADMAP.exists():
        _fail("Missing docs/ROADMAP.md")

    rows = _parse_table_rows(ROADMAP.read_text(encoding="utf-8"))
    if len(rows) != 4:
        _fail(f"Expected 4 phase rows, found {len(rows)}")

    seen_phases: set[str] = set()
    for row in rows:
        phase = row["Phase"]
        if phase in seen_phases:
            _fail(f"Duplicate phase row found: {phase}")
        seen_phases.add(phase)

        status = row["Status"]
        if status not in ALLOWED_STATUS:
            _fail(f"{phase} has invalid status: {status}")

        _validate_date(row["Last Updated"], phase)
        _validate_evidence(status, row["Evidence Links"], phase)

    if seen_phases != EXPECTED_PHASES:
        _fail(f"Phase set mismatch. expected={sorted(EXPECTED_PHASES)} found={sorted(seen_phases)}")

    print("[OK] roadmap status consistency checks passed")


if __name__ == "__main__":
    main()
