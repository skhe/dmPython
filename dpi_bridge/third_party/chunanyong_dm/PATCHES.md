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

## Patch: initial connection timeout through endpoint groups

- Files: `a.go`, `n.go`, `x.go`, `y.go`, `m.go`
- Problem: ordinary host connections are wrapped in an endpoint group. That path
  replaced the caller's context with `context.Background()`, so `login_timeout`
  could not stop an unresponsive handshake.
- Fix: pass the original context through endpoint selection and dialing, apply
  its deadline to the socket during handshake, then clear the deadline once the
  connection is established. Endpoint retries and wait intervals also respect
  cancellation. Reconnects retain a fresh background context.
- Regression: `test_login_timeout_interrupts_unresponsive_handshake` uses a
  local TCP listener that accepts a connection but never replies.

## Patch: interrupt blocked statements on context timeout

- File: `m.go`
- Problem: context cancellation called cleanup, which tried to roll back on a
  socket still blocked in a statement. The caller waited indefinitely.
- Fix: close the socket before cleanup so the blocked read and rollback return.
- Regression: `test_connection_timeout_limits_sql_execution` holds a row lock
  while another session executes an update with a one-second timeout.

## Patch: application name query value

- File: `n.go`
- Decode the escaped `appName` query value so spaces, `&`, and `+` reach the
  driver as one application name. Other existing DSN properties retain their
  previous parsing behavior.
- Regression: `test_connection_timeout_options_are_reported` connects with an
  application name containing `&` and `+`.

## Patch: validate user-defined object members before encoding

- File: `zzo.go`
- Problem: a nested object with the wrong member representation or count could
  trigger a Go type assertion panic or an index-out-of-range panic.
- Fix: check the member count and type before encoding, and return a driver
  error for invalid values. Accept pointer forms of nested objects as well.
- Regression: `tests/integration/test_p1_object_types.py` covers nested object
  writes and reads against a real database.

## Patch: expose array type metadata and encode nested arrays

- Files: `type_info.go`, `zzo.go`
- The bridge reads element type and maximum length from the driver's server
  descriptor instead of parsing type DDL text.
- Initialize nested `DmArray` values with the field descriptor before encoding;
  otherwise an array inside an object is silently written as empty.
- Regression: `tests/integration/test_p1_array_types.py` covers high-precision
  decimals, Unicode, nulls, empty arrays, length limits, and arrays in objects.

## Rollback

- Remove `replace gitee.com/chunanyong/dm => ./third_party/chunanyong_dm` in `dpi_bridge/go.mod`.
- Revert `third_party/chunanyong_dm` changes.
