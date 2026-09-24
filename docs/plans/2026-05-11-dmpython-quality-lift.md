# dmPython Quality Lift Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reduce macOS ARM64 bridge risk by tightening DPI compatibility, broadening real-DM integration coverage, and making releases and third-party patches repeatable and traceable.

**Architecture:** Keep the existing C extension API stable and improve the Go bridge behind the current `dpi_*` ABI. Add focused pytest integration coverage around observable DB-API behavior, then add small governance scripts that fail fast in CI before release.

**Tech Stack:** Python C extension, Go 1.21 c-shared library, `gitee.com/chunanyong/dm`, pytest, GitHub Actions, setuptools/delocate.

---

## File Structure

- Modify `dpi_bridge/dpi_bind.go`: implement real `dpi_exec_add_batch`, `dpi_exec_batch`, and explicit data-at-exec signaling instead of silent placeholder success.
- Modify `dpi_bridge/dpi_stmt.go`: expose a small internal helper for executing one bound parameter set so batch execution can reuse the same conversion path as normal execute.
- Modify `dpi_bridge/dpi_fetch.go`: preserve fetch/LOB boundary behavior while adding regression coverage only if a bridge change needs it.
- Create `tests/integration/test_p1_batch_contract.py`: real-DM tests for `executemany`, partial failure, rowcount, rollback, and generator/list parity.
- Create `tests/integration/test_p1_transaction_exception_contract.py`: commit/rollback/autocommit/error-state tests.
- Create `tests/integration/test_p1_encoding_contract.py`: Unicode, emoji, boundary, mixed `VARCHAR`/`CLOB`, and binary round-trip tests.
- Modify `tests/integration/test_p0_data_at_exec_matrix.py`: assert unsupported streaming/data-at-exec paths fail explicitly when direct API behavior is observable.
- Create `scripts/check_dpi_bridge_exports.py`: compare exported Go `//export dpi_*` symbols against the project DPI header declarations and an allowlist.
- Create `scripts/check_release_assets.py`: reusable release asset validation for local preflight and GitHub Actions.
- Modify `scripts/release_preflight.sh`: call release asset and bridge export checks.
- Modify `.github/workflows/workflow-lint.yml`: run the new bridge export check.
- Modify `.github/workflows/build-wheels.yml`: call the shared release asset checker instead of inline Python.
- Create `docs/dpi-bridge-compatibility.md`: document supported, intentionally ignored, and unsupported DPI functions/attributes.
- Modify `docs/ROADMAP.md`: update Phase 2-4 evidence fields when tasks complete.

---

### Task 1: Add DPI Bridge Export Drift Check

**Files:**
- Create: `scripts/check_dpi_bridge_exports.py`
- Modify: `.github/workflows/workflow-lint.yml`
- Test: run script locally

- [ ] **Step 1: Write the export checker**

Create `scripts/check_dpi_bridge_exports.py`:

```python
#!/usr/bin/env python3
"""Validate that Go bridge exports stay aligned with DPI header usage."""
from __future__ import annotations

import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
BRIDGE = ROOT / "dpi_bridge"
NATIVE = ROOT / "src/native"

ALLOWED_EXTRA_EXPORTS = {
    "dpi_module_init",
    "dpi_module_deinit",
}

ALLOWED_MISSING_EXPORTS = {
    # Add only when the C extension never calls the function on macOS.
}


def _symbols_from_go_exports() -> set[str]:
    symbols: set[str] = set()
    for path in BRIDGE.glob("*.go"):
        text = path.read_text(encoding="utf-8")
        symbols.update(re.findall(r"^//export\s+(dpi_[A-Za-z0-9_]+)\s*$", text, re.MULTILINE))
    return symbols


def _symbols_used_by_c_extension() -> set[str]:
    symbols: set[str] = set()
    for path in NATIVE.glob("*.c"):
        text = path.read_text(encoding="utf-8", errors="ignore")
        symbols.update(re.findall(r"\b(dpi_[A-Za-z0-9_]+)\s*\(", text))
    return symbols


def main() -> int:
    exported = _symbols_from_go_exports()
    used = _symbols_used_by_c_extension()

    missing = sorted((used - exported) - ALLOWED_MISSING_EXPORTS)
    extra = sorted((exported - used) - ALLOWED_EXTRA_EXPORTS)

    if missing:
        print("[FAIL] C extension calls DPI symbols not exported by Go bridge:")
        for name in missing:
            print(f"  - {name}")
        return 2

    if extra:
        print("[WARN] Go bridge exports symbols not currently called by C extension:")
        for name in extra:
            print(f"  - {name}")

    print(f"[OK] DPI bridge exports cover {len(used)} C-used symbols")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
```

