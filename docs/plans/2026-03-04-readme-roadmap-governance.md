# README Positioning and Roadmap Governance Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Clarify project positioning for local validation vs production usage and enforce roadmap status governance in-repo.

**Architecture:** Documentation changes are centralized in README + docs/ROADMAP as the SSOT, with CI lint enforcement through a dedicated roadmap status checker script.

**Tech Stack:** Markdown docs, Python validation script, GitHub Actions workflow lint pipeline.

---

## Scope

- Update project positioning statements in English and Chinese entry docs.
- Add a roadmap SSOT document with 4 phases / 8 weeks and strict status fields.
- Add CI-verifiable checks to prevent stale roadmap status metadata.
- Add contributor and release process hooks for roadmap status synchronization.

## Files

- Create: `docs/ROADMAP.md`
- Create: `scripts/check_roadmap_status.py`
- Modify: `README.md`
- Modify: `docs/README_zh.md`
- Modify: `.github/workflows/workflow-lint.yml`
- Modify: `docs/release-checklist.md`
- Modify: `CONTRIBUTING.md`

## Validation Commands

```bash
python3 scripts/check_workflow_yaml.py
python3 scripts/check_version_consistency.py
python3 scripts/check_third_party_patch.py
python3 scripts/check_roadmap_status.py
DYLD_LIBRARY_PATH=/Users/skhe/projects/dmPython/dpi_bridge python3 -m pytest -q tests
```

## Completion Notes Template

When a roadmap phase is done, update `docs/ROADMAP.md` with:

1. `Status: DONE`
2. `Last Updated: YYYY-MM-DD`
3. `Evidence Links`: PR/commit/workflow run links
4. `Completion Notes`: 3-5 line summary
