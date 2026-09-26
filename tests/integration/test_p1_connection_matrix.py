"""Common connection forms and options against a real DM database."""

from __future__ import annotations

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


@pytest.mark.parametrize("address", ["server", "host", "dsn"])
def test_connection_address_forms(conn_params, address):
    params = {"user": conn_params["user"], "password": conn_params["password"]}
    host = conn_params["server"]
    port = conn_params["port"]
    if address == "dsn":
        params["dsn"] = f"{host}:{port}"
    else:
        params[address] = host
        params["port"] = port
    with dmPython.connect(**params) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)
        assert conn.dsn == f"{host}:{port}"


@pytest.mark.parametrize("cursorclass", [dmPython.TupleCursor, dmPython.DictCursor])
def test_cursorclass_result_shape(conn_params, cursorclass):
    with dmPython.connect(**conn_params, cursorclass=cursorclass) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1 AS PROBE")
            row = cur.fetchone()
            if cursorclass == dmPython.DictCursor:
                assert row == {"PROBE": 1}
            else:
                assert row == (1,)


def test_schema_option_selects_test_schema(conn_params):
    with dmPython.connect(**conn_params, schema=conn_params["user"]) as conn:
        assert str(conn.current_schema).upper() == str(conn_params["user"]).upper()


@pytest.mark.parametrize("autocommit", [dmPython.DSQL_AUTOCOMMIT_OFF, dmPython.DSQL_AUTOCOMMIT_ON])
def test_autocommit_connect_option(conn_params, autocommit):
    with dmPython.connect(**conn_params, autoCommit=autocommit) as conn:
        assert int(conn.autocommit) == int(autocommit)
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_autocommit_on_persists_without_explicit_commit(
    conn, conn_params, table_name_factory, drop_table
):
    table = table_name_factory("DMPY_AUTOCOMMIT")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v INTEGER)")
        conn.commit()
        with dmPython.connect(**conn_params, autoCommit=dmPython.DSQL_AUTOCOMMIT_ON) as writer:
            with writer.cursor() as writer_cur:
                writer_cur.execute(f"INSERT INTO {table} VALUES (1)")
        cur.execute(f"SELECT COUNT(*) FROM {table}")
        assert cur.fetchone() == (1,)
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()


def test_isolation_connect_option(conn_params):
    with dmPython.connect(**conn_params, txn_isolation=dmPython.ISO_LEVEL_READ_COMMITTED) as conn:
        assert int(conn.txn_isolation) == int(dmPython.ISO_LEVEL_READ_COMMITTED)


def test_host_and_server_cannot_both_be_set(conn_params):
    with pytest.raises(dmPython.NotSupportedError, match="host or server"):
        dmPython.connect(**conn_params, host=conn_params["server"])


def test_port_rejects_non_numeric_value(conn_params):
    with pytest.raises((ValueError, TypeError, dmPython.Error)):
        dmPython.connect(**{**conn_params, "port": object()})


@pytest.mark.parametrize(
    "option",
    [
        {"connection_timeout": 5},
        {"login_timeout": 5},
        {"compress_msg": 0},
        {"use_stmt_pool": 1},
    ],
)
def test_optional_setting_connects_and_queries(conn_params, option):
    with dmPython.connect(**conn_params, **option) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)
