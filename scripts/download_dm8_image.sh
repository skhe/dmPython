#!/usr/bin/env bash
set -euo pipefail

archive_path="${1:?pass the output archive path}"
image_url='https://download.dameng.com/product/DM8/Docker/mouthversion/2025.03/dm8_20250924_HWarm_kylin10_sp1_64_ent_8.1.4.80_pack29/dm8_20250924_HWarm_kylin10_sp1_64_rq_ent_8.1.4.80_pack29.tar'
expected_sha256='f08359576241c17110446aa3e19794650b95add43840a5da9d5070240b1e53c8'

curl -fsSL --retry 3 --max-time 900 "$image_url" -o "$archive_path"
python3 - "$archive_path" "$expected_sha256" <<'PY'
import hashlib
import sys

archive_path, expected = sys.argv[1:]
digest = hashlib.sha256()
with open(archive_path, "rb") as archive:
    for chunk in iter(lambda: archive.read(4 * 1024 * 1024), b""):
        digest.update(chunk)
if digest.hexdigest() != expected:
    raise SystemExit("Official DM8 image checksum mismatch")
print("Official DM8 image checksum verified")
PY
