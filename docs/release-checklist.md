# Release Checklist

## 0. Public distribution gate

- [ ] Include the full Mulan PSL v2 text and preserve upstream notices; confirm the separately vendored Go driver's license or approved replacement and the resulting bridge binary (see [Stage 0](plans/2026-09-24-open-source-stage-0.md))
- [ ] Confirm the isolated DM test account and full integration run are recorded; a CI skip is not a pass
- [ ] Confirm the tag's `real-dm` job passed against the pinned DM8 development container; startup, connection, test failure, or skip must fail the job
- [ ] Set the GitHub repository variable `DMPYTHON_RELEASE_ENABLED=true` only after the Go component's distribution rights are documented; leave it unset while unresolved
- [ ] Confirm the final sdist excludes local DPI headers and the final wheel contains only approved components
- [ ] Confirm DPI headers and the database image were not uploaded as workflow artifacts
- [ ] Recheck PyPI project name and target version before any upload

## 1. Preflight

- [ ] Run `./scripts/release_preflight.sh vX.Y.Z`
- [ ] Confirm workflow lint and actionlint checks are green
- [ ] Confirm version consistency (`pyproject.toml`, `setup.py`, `src/native/py_Dameng.h`, `dmPython.version`)
- [ ] Confirm third-party patch checks pass (`scripts/check_third_party_patch.py`)

## 2. Regression

- [ ] Run `DYLD_LIBRARY_PATH="$PWD/dpi_bridge" python -m pytest -q --require-dm -m requires_dm tests` with an isolated test database
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
- [ ] Confirm `docs/ROADMAP.md` phase status is synchronized with the latest completed milestone (status/date/evidence links)
