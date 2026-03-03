from __future__ import annotations

import textwrap

import pytest


pytestmark = [pytest.mark.requires_dm, pytest.mark.p0_stability]

_PROBLEM_PATTERNS = ("稳态中文🚀X", "A中文🚀", "🚀🚀🚀A")
_SIZES = (16000, 16384, 40000)


def _make_text(base: str, size: int) -> str:
    if size <= 0:
        return ""
    return (base * ((size // len(base)) + 2))[:size]


def test_clob_unicode_problem_patterns_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_CLOBUNI")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB)")
        rid = 1
        expected = {}
        for pattern in _PROBLEM_PATTERNS:
            for size in _SIZES:
                payload = _make_text(pattern, size)
                expected[rid] = payload
                cur.execute(f"INSERT INTO {table_name} (id, c) VALUES (?, ?)", (rid, payload))
                rid += 1
        conn.commit()

        cur.execute(f"SELECT id, c FROM {table_name} ORDER BY id")
        rows = cur.fetchall()
        assert len(rows) == len(expected)
        for row_id, value in rows:
            payload = expected[row_id]
            assert isinstance(value, str)
            assert len(value) == len(payload)
            assert value == payload
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_clob_unicode_problem_patterns_executemany_roundtrip(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_CLOBEXM")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB)")
        rows = []
        rid = 1
        for pattern in _PROBLEM_PATTERNS:
            for size in _SIZES:
                rows.append((rid, _make_text(pattern, size)))
                rid += 1
        cur.executemany(f"INSERT INTO {table_name} (id, c) VALUES (?, ?)", rows)
        conn.commit()

        cur.execute(f"SELECT id, c FROM {table_name} ORDER BY id")
        got = cur.fetchall()
        assert len(got) == len(rows)
        for (row_id, expected), (_, value) in zip(rows, got):
            assert isinstance(value, str)
            assert len(value) == len(expected)
            assert value == expected
            assert row_id >= 1
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_clob_unicode_problem_patterns_length_contract(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P0_CLOBLEN")
    cur = conn.cursor()
    length_stable_cases = {
        ("稳态中文🚀X", 16000),
        ("稳态中文🚀X", 16384),
        ("A中文🚀", 16000),
        ("A中文🚀", 16384),
    }
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, p VARCHAR(20), s INT, c CLOB)")
        rid = 1
        expected = {}
        for pattern in _PROBLEM_PATTERNS:
            for size in _SIZES:
                payload = _make_text(pattern, size)
                expected[rid] = (pattern, size, payload)
                cur.execute(
                    f"INSERT INTO {table_name} (id, p, s, c) VALUES (?, ?, ?, ?)",
                    (rid, pattern, size, payload),
                )
                rid += 1
        conn.commit()

        for row_id, (pattern, size, payload) in expected.items():
            cur.execute(f"SELECT length(c), c FROM {table_name} WHERE id = ?", (row_id,))
            db_len, value = cur.fetchone()
            assert isinstance(value, str)
            assert len(value) == len(payload)
            assert value == payload
            if (pattern, size) in length_stable_cases:
                assert int(db_len) == len(payload)
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


@pytest.mark.crash_guard
def test_clob_unicode_problem_patterns_subprocess_no_crash(run_in_subprocess):
    code = textwrap.dedent(
        f"""
        import os
        import uuid
        import dmPython

        patterns = {list(_PROBLEM_PATTERNS)!r}
        sizes = {list(_SIZES)!r}

        def make_text(base, size):
            if size <= 0:
                return ""
            return (base * ((size // len(base)) + 2))[:size]

        conn = dmPython.connect(
            user=os.getenv("DM_TEST_USER", "SYSDBA"),
            password=os.getenv("DM_TEST_PASSWORD", "SYSDBA001"),
            server=os.getenv("DM_TEST_HOST", "localhost"),
            port=int(os.getenv("DM_TEST_PORT", "5237")),
        )
        cur = conn.cursor()
        table = "DMPY_P0_CLOBSP_" + uuid.uuid4().hex[:8].upper()
        try:
            cur.execute(f"CREATE TABLE {{table}} (id INT PRIMARY KEY, c CLOB)")
            rid = 1
            for pattern in patterns:
                for size in sizes:
                    payload = make_text(pattern, size)
                    cur.execute(f"INSERT INTO {{table}} (id, c) VALUES (?, ?)", (rid, payload))
                    rid += 1
            conn.commit()

            cur.execute(f"SELECT id, c FROM {{table}} ORDER BY id")
            got = cur.fetchall()
            rid = 1
            for pattern in patterns:
                for size in sizes:
                    expected = make_text(pattern, size)
                    row = got[rid - 1]
                    assert row[0] == rid
                    assert row[1] == expected
                    rid += 1
            print("OK:CLOB_UNICODE_REGRESSION")
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
    result = run_in_subprocess(code, timeout=300)
    assert result.returncode == 0, (
        f"subprocess failed rc={result.returncode}\nstdout:\n{result.stdout}\nstderr:\n{result.stderr}"
    )
    assert result.returncode not in (139, -11)
    assert "OK:CLOB_UNICODE_REGRESSION" in result.stdout