- [ ] **Step 2: Run it and capture the first signal**

Run:

```bash
python scripts/check_dpi_bridge_exports.py
```

Expected: either `[OK] DPI bridge exports cover ...` or a concrete missing-symbol list. If missing symbols appear, add implementations or document them in `ALLOWED_MISSING_EXPORTS` only when the symbol is unreachable on macOS.

- [ ] **Step 3: Wire it into workflow lint**

Add this step after `Check third-party patch consistency` in `.github/workflows/workflow-lint.yml`:

```yaml
      - name: Check DPI bridge exports
        run: |
          python scripts/check_dpi_bridge_exports.py
```

- [ ] **Step 4: Verify workflow lint scripts locally**

Run:

```bash
python scripts/check_workflow_yaml.py
python scripts/check_dpi_bridge_exports.py
```

Expected: both commands exit `0`.

- [ ] **Step 5: Commit**

```bash
git add scripts/check_dpi_bridge_exports.py .github/workflows/workflow-lint.yml
git commit -m "ci: check dpi bridge export coverage"
```

---

### Task 2: Document DPI Compatibility Boundaries

**Files:**
- Create: `docs/dpi-bridge-compatibility.md`
- Modify: `README.md`
- Test: documentation grep checks

- [ ] **Step 1: Create compatibility document**

Create `docs/dpi-bridge-compatibility.md`:

```markdown
# DPI Bridge Compatibility

This project exposes a Go-built `libdmdpi.dylib` that implements the subset of the Dameng DPI ABI needed by the bundled Python C extension on macOS ARM64.

## Supported Paths

- Environment, connection, statement, and descriptor handle allocation/free.
- Login/logout using host, port, user, password, and optional catalog.
- Basic transaction operations: commit, rollback, and end-transaction.
- Prepared and direct execution for normal SQL statements.
- Positional parameter binding used by DB-API `execute` and `executemany`.
- Result metadata, row fetching, scalar values, CLOB, and BLOB round trips covered by integration tests.

## Accepted But Ignored Attributes

The bridge accepts selected DPI attributes for compatibility with the C extension and official driver call shape. Ignored attributes must not change observable Python behavior. Each ignored attribute must be listed in the corresponding switch statement in `dpi_bridge/dpi_conn.go` or `dpi_bridge/dpi_stmt.go`.

## Explicitly Unsupported Paths

- Streaming data-at-exec callbacks through `dpi_put_data` / `dpi_param_data`.
- Object, collection, BFILE, and cursor positioning behavior beyond the paths covered by integration tests.
- Vendor-certified production compatibility with the official proprietary `libdmdpi`.

Unsupported paths must return an error or `DSQL_NO_DATA`; they must not silently report success for behavior that did not happen.

## Regression Rule

Any new DPI function or attribute behavior must include one of:

- A real-DM integration test under `tests/integration`.
- An explicit entry in this document explaining why it is accepted, ignored, or unsupported.
```

- [ ] **Step 2: Link it from README**

Add under `Support Policy` in `README.md`:

```markdown
See [DPI Bridge Compatibility](docs/dpi-bridge-compatibility.md) for the supported Go bridge surface and known unsupported DPI paths.
```

