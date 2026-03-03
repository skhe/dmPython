# Local Patch Notes

## Base

- Upstream module: `gitee.com/chunanyong/dm`
- Base version: `v1.8.22`

## Patch: CLOB Unicode read corruption on large payloads

- File: `a.go`
- Function: `dm_build_610`
- Problem:
  - Large CLOB payloads (for example `>=16000` chars) could return corrupted Unicode
    when fetched, with replacement characters (`U+FFFD`) inserted at chunk boundaries.
  - Database-side `length(c)` remained correct, while client-side decoded string was incorrect.
- Root cause:
  - The original implementation decoded each chunk independently and concatenated strings.
  - If a chunk boundary split a multi-byte sequence, per-chunk decode could emit replacement
    characters.
- Fix:
  - Accumulate raw bytes for all chunks first.
  - Decode once at the end using server encoding.
  - Keep original offset progression and `readOver` behavior.
  - Preserve fallback `charLen == -1` handling for compatibility.

## Regression coverage

- `tests/integration/test_p0_clob_unicode_regression.py`
  - `test_clob_unicode_problem_patterns_roundtrip`
  - `test_clob_unicode_problem_patterns_executemany_roundtrip`
  - `test_clob_unicode_problem_patterns_length_contract`
  - `test_clob_unicode_problem_patterns_subprocess_no_crash`

## Rollback

- Remove `replace gitee.com/chunanyong/dm => ./third_party/chunanyong_dm` in `dpi_bridge/go.mod`.
- Revert `third_party/chunanyong_dm` changes.
