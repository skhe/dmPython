from __future__ import annotations

import uuid

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


def test_context_manager_success_and_exception_paths(conn_params):
    with dmPython.connect(**conn_params) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)

    with pytest.raises(Exception):
        conn.cursor()

    conn2 = None
    with pytest.raises(RuntimeError):
        with dmPython.connect(**conn_params) as conn2:
            with conn2.cursor() as cur2:
                cur2.execute("SELECT 1")
                assert cur2.fetchone() == (1,)
                raise RuntimeError("force context exit")

    assert conn2 is not None
    with pytest.raises(Exception):
        conn2.cursor()


def test_prepare_execute_contract(conn):
    cur = conn.cursor()
    table_name = f"DMPY_PREP_{uuid.uuid4().hex[:8].upper()}"
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(20))")
        cur.prepare(f"INSERT INTO {table_name} VALUES (?, ?)")
        assert "INSERT INTO" in str(cur.statement).upper()

        cur.execute(f"INSERT INTO {table_name} VALUES (?, ?)", (1, "ok"))
        conn.commit()

        cur.execute(f"SELECT v FROM {table_name} WHERE id = 1")
        assert cur.fetchone() == ("ok",)
    finally:
        try:
            cur.execute(f"DROP TABLE {table_name}")
            conn.commit()
        except Exception:
            pass
        cur.close()


def test_setinputsizes_and_var_contract(cursor):
    bind_vars = cursor.setinputsizes(int, str)
    assert isinstance(bind_vars, list)
    assert len(bind_vars) == 2

    var_obj = cursor.var(int)
    assert var_obj is not None
    assert "BIGINT" in type(var_obj).__name__.upper()


def test_callproc_callfunc_contract(cursor):
    with pytest.raises(dmPython.DatabaseError):
        cursor.callproc("sp_not_exists_for_contract", [])

    with pytest.raises(dmPython.DatabaseError):
        cursor.callfunc("fn_not_exists_for_contract", int, [])


def test_not_supported_api_contract(cursor):
    not_supported_error = getattr(dmPython, "NotSupportedError", Exception)

    with pytest.raises(not_supported_error):
        cursor.parse("SELECT 1")

    with pytest.raises(not_supported_error):
        cursor.arrayvar(int, [1, 2, 3])

    with pytest.raises(not_supported_error):
        cursor.bindnames()


def test_connection_attr_contract(conn):
    original_autocommit = int(conn.autocommit)
    conn.autocommit = 1
    assert int(conn.autocommit) == 1
    conn.autocommit = 0
    assert int(conn.autocommit) == 0
    conn.autocommit = original_autocommit

    current_iso = int(conn.txn_isolation)
    conn.txn_isolation = 2
    assert int(conn.txn_isolation) == 2
    conn.txn_isolation = current_iso

    assert isinstance(str(conn.version), str)
    assert len(str(conn.version)) > 0
