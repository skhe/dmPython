# dmPython Roadmap

This document is the single source of truth (SSOT) for the project-level improvement roadmap and phase status. Updated 2026-09-25.

The current open-source track follows [Stage 0: scope and release prerequisites](plans/2026-09-24-open-source-stage-0.md). The four phases below preserve the earlier improvement plan; their release and maintenance work is sequenced against the current track.

## Status Legend

- `NOT_STARTED`: Planned but not started.
- `IN_PROGRESS`: Actively being implemented.
- `DONE`: Completed and verified.
- `BLOCKED`: Temporarily blocked by dependency or environment.

## Current open-source track (2026Q4)

The next objective is a repeatable, publicly distributable release. The [five-version full run](https://github.com/skhe/dmPython/actions/runs/36094943587) passed 66 real-DM tests on each of CPython 3.9–3.13; [PR #5](https://github.com/skhe/dmPython/pull/5) also passed five macOS ARM wheel builds and installation checks. A [non-public local release rehearsal](test-results/2026-09-25-five-python-release-rehearsal.md) validated five actual wheels, sdist, checksums and metadata. Public wheel upload and GitHub Release remain disabled while the separately included Go driver's distribution rights are unresolved.

Status: **3 done, 2 in progress, 1 blocked, 2 not started**.

| Horizon | Initiative | Status | Owner | Target / exit evidence | Dependency |
| --- | --- | --- | --- | --- | --- |
| Done | Establish the GitHub CI baseline: required real-DM regression, wheel matrix, lint and branch protection | DONE | Maintainers | [Merged PR #3](https://github.com/skhe/dmPython/pull/3), [main CI](https://github.com/skhe/dmPython/actions/runs/36089633170) and [lint](https://github.com/skhe/dmPython/actions/runs/36089632930) green | Pinned third-party DM8 development image; DPI header secret for trusted-branch wheel jobs |
| Now | Resolve distribution rights for the included Go driver and bridge binary, or replace that component | IN_PROGRESS | Maintainers (investigation and outreach) | Before any public sdist/wheel upload: record applicable terms or written permission covering source, modifications and binaries; otherwise land and regress a permitted replacement | [License investigation](plans/2026-09-24-go-driver-license-research.md); external answer may delay release |
| Now | Preserve an official-DM comparison baseline and keep nightly CI usable | IN_PROGRESS | Maintainers | Before **2026-10-09**, renew the local official-image trial or replace it with a valid official test installation; continue recording nightly results and image digest/server-version changes | Local official trial expiry; third-party CI image provenance and its publisher-stated expiry |
| Done | Expand real-DM behavior coverage across supported CPython versions | DONE | Maintainers | [GitHub full run](https://github.com/skhe/dmPython/actions/runs/36094943587): 66 tests per Python version, no failures/errors/skips; [version/server report](test-results/2026-09-25-five-python-release-rehearsal.md) | Pinned development image; one DM8 server version so far |
| Done | Rehearse release preparation without publishing | DONE | Maintainers | [Local rehearsal](test-results/2026-09-25-five-python-release-rehearsal.md): preflight, five real wheels, sdist contents, checksums, metadata, tag/version validation and deterministic rerun | Remote GitHub Release upsert remains a Phase 3 acceptance item after rights clearance |
| Next | Publish the first GitHub Release and PyPI package through gated automation | BLOCKED | Maintainers | After rights and regression gates pass: verified tagged release, five wheels and sdist on approved channels; PyPI trusted publishing and a tested recovery procedure | Go component rights or replacement; cross-version behavior results; release rehearsal; PyPI project-name check |
| Later | Let outside contributors verify macOS wheel builds | NOT_STARTED | Maintainers | Fork PRs obtain the required DPI headers through an approved, reproducible path and run the same wheel checks as trusted branches | SDK/header distribution terms; current fork PRs only receive the real-DM gate |
| Later | Document repeatable upstream sync and widen supported DM8 versions | NOT_STARTED | Maintainers | Patch drift guard plus sync playbook; compatibility results for additional DM8 releases | Phase 4 governance work; access to additional database versions |

### Priority and dependencies

The Go component's rights and real-database behavior can advance in parallel. The release rehearsal can also proceed locally, but publishing stays blocked until both gates pass. Keep `DMPYTHON_RELEASE_ENABLED` unset until the distribution decision is recorded. The official local trial expires on 2026-10-09; its renewal is the only fixed near-term date, while the other horizons are sequencing targets rather than release promises. A single DM8 instance on OrbStack remains sufficient for the official comparison baseline; the hosted ARM runner uses a fresh pinned development container for CI.

This update moves five-version real-database coverage and local release preparation into the completed baseline. The Go rights decision, official test continuity and remote release verification remain on the critical path. Frequent automatic releases are a later operating practice once each release can pass the same gates; version count is not a delivery target.

## Phase Overview

| Phase | Window | Status | Owner | Exit Criteria | Last Updated | Evidence Links |
| --- | --- | --- | --- | --- | --- | --- |
| Phase 1 | Week 1-2 | DONE | Maintainers | README/README_zh positioning updated, ROADMAP established, roadmap status check wired into CI | 2026-09-24 | [64152ae](https://github.com/skhe/dmPython/commit/64152ae) · [lint](https://github.com/skhe/dmPython/actions/runs/36018070212) · [wheels](https://github.com/skhe/dmPython/actions/runs/36018069837) |
| Phase 2 | Week 3-4 | DONE | Maintainers | `requires_dm` regression green, no crash/139, P0/P1 contract coverage strengthened | 2026-09-25 | [GitHub full regression](https://github.com/skhe/dmPython/actions/runs/36089236861) · [GitHub PR gate](https://github.com/skhe/dmPython/actions/runs/36089025280) · [Local baseline](test-results/2026-09-25-orb-arm-baseline.md) |
| Phase 3 | Week 5-6 | IN_PROGRESS | Maintainers | Release preflight and asset verification stable, tag release idempotency remains green | 2026-09-25 | [Gated workflow](../.github/workflows/build-wheels.yml) · [Release checklist](release-checklist.md) |
| Phase 4 | Week 7-8 | NOT_STARTED | Maintainers | Third-party patch drift guard and upstream sync governance are documented and enforced | 2026-03-04 | - |

## Phase 1 (Week 1-2): Positioning and Doc Governance

### Goal

Clarify project positioning and make roadmap state tracking enforceable in repository workflow.

### Scope

- In scope:
  - README and README_zh production usage notice and support policy.
  - Roadmap SSOT document with status table and phase details.
  - CI gate for roadmap status consistency checks.
- Out of scope:
  - Runtime feature implementation.
  - DB-API behavior changes.

### Deliverables

- Updated `README.md` and `docs/README_zh.md`.
- New `docs/ROADMAP.md`.
- New `scripts/check_roadmap_status.py`.
- Workflow integration in `.github/workflows/workflow-lint.yml`.

### Acceptance Criteria

- Positioning statements are present in both language entry docs.
- Roadmap status table contains Phase 1-4 with valid status/date values.
- CI lint workflow executes roadmap consistency check.

### Risks and Rollback

- Risk: Documentation drift between English and Chinese docs.
- Mitigation: Keep summary status in README and SSOT details in ROADMAP only.
- Rollback: Revert doc-only commits; no runtime impact.

### Required Status Updates On Completion

- Set `Status` to `DONE` in the phase overview row.
- Update `Last Updated`.
- Add evidence links (PR/commit/workflow run).
- Add a 3-5 line completion summary in this section.

### Completion Notes

- README.md and docs/README_zh.md carry the production usage notice, support policy, and a phase snapshot mirroring this table.
- docs/ROADMAP.md is the phase SSOT: 4 phases with status vocabulary, dates, and evidence links.
- scripts/check_roadmap_status.py runs inside the workflow-lint pipeline, so stale status metadata fails CI.
- Verified on 64152ae: Workflow Lint and Build macOS wheels both green.

## Phase 2 (Week 3-4): Test Coverage and Stability Convergence

### Goal

Strengthen P0/P1 contract coverage and converge integration stability signals.

### Scope

- In scope:
  - Expand tests for high-risk runtime and contract paths.
  - Normalize regression failure signaling.
- Out of scope:
  - New external API features.

### Deliverables

- Additional P0/P1 integration coverage and assertions.
- Updated CI failure taxonomy documentation.

### Acceptance Criteria

- `requires_dm` full regression green in configured environments.
- No segmentation faults, no process exit `139/-11`.

### Risks and Rollback

- Risk: Environment instability causes noisy CI.
- Mitigation: Maintain explicit skip reasons and deterministic markers.
- Rollback: Revert newly added unstable tests while retaining bug reproducer cases.

### Required Status Updates On Completion

- Set `Status` to `DONE`.
- Update `Last Updated`.
- Add evidence links.
- Add completion notes.

### Completion Notes

The GitHub ARM runner completed all 66 selected real-database tests with no failures, errors, skips, or crash exits. The PR gate completed 40 P0/P1 tests and five macOS ARM wheel build/install jobs for Python 3.9–3.13. New DB-API cases cover parameter counts, integrity-error classification, result metadata, and rollback on close. The local official-image baseline and the pinned CI development image both passed the full selected suite. The [CI evidence](test-results/2026-09-25-github-arm-ci.md) records the image-source limitation and result scope.

## Phase 3 (Week 5-6): Release Quality and Traceability

### Goal

Ensure release outputs are complete, repeatable, and verifiable.

### Scope

- In scope:
  - Strengthen release preflight checks and output verification.
  - Keep release process idempotent for existing tags.
- Out of scope:
  - Publishing to channels outside GitHub Releases.

### Deliverables

- Stable release preflight and asset completeness checks.
- Consistent checksums and metadata artifact workflow.

### Acceptance Criteria

- Tag release can be re-run without failure.
- All expected wheel targets and metadata assets are present.

### Risks and Rollback

- Risk: CI behavior changes due to action updates.
- Mitigation: Pin key actions where possible and monitor failures.
- Rollback: Restore previous release workflow while preserving verification scripts.

### Required Status Updates On Completion

- Set `Status` to `DONE`.
- Update `Last Updated`.
- Add evidence links.
- Add completion notes.

### Progress Notes

The workflow now requires a real-DM pass and all five wheel builds before its tag release job can run. It verifies the wheel set and prepares checksums and build metadata. Public artifact upload is disabled pending the Go component's distribution decision; tag rerun and published-asset completeness are not yet verified. Continue with a non-public rehearsal, then complete the release acceptance criteria after the rights gate clears.

## Phase 4 (Week 7-8): Maintainability and Upstream Sync Governance

### Goal

Reduce long-term maintenance risk for patched third-party dependencies and upstream alignment.

### Scope

- In scope:
  - Patch governance checks and synchronization policy.
  - Documented upstream sync cadence and conflict handling procedure.
- Out of scope:
  - Major architecture rewrites.

### Deliverables

- Policy and scripts that detect patch drift.
- Written upstream sync and conflict playbook.

### Acceptance Criteria

- Patch drift is detectible by CI before release.
- Sync process is documented and repeatable.

### Risks and Rollback

- Risk: Upstream delta introduces hard-to-merge changes.
- Mitigation: Keep patch surface minimal and traceable.
- Rollback: Pin to validated snapshots until sync conflict is resolved.

### Required Status Updates On Completion

- Set `Status` to `DONE`.
- Update `Last Updated`.
- Add evidence links.
- Add completion notes.

### Completion Notes

Pending.