- [ ] **Step 3: Verify docs references**

Run:

```bash
python - <<'PY'
from pathlib import Path
assert Path("docs/dpi-bridge-compatibility.md").exists()
assert "docs/dpi-bridge-compatibility.md" in Path("README.md").read_text(encoding="utf-8")
print("OK")
PY
```

Expected: `OK`.

- [ ] **Step 4: Commit**

```bash
git add docs/dpi-bridge-compatibility.md README.md
git commit -m "docs: define dpi bridge compatibility surface"
```

---

### Task 3: Make Unsupported Streaming/Data-at-Exec Explicit

**Files:**
- Modify: `dpi_bridge/dpi_bind.go`
- Modify: `docs/dpi-bridge-compatibility.md`
- Test: `python -m pytest -q tests/integration/test_p0_data_at_exec_matrix.py -m requires_dm`

- [ ] **Step 1: Change placeholder success into explicit unsupported behavior**

In `dpi_bridge/dpi_bind.go`, replace the placeholder functions with:

```go
//export dpi_put_data
func dpi_put_data(hstmt C.dhstmt, val C.dpointer, valLen C.slength) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	stmt.lastErr = &diagInfo{
		errorCode: -1,
		message:   "streaming data-at-exec is not supported by dmPython macOS bridge",
	}
	return DSQL_ERROR
}

//export dpi_param_data
func dpi_param_data(hstmt C.dhstmt, valPtr *C.dpointer) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	stmt.lastErr = &diagInfo{
		errorCode: -1,
		message:   "streaming data-at-exec is not supported by dmPython macOS bridge",
	}
	return DSQL_NO_DATA
}
```

- [ ] **Step 2: Confirm normal large LOB behavior still passes**

Run:

```bash
DYLD_LIBRARY_PATH=dpi_bridge python setup.py build_ext --inplace
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q tests/integration/test_p0_data_at_exec_matrix.py -m requires_dm
```

Expected: existing large CLOB/BLOB round-trip tests pass. These tests use normal parameter binding, not streaming callbacks.

- [ ] **Step 3: Commit**

```bash
git add dpi_bridge/dpi_bind.go docs/dpi-bridge-compatibility.md
git commit -m "fix: make unsupported streaming data-at-exec explicit"
```

---

### Task 4: Add Batch Execution Contract Tests

**Files:**
- Create: `tests/integration/test_p1_batch_contract.py`
- Test: `tests/integration/test_p1_batch_contract.py`

- [ ] **Step 1: Write failing tests for batch behavior**

Create `tests/integration/test_p1_batch_contract.py`:

```python
from __future__ import annotations

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


def test_executemany_list_rows_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_BATCH")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(32))")
        cur.executemany(
            f"INSERT INTO {table_name} (id, v) VALUES (?, ?)",
            [(1, "one"), (2, "two"), (3, "three")],
        )
        conn.commit()

        cur.execute(f"SELECT id, v FROM {table_name} ORDER BY id")
        assert cur.fetchall() == [(1, "one"), (2, "two"), (3, "three")]
        assert cur.rowcount in (-1, 3)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_executemany_generator_rows_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_BATCH_GEN")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(32))")

        def rows():
            for idx in range(1, 6):
                yield (idx, f"value_{idx}")

        cur.executemany(f"INSERT INTO {table_name} (id, v) VALUES (?, ?)", rows())
        conn.commit()

        cur.execute(f"SELECT COUNT(*), MIN(v), MAX(v) FROM {table_name}")
        assert cur.fetchone() == (5, "value_1", "value_5")
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_executemany_duplicate_key_failure_rolls_back(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_BATCH_FAIL")
    cur = conn.cursor()
    original_autocommit = int(conn.autocommit)
    try:
        conn.autocommit = 0
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(32))")
        conn.commit()

        with pytest.raises(dmPython.DatabaseError) as excinfo:
            cur.executemany(
                f"INSERT INTO {table_name} (id, v) VALUES (?, ?)",
                [(1, "one"), (2, "two"), (1, "duplicate")],
            )
        assert "[CODE:" in str(excinfo.value)

        conn.rollback()
        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert cur.fetchone() == (0,)
    finally:
        try:
            drop_table(cur, table_name)
            conn.commit()
        except Exception:
            pass
        conn.autocommit = original_autocommit
        cur.close()
```

