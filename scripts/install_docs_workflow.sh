#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$ROOT_DIR/docs/ci/docs-pages.workflow.yml"
DST_DIR="$ROOT_DIR/.github/workflows"
DST="$DST_DIR/docs-pages.yml"

if [[ ! -f "$SRC" ]]; then
  echo "missing template: $SRC" >&2
  exit 1
fi

mkdir -p "$DST_DIR"
cp "$SRC" "$DST"

echo "installed: $DST"

echo "next:"
echo "  git add .github/workflows/docs-pages.yml"
echo "  git commit -m 'ci(docs): add pages workflow'"
echo "  git push"
