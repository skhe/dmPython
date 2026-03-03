from __future__ import annotations

import textwrap

import pytest


pytestmark = [pytest.mark.requires_dm, pytest.mark.p0_stability]


def _assert_subprocess_ok(result, label: str) -> None:
    assert result.returncode == 0, (
        f"{label} failed\n"
        f"return code: {result.returncode}\n"
        f"stdout:\n{result.stdout}\n"
        f"stderr:\n{result.stderr}"
    )
    assert result.returncode not in (139, -11), (
        f"{label} hit segfault-like exit code: {result.returncode}\n"
        f"stdout:\n{result.stdout}\n"
        f"stderr:\n{result.stderr}"
    )


def test_clob_exact_4k_boundary_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_C4K")
    cur = conn.cursor()
    try:
        payload = ("中文🚀边界" * 1000)[:4000]
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB)")
        cur.execute(f"INSERT INTO {table_name} (id, c) VALUES (?, ?)", (1, payload))
        conn.commit()
        cur.execute(f"SELECT c FROM {table_name} WHERE id = 1")
        value = cur.fetchone()[0]
        assert isinstance(value, str)
        assert value == payload
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


@pytest.mark.parametrize(
    ("row_id", "payload"),
    [
        (1, bytes(range(256)) * 16),
        (2, bytes(range(256)) * 32),
    ],
    ids=["blob_4k", "blob_8k"],
)
def test_blob_exact_4k_and_8k_roundtrip(conn, table_name_factory, drop_table, row_id, payload):
    table_name = table_name_factory("DMPY_P0_B4K")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, b BLOB)")
        cur.execute(f"INSERT INTO {table_name} (id, b) VALUES (?, ?)", (row_id, payload))
        conn.commit()
        cur.execute(f"SELECT b FROM {table_name} WHERE id = ?", (row_id,))
        value = cur.fetchone()[0]
        assert isinstance(value, (bytes, bytearray))
        assert bytes(value) == payload
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_clob_over_4k_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_C5K")
    cur = conn.cursor()
    try:
        payload = ("中文🚀边界" * 1200)[:5000]
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB)")
        cur.execute(f"INSERT INTO {table_name} (id, c) VALUES (?, ?)", (1, payload))
        conn.commit()
        cur.execute(f"SELECT c FROM {table_name} WHERE id = 1")
        value = cur.fetchone()[0]
        assert isinstance(value, str)
        assert value == payload
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_lob_null_vs_empty_contract(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_NULL")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB, b BLOB)")
        cur.execute(f"INSERT INTO {table_name} (id, c, b) VALUES (?, ?, ?)", (1, None, None))
        cur.execute(f"INSERT INTO {table_name} (id, c, b) VALUES (?, ?, ?)", (2, "", b""))
        conn.commit()

        cur.execute(f"SELECT c, b FROM {table_name} WHERE id = 1")
        null_row = cur.fetchone()
        assert null_row[0] is None
        assert null_row[1] is None

        cur.execute(f"SELECT c, b FROM {table_name} WHERE id = 2")
        empty_row = cur.fetchone()
        assert empty_row[0] == ""
        assert isinstance(empty_row[1], (bytes, bytearray))
        assert bytes(empty_row[1]) == b""
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


@pytest.mark.crash_guard
def test_lob_error_path_after_close_no_segfault_subprocess(run_in_subprocess):
    code = textwrap.dedent(
        """
        import uuid
        import dmPython

        table = "DMPY_P0_CLOSE_" + uuid.uuid4().hex[:8].upper()
        conn = dmPython.connect(
            user="SYSDBA",
            password="SYSDBA001",
            server="localhost",
            port=5237,
        )
        cur = conn.cursor()
        try:
            payload = ("x" * 2400)
            cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, c CLOB)")
            cur.execute(f"INSERT INTO {table} (id, c) VALUES (?, ?)", (1, payload))
            conn.commit()
            cur.execute(f"SELECT c FROM {table} WHERE id = 1")
            row = cur.fetchone()
            assert row and isinstance(row[0], str)
            cur.execute(f"DROP TABLE {table}")
            conn.commit()
        finally:
            cur.close()
            conn.close()

        try:
            cur.execute("SELECT 1")
            raise AssertionError("expected closed-handle error")
        except Exception as exc:
            print("CLOSED_ERROR", type(exc).__name__, str(exc))
        """
    )
    result = run_in_subprocess(code)
    _assert_subprocess_ok(result, "LOB close-path crash guard")
    stdout = result.stdout.lower()
    assert "closed_error" in stdout
    assert ("closed" in stdout) or ("invalid" in stdout) or ("not open" in stdout)
