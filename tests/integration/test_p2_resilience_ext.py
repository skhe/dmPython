from __future__ import annotations

import gc
import random
import textwrap
import threading
import uuid

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p2_scale]

_TEXT_PATTERNS = ("中文🚀A", "稳态中文🚀X", "A中文🚀", "🚀🚀🚀A", "纯中文测试")


def _table_name(prefix: str) -> str:
    return f"{prefix}_{uuid.uuid4().hex[:8].upper()}"


def _make_text(size: int) -> str:
    if size <= 0:
        return ""
    base = _TEXT_PATTERNS[(size // 1024) % len(_TEXT_PATTERNS)]
    return (base * ((size // len(base)) + 2))[:size]


def _make_blob(size: int) -> bytes:
    if size <= 0:
        return b""
    return bytes((i * 17) % 256 for i in range(size))


def test_connection_close_race_during_fetch_no_crash(run_in_subprocess):
    code = textwrap.dedent(
        """
        import threading
        import uuid
        import dmPython

        table = "DMPY_P2_RACE_" + uuid.uuid4().hex[:8].upper()
        conn = dmPython.connect(user="SYSDBA", password="SYSDBA001", server="localhost", port=5237)
        cur = conn.cursor()
        cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, v VARCHAR(64))")
        cur.executemany(f"INSERT INTO {table} (id, v) VALUES (?, ?)", [(i, f"v{i}") for i in range(1, 2001)])
        conn.commit()

        errors = []

        def fetch_worker():
            c = conn.cursor()
            try:
                c.arraysize = 64
                c.execute(f"SELECT id, v FROM {table} ORDER BY id")
                while True:
                    rows = c.fetchmany(64)
                    if not rows:
                        break
            except Exception as exc:
                errors.append(type(exc).__name__)
            finally:
                try:
                    c.close()
                except Exception:
                    pass

        def close_worker():
            try:
                conn.close()
            except Exception as exc:
                errors.append(type(exc).__name__)

        t1 = threading.Thread(target=fetch_worker)
        t1.start()
        t2 = threading.Thread(target=close_worker)
        t2.start()
        t1.join()
        t2.join()

        try:
            cur2 = dmPython.connect(user="SYSDBA", password="SYSDBA001", server="localhost", port=5237).cursor()
            cur2.execute(f"DROP TABLE {table}")
            cur2.connection.commit()
            cur2.close()
        except Exception:
            pass

        print("OK:RACE", ",".join(errors))
        """
    )
    result = run_in_subprocess(code, timeout=240)
    assert result.returncode == 0, (
        f"race subprocess failed rc={result.returncode}\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}"
    )
    assert result.returncode not in (139, -11)
    assert "OK:RACE" in result.stdout


def test_multi_cursor_same_connection_isolation_contract(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P2_MCISO")
    writer = conn.cursor()
    reader1 = conn.cursor()
    reader2 = conn.cursor()
    try:
        writer.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(64))")
        conn.commit()

        for i in range(1, 101):
            writer.execute(f"INSERT INTO {table_name} (id, v) VALUES (?, ?)", (i, f"v{i}"))
            if i % 10 == 0:
                conn.commit()
                reader1.execute(f"SELECT COUNT(*) FROM {table_name}")
                c1 = int(reader1.fetchone()[0])
                reader2.execute(f"SELECT MAX(id) FROM {table_name}")
                c2 = int(reader2.fetchone()[0])
                assert c1 == i
                assert c2 == i
        conn.commit()

        reader1.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert int(reader1.fetchone()[0]) == 100
    finally:
        drop_table(writer, table_name)
        conn.commit()
        reader2.close()
        reader1.close()
        writer.close()


def test_lob_boundary_randomized_medium_fuzz(conn, table_name_factory, drop_table, boundary_sizes):
    table_name = table_name_factory("DMPY_P2_FUZZ")
    cur = conn.cursor()
    rng = random.Random(20260303)
    pool = [size for size in boundary_sizes if size <= 16384]
    loops = 200
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB, b BLOB)")
        for i in range(1, loops + 1):
            size = rng.choice(pool)
            c_payload = _make_text(size)
            b_payload = _make_blob(size)
            cur.execute(
                f"INSERT INTO {table_name} (id, c, b) VALUES (?, ?, ?)",
                (i, c_payload, b_payload),
            )
            if i % 20 == 0:
                conn.commit()
        conn.commit()

        cur.execute(f"SELECT id, c, b FROM {table_name} ORDER BY id")
        rows = cur.fetchall()
        assert len(rows) == loops
        rng_check = random.Random(20260303)
        expected_sizes = [rng_check.choice(pool) for _ in range(loops)]
        for idx, row in enumerate(rows, start=1):
            size = expected_sizes[idx - 1]
            assert row[0] == idx
            assert row[1] == _make_text(size)
            assert bytes(row[2]) == _make_blob(size)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_reconnect_after_error_burst_contract(conn_params):
    conn = dmPython.connect(**conn_params)
    cur = conn.cursor()
    try:
        for _ in range(30):
            with pytest.raises(dmPython.DatabaseError):
                cur.execute("SELECT * FROM NO_SUCH_TABLE_BURST_ABC")
        cur.execute("SELECT 1")
        assert cur.fetchone() == (1,)
    finally:
        cur.close()
        conn.close()

    conn2 = dmPython.connect(**conn_params)
    cur2 = conn2.cursor()
    try:
        cur2.execute("SELECT 1")
        assert cur2.fetchone() == (1,)
    finally:
        cur2.close()
        conn2.close()


def test_churn_gc_loops_env_controlled(conn_params, churn_loops, gc_loops):
    for i in range(churn_loops):
        conn = dmPython.connect(**conn_params)
        cur = conn.cursor()
        try:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)
        finally:
            cur.close()
            conn.close()

    for i in range(gc_loops):
        conn = dmPython.connect(**conn_params)
        cur = conn.cursor()
        try:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)
        finally:
            cur.close()
            conn.close()
        if i % 5 == 0:
            gc.collect()
