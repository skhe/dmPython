from __future__ import annotations

import textwrap

import pytest


pytestmark = [pytest.mark.requires_dm, pytest.mark.p0_stability]


def _make_clob_payload(size: int) -> str:
    if size <= 0:
        return ""
    base = "中文🚀A"
    return (base * ((size // len(base)) + 2))[:size]


def _make_blob_payload(size: int) -> bytes:
    if size <= 0:
        return b""
    return bytes((i % 256 for i in range(size)))


def _assert_subprocess_ok(result, label: str) -> None:
    assert result.returncode == 0, (
        f"{label} failed\n"
        f"return code: {result.returncode}\n"
        f"stdout:\n{result.stdout}\n"
        f"stderr:\n{result.stderr}"
    )
    assert result.returncode not in (139, -11), (
        f"{label} segfault-like exit code: {result.returncode}\n"
        f"stdout:\n{result.stdout}\n"
        f"stderr:\n{result.stderr}"
    )


def test_data_at_exec_clob_boundary_matrix_roundtrip(
    conn, table_name_factory, drop_table, boundary_sizes
):
    table_name = table_name_factory("DMPY_P0_DE_C")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB)")
        for idx, size in enumerate(boundary_sizes, start=1):
            payload = _make_clob_payload(size)
            cur.execute(f"INSERT INTO {table_name} (id, c) VALUES (?, ?)", (idx, payload))
            conn.commit()

            cur.execute(f"SELECT c FROM {table_name} WHERE id = ?", (idx,))
            value = cur.fetchone()[0]
            assert isinstance(value, str), f"size={size}, type={type(value)}"
            assert value == payload, f"size={size}"
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_data_at_exec_blob_boundary_matrix_roundtrip(
    conn, table_name_factory, drop_table, boundary_sizes
):
    table_name = table_name_factory("DMPY_P0_DE_B")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, b BLOB)")
        for idx, size in enumerate(boundary_sizes, start=1):
            payload = _make_blob_payload(size)
            cur.execute(f"INSERT INTO {table_name} (id, b) VALUES (?, ?)", (idx, payload))
            conn.commit()

            cur.execute(f"SELECT b FROM {table_name} WHERE id = ?", (idx,))
            value = cur.fetchone()[0]
            assert isinstance(value, (bytes, bytearray)), f"size={size}, type={type(value)}"
            assert bytes(value) == payload, f"size={size}"
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_data_at_exec_alternating_small_large_same_statement(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_DE_ALT")
    cur = conn.cursor()
    sizes = [16, 16384, 64, 8192, 1, 4097, 1024, 2048]
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB, b BLOB)")
        sql = f"INSERT INTO {table_name} (id, c, b) VALUES (?, ?, ?)"
        for idx, size in enumerate(sizes, start=1):
            c_payload = _make_clob_payload(size)
            b_payload = _make_blob_payload(size)
            cur.execute(sql, (idx, c_payload, b_payload))
        conn.commit()

        cur.execute(f"SELECT id, c, b FROM {table_name} ORDER BY id")
        rows = cur.fetchall()
        assert len(rows) == len(sizes)
        for idx, row in enumerate(rows, start=1):
            size = sizes[idx - 1]
            assert row[0] == idx
            assert row[1] == _make_clob_payload(size)
            assert bytes(row[2]) == _make_blob_payload(size)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_data_at_exec_executemany_generator_large_payloads(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_DE_EXM")
    cur = conn.cursor()
    sizes = [2048, 12288, 3072, 16384, 4097, 8192, 1024, 10000]
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB, b BLOB)")

        def _rows():
            for idx, size in enumerate(sizes, start=1):
                yield (idx, _make_clob_payload(size), _make_blob_payload(size))

        cur.executemany(f"INSERT INTO {table_name} (id, c, b) VALUES (?, ?, ?)", _rows())
        conn.commit()

        cur.execute(f"SELECT id, c, b FROM {table_name} ORDER BY id")
        rows = cur.fetchall()
        assert len(rows) == len(sizes)
        for row in rows:
            idx = row[0]
            expected_size = sizes[idx - 1]
            assert row[1] == _make_clob_payload(expected_size)
            assert bytes(row[2]) == _make_blob_payload(expected_size)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


@pytest.mark.crash_guard
def test_data_at_exec_subprocess_no_segfault_matrix(run_in_subprocess, boundary_sizes):
    code = textwrap.dedent(
        f"""
        import dmPython
        import uuid

        sizes = {boundary_sizes!r}
        table = "DMPY_P0_DE_SP_" + uuid.uuid4().hex[:8].upper()
        conn = dmPython.connect(
            user="SYSDBA",
            password="SYSDBA001",
            server="localhost",
            port=5237,
        )
        cur = conn.cursor()
        try:
            cur.execute(f"CREATE TABLE {{table}} (id INT PRIMARY KEY, c CLOB, b BLOB)")
            for idx, size in enumerate(sizes, start=1):
                text = ("中文🚀A" * ((size // 4) + 2))[:size] if size > 0 else ""
                blob = bytes((i % 256 for i in range(size))) if size > 0 else b""
                cur.execute(f"INSERT INTO {{table}} (id, c, b) VALUES (?, ?, ?)", (idx, text, blob))
                conn.commit()
                cur.execute(f"SELECT c, b FROM {{table}} WHERE id = ?", (idx,))
                row = cur.fetchone()
                assert row[0] == text, (size, len(row[0]), len(text))
                assert bytes(row[1]) == blob, size
            print("OK:DATA_AT_EXEC_MATRIX")
        finally:
            try:
                cur.execute(f"DROP TABLE {{table}}")
                conn.commit()
            except Exception:
                pass
            cur.close()
            conn.close()
        """
    )
    result = run_in_subprocess(code, timeout=240)
    _assert_subprocess_ok(result, "DATA_AT_EXEC matrix subprocess")
    assert "OK:DATA_AT_EXEC_MATRIX" in result.stdout
