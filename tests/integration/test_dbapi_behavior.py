"""DB-API behavior exercised against a real DM server."""

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


def test_positional_parameter_count(cursor):
    with pytest.raises(dmPython.ProgrammingError):
        cursor.execute("SELECT ?", ())
    with pytest.raises(dmPython.ProgrammingError):
        cursor.execute("SELECT ?", (1, 2))
    cursor.execute("SELECT ?", (3,))
    assert cursor.fetchone() == (3,)


def test_result_description_and_exhaustion(cursor):
    assert cursor.description is None
    cursor.execute("SELECT 1 AS value")
    assert len(cursor.description) == 1
    assert len(cursor.description[0]) == 7
    assert cursor.fetchone() == (1,)
    assert cursor.fetchone() is None


def test_duplicate_primary_key_is_integrity_error(conn, cursor, table_name_factory, drop_table):
    table = table_name_factory("DMPY_DBAPI_PK")
    try:
        cursor.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY)")
        cursor.execute(f"INSERT INTO {table} (id) VALUES (1)")
        with pytest.raises(dmPython.IntegrityError):
            cursor.execute(f"INSERT INTO {table} (id) VALUES (1)")
    finally:
        conn.rollback()
        drop_table(cursor, table)


def test_close_rolls_back_uncommitted_insert(conn_params, table_name_factory):
    table = table_name_factory("DMPY_DBAPI_RB")
    first = dmPython.connect(**conn_params)
    cur = first.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY)")
        first.commit()
        cur.execute(f"INSERT INTO {table} (id) VALUES (1)")
    finally:
        first.close()

    second = dmPython.connect(**conn_params)
    cur = second.cursor()
    try:
        cur.execute(f"SELECT COUNT(*) FROM {table}")
        assert cur.fetchone() == (0,)
    finally:
        cur.execute(f"DROP TABLE {table}")
        second.commit()
        second.close()