- [ ] **Step 2: Run tests to establish current behavior**

Run:

```bash
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q tests/integration/test_p1_batch_contract.py
```

Expected: failures identify exact current gaps, usually rowcount or partial failure handling.

- [ ] **Step 3: Commit tests before implementation**

```bash
git add tests/integration/test_p1_batch_contract.py
git commit -m "test: cover executemany batch contracts"
```

---

### Task 5: Implement Real Batch Execution in Go Bridge

**Files:**
- Modify: `dpi_bridge/dpi_stmt.go`
- Modify: `dpi_bridge/dpi_bind.go`
- Test: `tests/integration/test_p1_batch_contract.py`

- [ ] **Step 1: Add batch storage to `stmtHandle`**

In `dpi_bridge/dpi_stmt.go`, add this field to `stmtHandle` near `paramBindings`:

```go
	batches []map[int]bindParamInfo
```

Initialize it in `newStmtHandle`:

```go
		batches:       make([]map[int]bindParamInfo, 0),
```

- [ ] **Step 2: Add a copy helper in `dpi_bridge/dpi_bind.go`**

Add:

```go
func cloneParamBindings(src map[int]bindParamInfo) map[int]bindParamInfo {
	dst := make(map[int]bindParamInfo, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
```

- [ ] **Step 3: Replace batch placeholders**

Replace `dpi_exec_add_batch` and `dpi_exec_batch` in `dpi_bridge/dpi_bind.go`:

```go
//export dpi_exec_add_batch
func dpi_exec_add_batch(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	if len(stmt.paramBindings) == 0 {
		stmt.lastErr = &diagInfo{errorCode: -1, message: "cannot add empty parameter batch"}
		return DSQL_ERROR
	}
	stmt.batches = append(stmt.batches, cloneParamBindings(stmt.paramBindings))
	return DSQL_SUCCESS
}

//export dpi_exec_batch
func dpi_exec_batch(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	if len(stmt.batches) == 0 {
		return DSQL_SUCCESS
	}

	var total int64
	original := stmt.paramBindings
	for _, batch := range stmt.batches {
		stmt.paramBindings = batch
		if rt := stmt.execLocked(); rt != DSQL_SUCCESS {
			stmt.paramBindings = original
			stmt.batches = nil
			return rt
		}
		total += stmt.rowsAffected
	}
	stmt.paramBindings = original
	stmt.rowsAffected = total
	stmt.batches = nil
	return DSQL_SUCCESS
}
```

- [ ] **Step 4: Extract `execLocked` from existing execute path**

In `dpi_bridge/dpi_stmt.go`, move the body of `dpi_exec` after handle lookup into:

```go
func (stmt *stmtHandle) execLocked() C.DPIRETURN {
	// Move the existing implementation here without acquiring stmt.mu.
	// Keep all existing conversion, rowsAffected, rows, columns, and diagnostic behavior unchanged.
}
```

Then make `dpi_exec` call it:

```go
//export dpi_exec
func dpi_exec(hstmt C.dhstmt) C.DPIRETURN {
	stmt, err := getStmtHandle(hstmt)
	if err != nil {
		return DSQL_INVALID_HANDLE
	}
	stmt.mu.Lock()
	defer stmt.mu.Unlock()

	return stmt.execLocked()
}
```

- [ ] **Step 5: Run batch tests**

Run:

