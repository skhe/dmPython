"""Checks for the release bundle used by rehearsal and GitHub Releases."""

from __future__ import annotations

import io
import json
import sys
import tarfile
from pathlib import Path

import pytest


sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from check_sdist_contents import REQUIRED  # noqa: E402
from release_assets import prepare, project_version, verify  # noqa: E402


def bundle(tmp_path: Path, *, leaked_header: bool = False) -> tuple[Path, str]:
    version = project_version()
    for tag in ("cp39", "cp310", "cp311", "cp312", "cp313"):
        (tmp_path / f"dmpython_macos-{version}-{tag}-{tag}-macosx_14_0_arm64.whl").write_bytes(
            tag.encode()
        )

    archive = tmp_path / f"dmpython_macos-{version}.tar.gz"
    paths = set(REQUIRED)
    if leaked_header:
        paths.add("dpi_include/DPI.h")
    with tarfile.open(archive, "w:gz") as source:
        for path in sorted(paths):
            contents = b"test"
            entry = tarfile.TarInfo(f"dmpython_macos-{version}/{path}")
            entry.size = len(contents)
            source.addfile(entry, io.BytesIO(contents))
    return tmp_path, f"v{version}"


def test_release_bundle_is_complete_and_repeatable(tmp_path: Path) -> None:
    directory, tag = bundle(tmp_path)
    prepare(directory, tag)
    first_checksums = (directory / "checksums.txt").read_bytes()
    first_metadata = (directory / "build-metadata.json").read_bytes()
    prepare(directory, tag)
    assert (directory / "checksums.txt").read_bytes() == first_checksums
    assert (directory / "build-metadata.json").read_bytes() == first_metadata
    verify(directory, tag)

    metadata = json.loads(first_metadata)
    names = metadata["wheels"] + [metadata["sdist"], "checksums.txt", "build-metadata.json"]
    assets = tmp_path / "release-assets.json"
    assets.write_text(json.dumps({"assets": [{"name": name} for name in names]}))
    verify(directory, tag, assets)
    assets.write_text(json.dumps({"assets": [{"name": name} for name in names[:-1]]}))
    with pytest.raises(ValueError, match="missing assets"):
        verify(directory, tag, assets)


def test_missing_wheel_and_wrong_tag_fail(tmp_path: Path) -> None:
    directory, tag = bundle(tmp_path)
    with pytest.raises(ValueError, match="does not match project version"):
        prepare(directory, "v0.0.0")
    next(directory.glob("*cp39*.whl")).unlink()
    with pytest.raises(ValueError, match="expected one wheel"):
        prepare(directory, tag)


def test_changed_wheel_fails_checksum_verification(tmp_path: Path) -> None:
    directory, tag = bundle(tmp_path)
    prepare(directory, tag)
    next(directory.glob("*cp39*.whl")).write_bytes(b"changed")
    with pytest.raises(ValueError, match="checksums.txt"):
        verify(directory, tag)


def test_source_archive_with_headers_fails(tmp_path: Path) -> None:
    directory, tag = bundle(tmp_path, leaked_header=True)
    with pytest.raises(ValueError, match="DPI headers"):
        prepare(directory, tag)
