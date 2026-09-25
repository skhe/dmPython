"""Read DPI headers directly from an official Docker image archive."""

import argparse
import json
import shutil
import tarfile
from pathlib import Path


HEADER_NAMES = {"DPI.h", "DPIext.h", "DPItypes.h", "DPIucode.h"}
HEADER_PREFIX = "opt/dmdbms/drivers/dpi/include/"


def extract_headers(image_archive: Path, output_dir: Path) -> None:
    output_dir.mkdir(parents=True, exist_ok=True)
    found: set[str] = set()
    with tarfile.open(image_archive, "r:*") as image:
        manifest_file = image.extractfile("manifest.json")
        if manifest_file is None:
            raise ValueError("Docker archive has no manifest.json")
        manifest = json.load(manifest_file)
        if len(manifest) != 1:
            raise ValueError("Expected a single-image Docker archive")
        for layer_name in manifest[0]["Layers"]:
            layer_file = image.extractfile(layer_name)
            if layer_file is None:
                raise ValueError(f"Missing layer: {layer_name}")
            with tarfile.open(fileobj=layer_file, mode="r|*") as layer:
                for member in layer:
                    name = member.name.removeprefix("./")
                    if not member.isfile() or not name.startswith(HEADER_PREFIX):
                        continue
                    header_name = name.removeprefix(HEADER_PREFIX)
                    if header_name not in HEADER_NAMES:
                        continue
                    source = layer.extractfile(member)
                    if source is None:
                        raise ValueError(f"Cannot read {header_name}")
                    with (output_dir / header_name).open("wb") as target:
                        shutil.copyfileobj(source, target)
                    found.add(header_name)
    if found != HEADER_NAMES:
        raise ValueError(f"Missing DPI headers: {sorted(HEADER_NAMES - found)}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("archive", type=Path)
    parser.add_argument("output_dir", type=Path)
    args = parser.parse_args()
    extract_headers(args.archive, args.output_dir)
