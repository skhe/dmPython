from __future__ import annotations

import textwrap

import pytest


pytestmark = [pytest.mark.requires_dm, pytest.mark.p0_stability]


def _assert_subprocess_ok(result, label: str) -> None:
    assert result.returncode == 0, (
        f"{label} failed\n"
        f"stdout:\n{result.stdout}\n"
        f"stderr:\n{result.stderr}"
    )


@pytest.mark.crash_guard
def test_lob_clob_roundtrip_no_crash(run_in_subprocess):
    code = textwrap.dedent(
        """
        import uuid
        import dmPython

        table = "DMPY_CLOB_" + uuid.uuid4().hex[:8].upper()
        conn = dmPython.connect(
            user="SYSDBA",
            password="SYSDBA001",
            server="localhost",
            port=5237,
        )
        cur = conn.cursor()
        try:
            payload = ("中文🚀" * 700)[:2000]
            cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, c CLOB)")
            cur.execute(f"INSERT INTO {table} (id, c) VALUES (?, ?)", (1, payload))
            conn.commit()
            cur.execute(f"SELECT c FROM {table} WHERE id = 1")
            value = cur.fetchone()[0]
            assert isinstance(value, str), type(value)
            assert value == payload
            print("OK:CLOB")
        finally:
            try:
                cur.execute(f"DROP TABLE {table}")
                conn.commit()
            except Exception:
                pass
            cur.close()
            conn.close()
        """
    )
    result = run_in_subprocess(code)
    _assert_subprocess_ok(result, "CLOB roundtrip")
    assert "OK:CLOB" in result.stdout


@pytest.mark.crash_guard
def test_lob_blob_roundtrip_no_crash(run_in_subprocess):
    code = textwrap.dedent(
        """
        import uuid
        import dmPython

        table = "DMPY_BLOB_" + uuid.uuid4().hex[:8].upper()
        conn = dmPython.connect(
            user="SYSDBA",
            password="SYSDBA001",
            server="localhost",
            port=5237,
        )
        cur = conn.cursor()
        try:
            payload = bytes(range(256)) * 12
            cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, b BLOB)")
            cur.execute(f"INSERT INTO {table} (id, b) VALUES (?, ?)", (1, payload))
            conn.commit()
            cur.execute(f"SELECT b FROM {table} WHERE id = 1")
            value = cur.fetchone()[0]
            assert isinstance(value, (bytes, bytearray)), type(value)
            assert bytes(value) == payload
            print("OK:BLOB")
        finally:
            try:
                cur.execute(f"DROP TABLE {table}")
                conn.commit()
            except Exception:
                pass
            cur.close()
            conn.close()
        """
    )
    result = run_in_subprocess(code)
    _assert_subprocess_ok(result, "BLOB roundtrip")
    assert "OK:BLOB" in result.stdout


@pytest.mark.crash_guard
def test_lob_error_path_no_segfault_subprocess(run_in_subprocess):
    code = textwrap.dedent(
        """
        import gc
        import uuid
        import dmPython

        table = "DMPY_LERR_" + uuid.uuid4().hex[:8].upper()
        conn = dmPython.connect(
            user="SYSDBA",
            password="SYSDBA001",
            server="localhost",
            port=5237,
        )
        cur = conn.cursor()
        try:
            payload = "x" * 3000
            cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, c CLOB)")
            cur.execute(f"INSERT INTO {table} (id, c) VALUES (?, ?)", (1, payload))
            conn.commit()
            cur.execute(f"SELECT c FROM {table} WHERE id = 1")
            row = cur.fetchone()
            assert row and isinstance(row[0], str)
            gc.collect()
            print("OK:NO_SEGFAULT")
        finally:
            try:
                cur.execute(f"DROP TABLE {table}")
                conn.commit()
            except Exception:
                pass
            cur.close()
            conn.close()
        """
    )
    result = run_in_subprocess(code)
    _assert_subprocess_ok(result, "LOB crash guard")
    assert "OK:NO_SEGFAULT" in result.stdout


def test_closed_handle_operations_raise_not_crash(conn):
    cur = conn.cursor()
    cur.execute("SELECT 1")
    cur.close()

    with pytest.raises(Exception):
        cur.execute("SELECT 1")

    conn.close()
    with pytest.raises(Exception):
        conn.cursor()
