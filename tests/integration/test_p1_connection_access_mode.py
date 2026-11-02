"""Connection access mode and catalog must reflect the real DM session."""

from __future__ import annotations

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


@pytest.fixture()
def access_mode_table(conn_params, table_name_factory):
    name = table_name_factory("DMPY_TEST_ACCESS")
    conn = dmPython.connect(**conn_params, autoCommit=True)
    try:
        with conn.cursor() as cur:
            cur.execute(f"CREATE TABLE {name} (ID INT PRIMARY KEY)")
            cur.execute(f"INSERT INTO {name} VALUES (1)")
        yield name
    finally:
        try:
            with conn.cursor() as cur:
                cur.execute(f"DROP TABLE {name}")
        finally:
            conn.close()


def assert_read_only(conn, table):
    assert conn.access_mode == dmPython.DSQL_MODE_READ_ONLY
    with conn.cursor() as cur:
        cur.execute(f"SELECT COUNT(*) FROM {table}")
        assert cur.fetchone() == (1,)
        # A zero-row write still requires write access, without modifying rows
        # when this regression is run against the original buggy library.
        with pytest.raises(dmPython.DatabaseError, match="6506"):
            cur.execute(f"DELETE FROM {table} WHERE ID = ?", (-1,))


@pytest.mark.parametrize("autocommit", [False, True])
def test_read_only_access_mode_is_enforced(conn_params, access_mode_table, autocommit):
    with dmPython.connect(
        **conn_params,
        autoCommit=autocommit,
        access_mode=dmPython.DSQL_MODE_READ_ONLY,
    ) as conn:
        assert_read_only(conn, access_mode_table)


def test_read_only_survives_transaction_and_autocommit_changes(conn_params, access_mode_table):
    with dmPython.connect(
        **conn_params, access_mode=dmPython.DSQL_MODE_READ_ONLY, autoCommit=True
    ) as conn:
        for autocommit in [False, True, False, True]:
            conn.autoCommit = autocommit
            assert_read_only(conn, access_mode_table)
            conn.commit()
            assert_read_only(conn, access_mode_table)
            conn.rollback()
            assert_read_only(conn, access_mode_table)


@pytest.mark.parametrize("autocommit", [False, True])
def test_access_mode_changes_rebind_existing_statements(conn_params, access_mode_table, autocommit):
    with dmPython.connect(**conn_params, autoCommit=autocommit) as conn:
        with conn.cursor() as cur:
            sql = f"DELETE FROM {access_mode_table} WHERE ID = ?"
            cur.execute(sql, (-1,))
            conn.commit()
            conn.access_mode = dmPython.DSQL_MODE_READ_ONLY
            with pytest.raises(dmPython.DatabaseError, match="6506"):
                cur.execute(sql, (-1,))
            conn.rollback()
            conn.access_mode = dmPython.DSQL_MODE_READ_WRITE
            assert conn.access_mode == dmPython.DSQL_MODE_READ_WRITE
            cur.execute(sql, (-1,))
            conn.commit()
            cur.execute(f"SELECT COUNT(*) FROM {access_mode_table}")
            assert cur.fetchone() == (1,)


def test_invalid_access_mode_does_not_relax_read_only(conn_params, access_mode_table):
    with dmPython.connect(
        **conn_params, autoCommit=True, access_mode=dmPython.DSQL_MODE_READ_ONLY
    ) as conn:
        with pytest.raises(dmPython.DatabaseError, match="access_mode"):
            conn.access_mode = 2
        assert_read_only(conn, access_mode_table)


def test_current_catalog_comes_from_the_connected_database(conn_params):
    with dmPython.connect(
        **conn_params, autoCommit=True, access_mode=dmPython.DSQL_MODE_READ_ONLY
    ) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT NAME FROM V$DATABASE")
            expected = cur.fetchone()[0]
        assert conn.current_catalog == expected
        assert expected
