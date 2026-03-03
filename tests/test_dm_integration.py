"""Integration tests for dmPython against a running DM database.

Defaults target a local Docker container:
  host=localhost, port=5237, user=SYSDBA, password=SYSDBA001
Override with environment variables:
  DM_TEST_HOST, DM_TEST_PORT, DM_TEST_USER, DM_TEST_PASSWORD
"""

from __future__ import annotations

import datetime as dt
import os
import uuid
from decimal import Decimal

import pytest

import dmPython

pytestmark = [pytest.mark.requires_dm]


def _conn_params() -> dict[str, object]:
    return {
        "user": os.getenv("DM_TEST_USER", "SYSDBA"),
        "password": os.getenv("DM_TEST_PASSWORD", "SYSDBA001"),
        "server": os.getenv("DM_TEST_HOST", "localhost"),
        "port": int(os.getenv("DM_TEST_PORT", "5237")),
    }


def _new_table_name(prefix: str = "DMPY_TEST") -> str:
    return f"{prefix}_{uuid.uuid4().hex[:10].upper()}"


def _drop_table(cursor, table_name: str) -> None:
    try:
        cursor.execute(f"DROP TABLE {table_name}")
    except Exception:
        # Ignore missing table or other cleanup-time failures.
        pass


@pytest.fixture()
def conn():
    c = dmPython.connect(**_conn_params())
    yield c
    c.close()


@pytest.fixture()
def cursor(conn):
    cur = conn.cursor()
    yield cur
    cur.close()


def test_connect_and_select_one(cursor):
    cursor.execute("SELECT 1")
    assert cursor.fetchone() == (1,)


def test_query_server_version(cursor):
    cursor.execute("SELECT * FROM V$VERSION")
    rows = cursor.fetchall()
    assert len(rows) > 0


def test_parameterized_insert_with_unicode_and_null(conn):
    table_name = _new_table_name()
    cur = conn.cursor()
    try:
        cur.execute(
            f"""
            CREATE TABLE {table_name} (
                id INT PRIMARY KEY,
                name VARCHAR(100),
                note VARCHAR(100)
            )
            """
        )
        cur.execute(
            f"INSERT INTO {table_name} (id, name, note) VALUES (?, ?, ?)",
            (1, "中文🚀", None),
        )
        conn.commit()

        cur.execute(f"SELECT name, note FROM {table_name} WHERE id = ?", (1,))
        row = cur.fetchone()
        assert row[0] == "中文🚀"
        assert row[1] is None
    finally:
        _drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_executemany_bulk_insert(conn):
    table_name = _new_table_name()
    cur = conn.cursor()
    try:
        cur.execute(
            f"CREATE TABLE {table_name} (id INT PRIMARY KEY, val VARCHAR(32))"
        )
        rows = [(1, "a"), (2, "b"), (3, "c"), (4, "d")]
        cur.executemany(
            f"INSERT INTO {table_name} (id, val) VALUES (?, ?)",
            rows,
        )
        conn.commit()

        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert cur.fetchone()[0] == 4
    finally:
        _drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_transaction_rollback(conn):
    table_name = _new_table_name()
    cur = conn.cursor()
    try:
        conn.autocommit = False
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY)")
        conn.commit()

        cur.execute(f"INSERT INTO {table_name} VALUES (100)")
        conn.rollback()

        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert cur.fetchone()[0] == 0
    finally:
        _drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_transaction_commit_persists(conn):
    table_name = _new_table_name()
    cur = conn.cursor()
    cur2 = conn.cursor()
    try:
        conn.autocommit = False
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY)")
        conn.commit()

        cur.execute(f"INSERT INTO {table_name} VALUES (200)")
        conn.commit()

        cur2.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert cur2.fetchone()[0] == 1
    finally:
        _drop_table(cur, table_name)
        conn.commit()
        cur2.close()
        cur.close()


def test_fetchone_fetchmany_fetchall(conn):
    table_name = _new_table_name()
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY)")
        cur.executemany(
            f"INSERT INTO {table_name} (id) VALUES (?)",
            [(1,), (2,), (3,), (4,), (5,)],
        )
        conn.commit()

        cur.execute(f"SELECT id FROM {table_name} ORDER BY id")
        assert cur.fetchone() == (1,)
        assert cur.fetchmany(2) == [(2,), (3,)]
        assert cur.fetchall() == [(4,), (5,)]
    finally:
        _drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_cursor_description(cursor):
    cursor.execute("SELECT 1 AS col_a, 'x' AS col_b")
    assert cursor.description is not None
    assert len(cursor.description) == 2
    assert cursor.description[0][0].upper() == "COL_A"
    assert cursor.description[1][0].upper() == "COL_B"


def test_datetime_round_trip(conn):
    table_name = _new_table_name()
    cur = conn.cursor()
    expected = dt.datetime(2024, 1, 2, 3, 4, 5)
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, ts TIMESTAMP)")
        cur.execute(
            f"INSERT INTO {table_name} (id, ts) VALUES (?, ?)",
            (1, expected),
        )
        conn.commit()

        cur.execute(f"SELECT ts FROM {table_name} WHERE id = 1")
        actual = cur.fetchone()[0]
        if isinstance(actual, dt.datetime):
            assert actual.replace(microsecond=0) == expected
        else:
            assert isinstance(actual, str)
            assert actual.startswith("2024-01-02 03:04:05")
    finally:
        _drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_decimal_round_trip(conn):
    table_name = _new_table_name()
    cur = conn.cursor()
    expected = Decimal("12345.678901")
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, n DECIMAL(18, 6))")
        cur.execute(
            f"INSERT INTO {table_name} (id, n) VALUES (?, ?)",
            (1, expected),
        )
        conn.commit()

        cur.execute(f"SELECT n FROM {table_name} WHERE id = 1")
        actual = cur.fetchone()[0]
        assert Decimal(str(actual)).quantize(Decimal("0.000001")) == expected
    finally:
        _drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_invalid_sql_raises_database_error(cursor):
    with pytest.raises(dmPython.DatabaseError):
        cursor.execute("SELECT * FROM TABLE_NOT_EXISTS_ABC")


def test_multiple_cursors_can_read_independently(conn):
    c1 = conn.cursor()
    c2 = conn.cursor()
    table_name = _new_table_name()
    try:
        c1.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, val VARCHAR(10))")
        c1.executemany(
            f"INSERT INTO {table_name} (id, val) VALUES (?, ?)",
            [(1, "a"), (2, "b"), (3, "c")],
        )
        conn.commit()

        c1.execute(f"SELECT id FROM {table_name} ORDER BY id")
        c2.execute(f"SELECT val FROM {table_name} ORDER BY id")

        assert c1.fetchone() == (1,)
        assert c2.fetchone() == ("a",)
        assert c1.fetchall() == [(2,), (3,)]
        assert c2.fetchall() == [("b",), ("c",)]
    finally:
        _drop_table(c1, table_name)
        conn.commit()
        c2.close()
        c1.close()
