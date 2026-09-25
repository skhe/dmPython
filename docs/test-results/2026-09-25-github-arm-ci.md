# GitHub ARM CI validation

Branch: `codex/dmpython-ci-regression`. Pull request: [#3](https://github.com/skhe/dmPython/pull/3).

The [pull-request run](https://github.com/skhe/dmPython/actions/runs/36089025280) passed the pinned ARM DM8 development container regression, five macOS ARM wheel build/install jobs for CPython 3.9–3.13, and the unified CI gate. Its JUnit artifact contains **40 tests, 0 failures, 0 errors, 0 skipped**. [Workflow Lint](https://github.com/skhe/dmPython/actions/runs/36089025066) also passed.

The [manually triggered full run](https://github.com/skhe/dmPython/actions/runs/36089236861) exercised **66 real-database tests, 0 failures, 0 errors, 0 skipped** on the GitHub ARM runner, including P0, P1, and P2. Two local-only tests were outside the `requires_dm` selection.

The local [official-image Orb baseline](2026-09-25-orb-arm-baseline.md) passed 68 tests after the behavior fixes on macOS ARM / CPython 3.10. The complete real-database suite against the CI image passed locally with 66 selected tests and 2 non-database tests deselected. A deliberate skip under `--require-dm` exited with code 1.

The CI image is published by a third party and pinned by digest in the workflow. The vendor-hosted official image URL returned HTTP 403 on GitHub-hosted runners, so the official image remains the local comparison baseline. No public release is enabled while the Go driver's distribution rights remain unresolved.
