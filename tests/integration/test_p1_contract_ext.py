from __future__ import annotations

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


def test_executemany_generator_partial_failure_contract(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P1_EXM")
    cur = conn.cursor()
    original_autocommit = int(conn.autocommit)
    try:
        conn.autocommit = 0
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(32))")
        conn.commit()

        def row_gen():
            yield (1, "ok_1")
            yield (2, "ok_2")
            yield (1, "dup_pk")

        with pytest.raises(dmPython.DatabaseError) as excinfo:
            cur.executemany(f"INSERT INTO {table_name} (id, v) VALUES (?, ?)", row_gen())
        assert "[CODE:" in str(excinfo.value)

        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        count_before_rollback = int(cur.fetchone()[0])
        assert count_before_rollback >= 1

        conn.rollback()
        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        count_after_rollback = int(cur.fetchone()[0])
        assert count_after_rollback == 0
    finally:
        try:
            drop_table(cur, table_name)
            conn.commit()
        except Exception:
            pass
        conn.autocommit = original_autocommit
        cur.close()


def test_cursor_reuse_after_statement_error_contract(cursor):
    with pytest.raises(dmPython.DatabaseError):
        cursor.execute("SELECT * FROM TABLE_NOT_EXISTS_ABC")

    cursor.execute("SELECT 1")
    assert cursor.fetchone() == (1,)


def test_not_supported_error_message_contract_ext(cursor):
    not_supported_error = getattr(dmPython, "NotSupportedError", Exception)

    with pytest.raises(not_supported_error) as parse_error:
        cursor.parse("SELECT 1")
    assert "not support" in str(parse_error.value).lower()

    with pytest.raises(not_supported_error) as arrayvar_error:
        cursor.arrayvar(int, [1, 2, 3])
    assert "not support" in str(arrayvar_error.value).lower()

    with pytest.raises(not_supported_error) as bindnames_error:
        cursor.bindnames()
    assert "not support" in str(bindnames_error.value).lower()


def test_callproc_callfunc_argument_error_contract(cursor):
    with pytest.raises(dmPython.DatabaseError) as proc_error:
        cursor.callproc("sp_not_exists_for_contract", [1, "x"])
    proc_text = str(proc_error.value)
    assert proc_text
    assert "[CODE:" in proc_text

    with pytest.raises(dmPython.DatabaseError) as func_error:
        cursor.callfunc("fn_not_exists_for_contract", int, [1, "x"])
    func_text = str(func_error.value)
    assert func_text
    assert "[CODE:" in func_text
