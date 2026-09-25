#!/usr/bin/env python3
"""Check that a source archive is safe to distribute and can locate its Go module."""

from __future__ import annotations

import sys
import tarfile
from pathlib import Path


REQUIRED = {
    "dpi_bridge/go.mod",
    "dpi_bridge/go.sum",
    "dpi_bridge/third_party/chunanyong_dm/go.mod",
    "dpi_bridge/third_party/chunanyong_dm/go.sum",
}


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: check_sdist_contents.py <source-archive.tar.gz>")

    archive = Path(sys.argv[1])
    with tarfile.open(archive, "r:gz") as source:
        paths = {Path(*Path(name).parts[1:]).as_posix() for name in source.getnames()}

    leaked = sorted(path for path in paths if path == "dpi_include" or path.startswith("dpi_include/"))
    missing = sorted(REQUIRED - paths)
    if leaked or missing:
        if leaked:
            print("[FAIL] source archive contains local DPI headers:", ", ".join(leaked))
        if missing:
            print("[FAIL] source archive is missing Go module files:", ", ".join(missing))
        raise SystemExit(1)

    print("[OK] source archive excludes DPI headers and includes Go module files")


if __name__ == "__main__":
    main()
