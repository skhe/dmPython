#!/usr/bin/env python3
"""Prepare and verify a complete local release bundle without publishing it."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import tomllib
from pathlib import Path

from check_sdist_contents import check_archive


ROOT = Path(__file__).resolve().parents[1]
PYTHON_TAGS = {"cp39", "cp310", "cp311", "cp312", "cp313"}


def project_version() -> str:
    with (ROOT / "pyproject.toml").open("rb") as source:
        return tomllib.load(source)["project"]["version"]


def release_files(directory: Path, tag: str) -> tuple[list[Path], Path]:
    version = project_version()
    if tag != f"v{version}":
        raise ValueError(f"tag {tag!r} does not match project version {version}")

    wheels = sorted(directory.glob("*.whl"))
    pattern = re.compile(
        rf"^dmpython_macos-{re.escape(version)}-(cp(?:39|310|311|312|313))-\1-"
        r"macosx_\d+_\d+_arm64\.whl$"
    )
    found_tags = []
    for wheel in wheels:
        match = pattern.fullmatch(wheel.name)
        if match is None:
            raise ValueError(f"unexpected wheel: {wheel.name}")
        found_tags.append(match.group(1))
    if len(wheels) != 5 or set(found_tags) != PYTHON_TAGS:
        raise ValueError(f"expected one wheel per Python tag {sorted(PYTHON_TAGS)}; found {found_tags}")

    source_archive = directory / f"dmpython_macos-{version}.tar.gz"
    if not source_archive.is_file():
        raise ValueError(f"missing source archive: {source_archive.name}")
    other_archives = sorted(path.name for path in directory.glob("*.tar.gz") if path != source_archive)
    if other_archives:
        raise ValueError(f"unexpected source archives: {other_archives}")
    check_archive(source_archive)
    return wheels, source_archive


def checksum_lines(files: list[Path]) -> str:
    lines = []
    for path in sorted(files, key=lambda item: item.name):
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        lines.append(f"{digest}  {path.name}")
    return "\n".join(lines) + "\n"


def prepare(directory: Path, tag: str) -> None:
    wheels, source_archive = release_files(directory, tag)
    metadata = {
        "tag": tag,
        "commit": os.getenv("GITHUB_SHA", ""),
        "workflow": os.getenv("GITHUB_WORKFLOW", "local rehearsal"),
        "run_id": os.getenv("GITHUB_RUN_ID", ""),
        "run_attempt": os.getenv("GITHUB_RUN_ATTEMPT", ""),
        "wheels": [wheel.name for wheel in wheels],
        "sdist": source_archive.name,
    }
    (directory / "build-metadata.json").write_text(
        json.dumps(metadata, indent=2, sort_keys=True) + "\n", encoding="utf-8"
    )
    (directory / "checksums.txt").write_text(
        checksum_lines([*wheels, source_archive]), encoding="utf-8"
    )


def verify(directory: Path, tag: str, assets_json: Path | None = None) -> None:
    wheels, source_archive = release_files(directory, tag)
    metadata = json.loads((directory / "build-metadata.json").read_text(encoding="utf-8"))
    if metadata["tag"] != tag or metadata["wheels"] != [wheel.name for wheel in wheels]:
        raise ValueError("release metadata does not match tag and wheel set")
    if metadata["sdist"] != source_archive.name:
        raise ValueError("release metadata does not match source archive")

    expected_checksums = checksum_lines([*wheels, source_archive])
    if (directory / "checksums.txt").read_text(encoding="utf-8") != expected_checksums:
        raise ValueError("checksums.txt does not match release files")

    if assets_json is not None:
        release = json.loads(assets_json.read_text(encoding="utf-8"))
        asset_names = [asset["name"] for asset in release["assets"]]
        expected_names = {path.name for path in [*wheels, source_archive]}
        expected_names.update({"checksums.txt", "build-metadata.json"})
        missing = sorted(expected_names - set(asset_names))
        if missing:
            raise ValueError(f"release is missing assets: {missing}")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=("prepare", "verify"))
    parser.add_argument("directory", type=Path)
    parser.add_argument("--tag", required=True)
    parser.add_argument("--assets-json", type=Path, help="gh release view --json assets output")
    args = parser.parse_args()
    try:
        if args.operation == "prepare":
            if args.assets_json is not None:
                parser.error("--assets-json only applies to verify")
            prepare(args.directory, args.tag)
        else:
            verify(args.directory, args.tag, args.assets_json)
    except (KeyError, OSError, ValueError) as exc:
        parser.exit(1, f"[FAIL] {exc}\n")
    print(f"[OK] {args.operation} release bundle for {args.tag}")


if __name__ == "__main__":
    main()
