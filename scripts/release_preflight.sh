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
python3 -m pip install --quiet pyyaml build delocate

echo "[INFO] workflow YAML syntax"
python3 scripts/check_workflow_yaml.py

echo "[INFO] version and patch consistency"
python3 scripts/check_version_consistency.py
python3 scripts/check_third_party_patch.py

EXPECTED_VERSION="$(python3 scripts/check_version_consistency.py --print-version)"
echo "[INFO] expected version: $EXPECTED_VERSION"

echo "[INFO] clean previous wheel artifacts"
rm -rf dist dist_fixed

echo "[INFO] build wheel"
python3 -m build --wheel

mkdir -p dist_fixed
DYLD_LIBRARY_PATH=dpi_bridge delocate-wheel -w dist_fixed dist/*.whl -v

echo "[INFO] verify wheel import in isolated env"
python3 -m venv /tmp/dmpython_release_preflight_venv
WHEEL_PATHS=("$ROOT_DIR"/dist_fixed/*.whl)
/tmp/dmpython_release_preflight_venv/bin/pip install --quiet "${WHEEL_PATHS[@]}"
ACTUAL_VERSION=$(cd /tmp && /tmp/dmpython_release_preflight_venv/bin/python - <<'PY'
import dmPython
print(dmPython.version)
PY
)
rm -rf /tmp/dmpython_release_preflight_venv

if [[ "$ACTUAL_VERSION" != "$EXPECTED_VERSION" ]]; then
  echo "[FAIL] wheel runtime version mismatch: $ACTUAL_VERSION != $EXPECTED_VERSION"
  exit 2
fi

echo "[OK] release preflight passed for version $EXPECTED_VERSION"
if [[ -n "$TAG_NAME" ]]; then
  echo "[OK] tag format validated: $TAG_NAME"
fi