```bash
DYLD_LIBRARY_PATH=dpi_bridge go test ./dpi_bridge
DYLD_LIBRARY_PATH=dpi_bridge python setup.py build_ext --inplace
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q tests/integration/test_p1_batch_contract.py
```

Expected: Go build/test succeeds and all batch contract tests pass.

- [ ] **Step 6: Commit**

```bash
git add dpi_bridge/dpi_stmt.go dpi_bridge/dpi_bind.go
git commit -m "fix: execute dpi parameter batches"
```

---

### Task 6: Expand Transaction and Exception Contracts

**Files:**
- Create: `tests/integration/test_p1_transaction_exception_contract.py`
- Test: that file

- [ ] **Step 1: Add transaction/error-state tests**

Create `tests/integration/test_p1_transaction_exception_contract.py`:

```python
from __future__ import annotations

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


def test_rollback_removes_uncommitted_insert(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_TX")
    cur = conn.cursor()
    original_autocommit = int(conn.autocommit)
    try:
        conn.autocommit = 0
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(16))")
        conn.commit()
        cur.execute(f"INSERT INTO {table_name} VALUES (?, ?)", (1, "rollback"))
        conn.rollback()

        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert cur.fetchone() == (0,)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        conn.autocommit = original_autocommit
        cur.close()


def test_commit_persists_insert(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_COMMIT")
    cur = conn.cursor()
    original_autocommit = int(conn.autocommit)
    try:
        conn.autocommit = 0
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(16))")
        conn.commit()
        cur.execute(f"INSERT INTO {table_name} VALUES (?, ?)", (1, "commit"))
        conn.commit()

        cur.execute(f"SELECT v FROM {table_name} WHERE id = 1")
        assert cur.fetchone() == ("commit",)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        conn.autocommit = original_autocommit
        cur.close()


def test_cursor_usable_after_database_error(conn):
    cur = conn.cursor()
    try:
        with pytest.raises(dmPython.DatabaseError):
            cur.execute("SELECT * FROM DMPY_TABLE_DOES_NOT_EXIST")
        cur.execute("SELECT 1")
        assert cur.fetchone() == (1,)
    finally:
        cur.close()


def test_connection_close_errors_are_python_exceptions(conn_params):
    conn = dmPython.connect(**conn_params)
    cur = conn.cursor()
    cur.close()
    conn.close()

    with pytest.raises(Exception) as excinfo:
        conn.cursor()
    assert "not connected" in str(excinfo.value).lower() or "closed" in str(excinfo.value).lower()
```

- [ ] **Step 2: Run the tests**

Run:

```bash
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q tests/integration/test_p1_transaction_exception_contract.py
```

Expected: pass, or fail with a precise bridge transaction/error-state regression.

- [ ] **Step 3: Fix only observed failures**

If rollback/commit fails, inspect `dpi_bridge/dpi_conn.go` functions `dpi_commit`, `dpi_rollback`, and autocommit handling. Keep the fix within `dpi_bridge/dpi_conn.go`; do not change Python API shape.

- [ ] **Step 4: Re-run combined P1**

Run:

```bash
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q tests/integration -m p1_contract
```

Expected: all P1 contract tests pass.

- [ ] **Step 5: Commit**

```bash
git add tests/integration/test_p1_transaction_exception_contract.py dpi_bridge/dpi_conn.go
git commit -m "test: expand transaction and exception contracts"
```

---

### Task 7: Expand Encoding and LOB Regression Coverage

**Files:**
- Create: `tests/integration/test_p1_encoding_contract.py`
- Test: that file plus existing LOB tests

- [ ] **Step 1: Add encoding tests**

Create `tests/integration/test_p1_encoding_contract.py`:

