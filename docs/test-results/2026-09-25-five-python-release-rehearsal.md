# Five-Python real-DM regression and local release rehearsal

Verified 2026-09-25 on PR [#5](https://github.com/skhe/dmPython/pull/5). The source commit for the full run was `d51710c`. Public publishing stayed disabled: `DMPYTHON_RELEASE_ENABLED` was unset, the PR run uploaded only real-DM reports, and no tag, GitHub Release or PyPI upload was created.

## Real database behavior

The [GitHub ARM full run](https://github.com/skhe/dmPython/actions/runs/36094943587) used the pinned community DM8 development image. Its reported server `ID_CODE` was `--03134284604-20260707-335949-20228`. Each job had a fresh container and test account; the downloaded per-version JUnit summaries reported:

| CPython | Full real-DM tests | Failures | Errors | Skips |
| --- | ---: | ---: | ---: | ---: |
| 3.9.25 | 66 | 0 | 0 | 0 |
| 3.10.21 | 66 | 0 | 0 | 0 |
| 3.11.16 | 66 | 0 | 0 | 0 |
| 3.12.14 | 66 | 0 | 0 | 0 |
| 3.13.15 | 66 | 0 | 0 | 0 |

The [PR gate](https://github.com/skhe/dmPython/actions/runs/36094936469) passed P0/P1 real-DM jobs for all five versions, five macOS ARM wheel build/install jobs, and the aggregate `CI gate`. [Workflow lint](https://github.com/skhe/dmPython/actions/runs/36094936214) also passed.

For an independent official-image comparison, the local OrbStack DM8 instance (`ID_CODE` `--03134284368-20250821-288894-20149 Pack29`) ran the same `--require-dm -m requires_dm tests` selection sequentially with macOS ARM CPython 3.9.6, 3.10.20, 3.11.15, 3.12.0 and 3.13.14. Each version passed **66 tests, 0 failures, 0 errors, 0 skips**; two non-database tests were deselected. The local official trial expires on 2026-10-09.

## Non-public release rehearsal

The local `release_preflight.sh v2.5.32` built and installed a wheel in an isolated environment, built an sdist, ran `twine check`, checked the sdist for DPI headers and required Go module files, and checked version consistency. The rehearsal then built actual `macosx_14_0_arm64` wheels for `cp39`, `cp310`, `cp311`, `cp312` and `cp313` with the local interpreters. These six files stayed under `/private/tmp/dmpython-release-bundle.7C5I10` and were not uploaded.

`scripts/release_assets.py prepare` validated the tag against `pyproject.toml`, all five wheel tags and the sdist contents, then generated `checksums.txt` and `build-metadata.json`. `verify` checked the bundle. Running `prepare` a second time produced byte-identical metadata and checksums. Their SHA-256 values were `1bc072bc099b9d012c3fbd1524b82d423fd611a63c5f19a0d24833f6674b972b` and `1dda31a847688015b88a5a530b7d04d89084da158a3988b1f4afbfc806c566b7`, respectively. All five wheels contained `libdmdpi.dylib` and no `DPI.h`.

The local preflight used installed build dependencies without a second isolated dependency download; the GitHub wheel jobs used their existing isolated build path and passed. This rehearsal verifies preparation through the upload boundary. GitHub Release creation, remote asset replacement on tag rerun, PyPI trusted publishing and recovery remain untested until distribution rights are resolved. Only one official and one community DM8 server version were exercised.
