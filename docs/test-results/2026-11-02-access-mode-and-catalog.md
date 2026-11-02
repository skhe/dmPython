# Connection access mode and catalog regression

Validated on 2026-11-02 with a native macOS ARM64 build, CPython 3.12 and a real DM8 database. No database proxy or Linux container was used for the client.

## Reproduced behavior

The original bridge accepted `access_mode=dmPython.DSQL_MODE_READ_ONLY` but reported mode 0 and allowed a zero-row DELETE. It also returned an empty `current_catalog`, although `SELECT NAME FROM V$DATABASE` returned a database name. All seven new access-mode/catalog regression cases failed against the original bridge.

## Fixed behavior

- Apply read-only mode on the server before returning a successful login, and report the effective requested mode.
- Keep the read-only session pinned in autocommit mode. Reuse it when disabling autocommit, and reapply protection after enabling autocommit. Close the connection if protection cannot be restored.
- Apply runtime mode changes and invalidate prepared statements when their connection binding changes. Reject invalid modes without relaxing the current mode.
- Read the current catalog from the connected database, propagating metadata query errors.

The following command passed **20 tests with no skips**, including all seven new regressions:

```bash
python -m pytest \
  tests/integration/test_p1_connection_access_mode.py \
  tests/integration/test_p1_contract.py \
  tests/integration/test_p1_cursor_state_contract.py \
  --require-dm -p no:randomly -q
```

`DM_TEST_HOST`, `DM_TEST_PORT`, `DM_TEST_USER` and `DM_TEST_PASSWORD` must point to a dedicated test schema. The new tests create and remove their own randomly named table. Their write-rejection checks use zero-row DELETE statements.

The bridge was also exercised through the unmodified official 2.5.38 C extension in a downstream application: connection testing, catalog/schema discovery, parameterized read-only queries, eight concurrent queries, write rejection and recovery after invalid credentials passed. Source-table snapshots before and after that application check were identical.

This does not establish complete interchangeability with the vendor DPI SDK. An additional `executemany` error-contract check failed with the official 2.5.38 C extension against both the original and patched bridge; the same check passed with this fork's C extension and is included in the 20-test result above. No vendor SDK or compiled driver artifact is distributed by this change.