```python
from __future__ import annotations

import pytest


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


UNICODE_CASES = [
    "ascii",
    "中文",
    "emoji_🚀_边界",
    "combining_e\u0301",
    "rare_𠮷字",
    "mixed_中文_emoji_🚀_ascii_123",
]


@pytest.mark.parametrize("payload", UNICODE_CASES)
def test_varchar_unicode_roundtrip(conn, table_name_factory, drop_table, payload):
    table_name = table_name_factory("DMPY_P1_ENC_V")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(512))")
        cur.execute(f"INSERT INTO {table_name} VALUES (?, ?)", (1, payload))
        conn.commit()

        cur.execute(f"SELECT v FROM {table_name} WHERE id = 1")
        assert cur.fetchone() == (payload,)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_clob_unicode_multiboundary_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_ENC_C")
    payload = ("中文🚀𠮷e\u0301" * 1800)[:12000]
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB)")
        cur.execute(f"INSERT INTO {table_name} VALUES (?, ?)", (1, payload))
        conn.commit()

        cur.execute(f"SELECT c FROM {table_name} WHERE id = 1")
        value = cur.fetchone()[0]
        assert isinstance(value, str)
        assert value == payload
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_varbinary_all_byte_values_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_ENC_B")
    payload = bytes(range(256)) * 8
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, b BLOB)")
        cur.execute(f"INSERT INTO {table_name} VALUES (?, ?)", (1, payload))
        conn.commit()

        cur.execute(f"SELECT b FROM {table_name} WHERE id = 1")
        value = cur.fetchone()[0]
        assert bytes(value) == payload
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()
```

- [ ] **Step 2: Run encoding and existing LOB tests**

Run:

```bash
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q \
  tests/integration/test_p1_encoding_contract.py \
  tests/integration/test_p0_lob_boundary.py \
  tests/integration/test_p0_clob_unicode_regression.py
```

Expected: all pass.

- [ ] **Step 3: Fix only encoding-specific failures**

If failures are CLOB chunk-boundary related, inspect vendored patch in `dpi_bridge/third_party/chunanyong_dm/a.go` and update `dpi_bridge/third_party/chunanyong_dm/PATCHES.md` in the same commit.

- [ ] **Step 4: Commit**

```bash
git add tests/integration/test_p1_encoding_contract.py dpi_bridge/third_party/chunanyong_dm/a.go dpi_bridge/third_party/chunanyong_dm/PATCHES.md
git commit -m "test: expand unicode and binary roundtrip coverage"
```

---

### Task 8: Share Release Asset Validation

**Files:**
- Create: `scripts/check_release_assets.py`
- Modify: `.github/workflows/build-wheels.yml`
- Modify: `scripts/release_preflight.sh`
- Test: script against `dist_fixed`

- [ ] **Step 1: Create reusable release checker**

Create `scripts/check_release_assets.py`:

```python
#!/usr/bin/env python3
"""Validate dmPython macOS release wheel assets."""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path


EXPECTED_TAGS = {"cp39-cp39", "cp310-cp310", "cp311-cp311", "cp312-cp312", "cp313-cp313"}


def _wheel_tags(paths: list[Path]) -> set[str]:
    tags: set[str] = set()
    for path in paths:
        name = path.name
        if "arm64.whl" not in name:
            raise RuntimeError(f"non-arm64 wheel found: {name}")
        match = re.search(r"-(cp\d{2,3}-cp\d{2,3})-macosx_.*_arm64\.whl$", name)
        if not match:
            raise RuntimeError(f"unrecognized wheel name: {name}")
        tags.add(match.group(1))
    return tags


def check_directory(path: Path) -> None:
    wheels = sorted(path.glob("*.whl"))
    if len(wheels) != len(EXPECTED_TAGS):
        raise RuntimeError(f"expected {len(EXPECTED_TAGS)} wheels, got {len(wheels)}")
    found = _wheel_tags(wheels)
    if found != EXPECTED_TAGS:
        raise RuntimeError(f"wheel tag mismatch: expected={sorted(EXPECTED_TAGS)} found={sorted(found)}")


def check_github_release(tag: str) -> None:
    raw = subprocess.check_output(["gh", "release", "view", tag, "--json", "assets"], text=True)
    assets = [item["name"] for item in json.loads(raw)["assets"]]
    wheels = [Path(name) for name in assets if name.endswith(".whl")]
    found = _wheel_tags(wheels)
    missing_tags = sorted(EXPECTED_TAGS - found)
    missing_files = sorted({"checksums.txt", "build-metadata.json"} - set(assets))
    if missing_tags or missing_files:
        raise RuntimeError(
            f"release assets incomplete: missing_tags={missing_tags} missing_files={missing_files} assets={assets}"
        )


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dir", type=Path, help="Directory containing local wheels")
    parser.add_argument("--github-release", help="GitHub release tag to validate with gh")
    args = parser.parse_args()

    if not args.dir and not args.github_release:
        parser.error("provide --dir or --github-release")

    if args.dir:
        check_directory(args.dir)
    if args.github_release:
        check_github_release(args.github_release)

    print("[OK] release assets validated")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except RuntimeError as exc:
        print(f"[FAIL] {exc}")
        raise SystemExit(2)
```

