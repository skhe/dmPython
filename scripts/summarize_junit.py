"""Validate and summarize a real-database pytest JUnit report."""

import argparse
import json
import xml.etree.ElementTree as ET
from pathlib import Path


def summarize(report: Path) -> dict[str, float | int]:
    root = ET.parse(report).getroot()
    suites = [root] if root.tag == "testsuite" else root.findall("testsuite")
    if not suites:
        raise ValueError("JUnit report has no test suite")
    result: dict[str, float | int] = {
        "tests": 0,
        "failures": 0,
        "errors": 0,
        "skipped": 0,
        "pytest_reported_seconds": 0.0,
    }
    for suite in suites:
        for key in ("tests", "failures", "errors", "skipped"):
            result[key] += int(suite.attrib[key])
        result["pytest_reported_seconds"] += float(suite.attrib["time"])
    if result["tests"] <= 0:
        raise ValueError("JUnit report contains no tests")
    if result["failures"] or result["errors"] or result["skipped"]:
        raise ValueError(f"Real-database regression is incomplete: {result}")
    return result


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("report", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    result = summarize(args.report)
    if args.output:
        args.output.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(result))


if __name__ == "__main__":
    main()
