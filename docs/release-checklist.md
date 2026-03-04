# Release Checklist

## 1. Preflight
- [ ] Run `./scripts/release_preflight.sh vX.Y.Z`
- [ ] Confirm workflow lint and actionlint checks are green
- [ ] Confirm version consistency (`pyproject.toml`, `setup.py`, `src/native/py_Dameng.h`, `dmPython.version`)
- [ ] Confirm third-party patch checks pass (`scripts/check_third_party_patch.py`)

## 2. Regression
- [ ] Run `DYLD_LIBRARY_PATH=/Users/skhe/projects/dmPython/dpi_bridge python3 -m pytest -q -m requires_dm tests`
- [ ] Confirm P0/P1/P2 integration markers are green
- [ ] Confirm no `Segmentation fault` / no `139/-11` exits

## 3. Tag & CI
- [ ] Push release tag `vX.Y.Z`
- [ ] Confirm `Build macOS wheels` workflow succeeds for all Python targets (`cp39/cp310/cp311/cp312/cp313`)
- [ ] Confirm release step is idempotent (re-run does not fail)

## 4. Release Assets
- [ ] Confirm 5 arm64 wheel assets exist on release page
- [ ] Confirm `checksums.txt` is attached
- [ ] Confirm `build-metadata.json` is attached
- [ ] Spot-check one wheel install and `import dmPython`

## 5. Post-release
- [ ] Update `CHANGELOG.md` if needed
- [ ] Verify release notes and links are correct
- [ ] Record any incidents/fixes back into `PATCHES.md` or docs