- [ ] **Step 2: Replace inline release asset validation in workflow**

In `.github/workflows/build-wheels.yml`, replace the duplicated wheel tag validation block with:

```yaml
          python scripts/check_release_assets.py --dir dist_fixed
```

Keep the existing metadata JSON and checksum generation in the workflow.

- [ ] **Step 3: Replace release verification block**

In the `Verify release assets completeness` step, replace inline Python with:

```yaml
          python scripts/check_release_assets.py --github-release "$TAG_NAME"
```

- [ ] **Step 4: Add local preflight call**

In `scripts/release_preflight.sh`, after version consistency checks, add:

```bash
echo "[INFO] dpi bridge export consistency"
python3 scripts/check_dpi_bridge_exports.py
```

If `dist_fixed` exists, validate it:

```bash
if [ -d dist_fixed ]; then
  echo "[INFO] local release asset consistency"
  python3 scripts/check_release_assets.py --dir dist_fixed
fi
```

- [ ] **Step 5: Verify scripts**

Run:

```bash
python scripts/check_release_assets.py --help
python scripts/check_dpi_bridge_exports.py
python scripts/check_workflow_yaml.py
```

Expected: commands exit `0`.

- [ ] **Step 6: Commit**

```bash
git add scripts/check_release_assets.py scripts/release_preflight.sh .github/workflows/build-wheels.yml
git commit -m "ci: share release asset validation"
```

---

### Task 9: Strengthen Third-Party Patch Traceability

**Files:**
- Modify: `scripts/check_third_party_patch.py`
- Modify: `dpi_bridge/third_party/chunanyong_dm/PATCHES.md`
- Test: patch script

- [ ] **Step 1: Add source version and checksum checks**

Modify `scripts/check_third_party_patch.py` to include:

```python
import hashlib
import re
```

Add paths:

```python
VERSION_FILE = ROOT / "dpi_bridge/third_party/chunanyong_dm/VERSION"
```

Add helper:

```python
def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()
```

Add checks inside `main()`:

```python
    if not VERSION_FILE.exists():
        fail(f"missing {VERSION_FILE}")
    version = VERSION_FILE.read_text(encoding="utf-8").strip()
    require_contains(PATCH_DOC, f"Vendored version: `{version}`")

    patch_text = PATCH_DOC.read_text(encoding="utf-8")
    expected_hash = sha256(PATCH_FILE)
    if expected_hash not in patch_text:
        fail(f"{PATCH_DOC} missing current sha256 for {PATCH_FILE.name}: {expected_hash}")
```

- [ ] **Step 2: Update patch documentation**

Add to `dpi_bridge/third_party/chunanyong_dm/PATCHES.md`:

```markdown
## Traceability

Vendored version: `1.8.22`

Patched file checksums:

- `a.go`: `<actual sha256 from scripts/check_third_party_patch.py failure output>`
```

