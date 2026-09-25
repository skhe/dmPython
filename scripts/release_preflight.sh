#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

TAG_NAME="${1:-${TAG_NAME:-}}"
if [[ -n "$TAG_NAME" ]]; then
  if [[ ! "$TAG_NAME" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "[FAIL] invalid tag format: $TAG_NAME (expected vX.Y.Z)"
    exit 2
  fi
fi

echo "[INFO] install preflight dependencies"
if ! python3 -c 'import yaml, build, delocate, twine' >/dev/null 2>&1; then
  python3 -m pip install --quiet pyyaml build delocate twine
fi

echo "[INFO] workflow YAML syntax"
python3 scripts/check_workflow_yaml.py

echo "[INFO] version and patch consistency"
python3 scripts/check_version_consistency.py
python3 scripts/check_third_party_patch.py

EXPECTED_VERSION="$(python3 scripts/check_version_consistency.py --print-version)"
echo "[INFO] expected version: $EXPECTED_VERSION"

OUTPUT_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dmpython-preflight.XXXXXX")"
mkdir -p "$OUTPUT_DIR/wheels" "$OUTPUT_DIR/fixed" "$OUTPUT_DIR/sdist"
echo "[INFO] preflight output: $OUTPUT_DIR"

echo "[INFO] build wheel"
MACOSX_DEPLOYMENT_TARGET=14.0 _PYTHON_HOST_PLATFORM=macosx-14.0-arm64 \
  python3 -m build --wheel --outdir "$OUTPUT_DIR/wheels"

echo "[INFO] build and check source archive"
python3 -m build --sdist --outdir "$OUTPUT_DIR/sdist"
python3 -m twine check "$OUTPUT_DIR"/sdist/*.tar.gz
python3 scripts/check_sdist_contents.py "$OUTPUT_DIR"/sdist/*.tar.gz

DYLD_LIBRARY_PATH=dpi_bridge delocate-wheel -w "$OUTPUT_DIR/fixed" "$OUTPUT_DIR"/wheels/*.whl -v

echo "[INFO] verify wheel import in isolated env"
python3 -m venv "$OUTPUT_DIR/venv"
WHEEL_PATHS=("$OUTPUT_DIR"/fixed/*.whl)
"$OUTPUT_DIR/venv/bin/pip" install --quiet "${WHEEL_PATHS[@]}"
ACTUAL_VERSION=$(cd /tmp && "$OUTPUT_DIR/venv/bin/python" - <<'PY'
import dmPython
print(dmPython.version)
PY
)
rm -rf "$OUTPUT_DIR/venv"

if [[ "$ACTUAL_VERSION" != "$EXPECTED_VERSION" ]]; then
  echo "[FAIL] wheel runtime version mismatch: $ACTUAL_VERSION != $EXPECTED_VERSION"
  exit 2
fi

echo "[OK] release preflight passed for version $EXPECTED_VERSION"
if [[ -n "$TAG_NAME" ]]; then
  echo "[OK] tag format validated: $TAG_NAME"
fi
