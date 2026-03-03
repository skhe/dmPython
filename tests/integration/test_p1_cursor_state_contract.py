from __future__ import annotations

import uuid

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


def _table_name(prefix: str) -> str:
    return f"{prefix}_{uuid.uuid4().hex[:8].upper()}"


def _proc_name(prefix: str) -> str:
    return f"{prefix}_{uuid.uuid4().hex[:8].upper()}"


def test_rowcount_lifecycle_contract(conn):
    table_name = _table_name("DMPY_P1_RC")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(32))")
        assert int(cur.rowcount) == -1

        cur.execute(f"INSERT INTO {table_name} (id, v) VALUES (?, ?)", (1, "a"))
        conn.commit()
        assert int(cur.rowcount) == 1

        cur.execute(f"UPDATE {table_name} SET v = ? WHERE id = ?", ("b", 1))
        conn.commit()
        assert int(cur.rowcount) == 1

        cur.execute(f"DELETE FROM {table_name} WHERE id = ?", (1,))
        conn.commit()
        assert int(cur.rowcount) == 1

        cur.execute(f"INSERT INTO {table_name} (id, v) VALUES (?, ?)", (2, "c"))
        conn.commit()
        cur.execute(f"SELECT id, v FROM {table_name} ORDER BY id")
        row = cur.fetchone()
        assert row == (2, "c")
        assert int(cur.rowcount) >= 1
    finally:
        try:
            cur.execute(f"DROP TABLE {table_name}")
            conn.commit()
        except Exception:
            pass
        cur.close()


def test_lastrowid_contract_dml_paths(conn):
    table_name = _table_name("DMPY_P1_LRID")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(32))")
        conn.commit()
        cur.execute(f"INSERT INTO {table_name} (id, v) VALUES (?, ?)", (1, "a"))
        conn.commit()
        lastrowid = cur.lastrowid
        assert (lastrowid is None) or isinstance(lastrowid, (int, str, bytes, bytearray))

        cur.execute(f"UPDATE {table_name} SET v = ? WHERE id = ?", ("b", 1))
        conn.commit()
        _ = cur.lastrowid
    finally:
        try:
            cur.execute(f"DROP TABLE {table_name}")
            conn.commit()
        except Exception:
            pass
        cur.close()


def test_error_object_contract_dmerror_fields(cursor):
    with pytest.raises(dmPython.DatabaseError) as excinfo:
        cursor.execute("SELECT * FROM NO_SUCH_TABLE_ABC")

    assert excinfo.value.args
    dm_error = excinfo.value.args[0]
    assert type(dm_error).__name__ == "DmError"
    assert hasattr(dm_error, "code")
    assert hasattr(dm_error, "message")
    assert hasattr(dm_error, "context")
    assert isinstance(dm_error.code, int)
    assert isinstance(dm_error.message, str) and dm_error.message
    assert isinstance(dm_error.context, str) and dm_error.context


def test_executemany_partial_failure_autocommit_matrix(conn):
    table_name = _table_name("DMPY_P1_ACM")
    cur = conn.cursor()
    original_autocommit = int(conn.autocommit)
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, v VARCHAR(32))")
        conn.commit()

        conn.autocommit = 0
        with pytest.raises(dmPython.DatabaseError):
            cur.executemany(
                f"INSERT INTO {table_name} (id, v) VALUES (?, ?)",
                [(1, "a"), (2, "b"), (1, "dup")],
            )
        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert int(cur.fetchone()[0]) == 2
        conn.rollback()
        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert int(cur.fetchone()[0]) == 0

        conn.autocommit = 1
        with pytest.raises(dmPython.DatabaseError):
            cur.executemany(
                f"INSERT INTO {table_name} (id, v) VALUES (?, ?)",
                [(11, "x"), (12, "y"), (11, "dup")],
            )
        cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert int(cur.fetchone()[0]) == 2
    finally:
        conn.autocommit = original_autocommit
        try:
            cur.execute(f"DROP TABLE {table_name}")
            conn.commit()
        except Exception:
            pass
        cur.close()


def test_nextset_contract_when_not_supported(cursor):
    cursor.execute("SELECT 1")
    assert cursor.nextset() is None


def test_output_stream_and_setoutputsize_contract(cursor):
    assert int(cursor.output_stream) == 0
    cursor.output_stream = 1
    assert int(cursor.output_stream) == 1
    cursor.output_stream = 0
    assert int(cursor.output_stream) == 0

    cursor.setoutputsize(2048)
    cursor.setoutputsize(1024, 0)


def test_callproc_minimal_success_contract(conn):
    proc_name = _proc_name("P_DMPY_P1_MIN")
    cur = conn.cursor()
    ddl = f"""
    CREATE OR REPLACE PROCEDURE {proc_name}(p_in IN INT) AS
    BEGIN
      NULL;
    END;
    """
    try:
        cur.execute(ddl)
        conn.commit()
        result = cur.callproc(proc_name, [41])
        assert isinstance(result, list)
        assert result == [41]
    finally:
        try:
            cur.execute(f"DROP PROCEDURE {proc_name}")
            conn.commit()
        except Exception:
            pass
        cur.close()