Run the script once, copy the exact reported SHA-256 into the document, then rerun.

- [ ] **Step 3: Verify**

Run:

```bash
python scripts/check_third_party_patch.py
```

Expected: `[OK] third-party patch consistency checks passed`.

- [ ] **Step 4: Commit**

```bash
git add scripts/check_third_party_patch.py dpi_bridge/third_party/chunanyong_dm/PATCHES.md
git commit -m "ci: trace vendored dm driver patch"
```

---

### Task 10: Run Final Regression Matrix and Update Roadmap

**Files:**
- Modify: `docs/ROADMAP.md`
- Test: full local feasible matrix

- [ ] **Step 1: Run non-DM checks**

Run:

```bash
python scripts/check_version_consistency.py
python scripts/check_workflow_yaml.py
python scripts/check_third_party_patch.py
python scripts/check_dpi_bridge_exports.py
python -m pytest -q tests/test_version_metadata.py
```

Expected: all pass.

- [ ] **Step 2: Run DM integration checks**

Run with a reachable DM instance:

```bash
DYLD_LIBRARY_PATH=dpi_bridge python setup.py build_ext --inplace
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q tests/integration -m "p0_stability or p1_contract"
```

Expected: all P0/P1 tests pass, no exit `139` or `-11`.

- [ ] **Step 3: Run optional full regression**

Run:

```bash
DYLD_LIBRARY_PATH=dpi_bridge python -m pytest -q tests/integration -m "p0_stability or p1_contract or p2_scale"
```

Expected: pass in a stable DM environment. If environment capacity is insufficient, record the exact skipped command and reason in the final implementation notes.

- [ ] **Step 4: Update roadmap evidence**

In `docs/ROADMAP.md`, update:

```markdown
| Phase 2 | Week 3-4 | DONE | Maintainers | `requires_dm` regression green, no crash/139, P0/P1 contract coverage strengthened | 2026-05-11 | P0/P1 local regression, CI workflow lint |
| Phase 3 | Week 5-6 | DONE | Maintainers | Release preflight and asset verification stable, tag release idempotency remains green | 2026-05-11 | `scripts/check_release_assets.py`, release workflow validation |
| Phase 4 | Week 7-8 | DONE | Maintainers | Third-party patch drift guard and upstream sync governance are documented and enforced | 2026-05-11 | `scripts/check_third_party_patch.py`, `PATCHES.md` traceability |
```

Add a short completion note under each phase:

```markdown
Completed on 2026-05-11.
- Added integration coverage for batch, transaction, exception, LOB, and encoding contracts.
- Added bridge export drift checks and explicit unsupported-path documentation.
- Added reusable release asset and third-party patch traceability checks.
```

- [ ] **Step 5: Commit**

```bash
git add docs/ROADMAP.md
git commit -m "docs: record quality lift roadmap evidence"
```

---

## Execution Notes

- Keep commits small and ordered by task.
- Do not change public Python API names unless a failing compatibility test proves the current API is wrong.
- Do not add new dependencies.
- Do not broaden CI secrets requirements; integration tests must continue to skip clearly when DM secrets are missing.
- If a test exposes a C extension crash, preserve the failing test and fix the smallest C/Go boundary that removes the crash.

## Self-Review

- Spec coverage:
  - Go bridge vs official DPI differences: Tasks 1-3 and 5.
  - Real DM integration coverage for LOB, batch, transaction, exception, encoding: Tasks 4, 6, 7, and final regression in Task 10.
  - Repeatable release/version/third-party traceability: Tasks 8-10.
- Placeholder scan:
  - No open-ended implementation placeholders are intended for executable steps. The only implementation extraction in Task 5 points to the existing `dpi_exec` body and gives the exact target shape.
- Type consistency:
  - New Go field `batches []map[int]bindParamInfo` matches helper `cloneParamBindings`.
  - New scripts use only standard library modules.
  - New pytest files use existing fixtures from `tests/conftest.py`.
