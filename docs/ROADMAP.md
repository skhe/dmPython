# dmPython Improvement Roadmap (2026Q2)

This document is the single source of truth (SSOT) for the project-level improvement roadmap and phase status.

## Status Legend

- `NOT_STARTED`: Planned but not started.
- `IN_PROGRESS`: Actively being implemented.
- `DONE`: Completed and verified.
- `BLOCKED`: Temporarily blocked by dependency or environment.

## Phase Overview

| Phase | Window | Status | Owner | Exit Criteria | Last Updated | Evidence Links |
| --- | --- | --- | --- | --- | --- | --- |
| Phase 1 | Week 1-2 | IN_PROGRESS | Maintainers | README/README_zh positioning updated, ROADMAP established, roadmap status check wired into CI | 2026-03-04 | - |
| Phase 2 | Week 3-4 | NOT_STARTED | Maintainers | `requires_dm` regression green, no crash/139, P0/P1 contract coverage strengthened | 2026-03-04 | - |
| Phase 3 | Week 5-6 | NOT_STARTED | Maintainers | Release preflight and asset verification stable, tag release idempotency remains green | 2026-03-04 | - |
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

Pending.

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

Pending.

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

### Completion Notes

Pending.

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
