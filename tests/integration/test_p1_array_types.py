"""User-defined VARRAY values against a real DM database."""

from __future__ import annotations

from decimal import Decimal

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


@pytest.mark.parametrize(
    ("element_type", "values"),
    [
        ("DECIMAL(30,8)", [Decimal("12345678901234567890.12345678"), None, Decimal("0.00000001")]),
        ("VARCHAR(20)", ["汉字", None, "emoji 😀"]),
        ("INTEGER", [7, None, -3]),
        ("INTEGER", []),
    ],
)
def test_varray_roundtrip(conn, table_name_factory, drop_table, element_type, values):
    type_name = table_name_factory("DMPY_ARRAY")
    table = table_name_factory("DMPY_ARRAY_TAB")
    cur = conn.cursor()
    created_type = False
    created_table = False
    closed = False
    try:
        cur.execute(f"CREATE TYPE {type_name} AS VARRAY(3) OF {element_type}")
        created_type = True
        cur.execute(f"CREATE TABLE {table} (V {type_name})")
        created_table = True
        conn.commit()

        value = dmPython.objectvar(conn, type_name)
        assert value.type.name == type_name
        assert value.valuecount == 0
        value.setvalue(values)
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()

        cur.execute(f"SELECT V FROM {table}")
        fetched = cur.fetchone()[0]
        cur.close()
        closed = True
        assert fetched.valuecount == len(values)
        assert fetched.getvalue() == values
    finally:
        conn.rollback()
        if not closed:
            cur.close()
        with conn.cursor() as cleanup:
            if created_table:
                drop_table(cleanup, table)
            if created_type:
                cleanup.execute(f"DROP TYPE {type_name}")
        conn.commit()


def test_varray_rejects_more_than_declared_elements(conn, table_name_factory):
    type_name = table_name_factory("DMPY_ARRAY_LIMIT")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TYPE {type_name} AS VARRAY(2) OF INTEGER")
        conn.commit()
        value = dmPython.objectvar(conn, type_name)
        with pytest.raises(dmPython.DatabaseError, match="position is invalid"):
            value.setvalue([1, 2, 3])
    finally:
        conn.rollback()
        cur.execute(f"DROP TYPE {type_name}")
        conn.commit()
        cur.close()


def test_object_with_varray_keeps_decimal_elements(conn, table_name_factory, drop_table):
    array_type = table_name_factory("DMPY_INNER_ARRAY")
    object_type = table_name_factory("DMPY_ARRAY_OBJECT")
    table = table_name_factory("DMPY_ARRAY_OBJECT_TAB")
    values = [Decimal("12345678901234567890.12345678"), Decimal("0.00000001")]
    cur = conn.cursor()
    created = []
    closed = False
    try:
        cur.execute(f"CREATE TYPE {array_type} AS VARRAY(3) OF DECIMAL(30,8)")
        created.append(("TYPE", array_type))
        cur.execute(f"CREATE TYPE {object_type} AS OBJECT (ID INTEGER, ITEMS {array_type})")
        created.append(("TYPE", object_type))
        cur.execute(f"CREATE TABLE {table} (V {object_type})")
        created.append(("TABLE", table))
        conn.commit()

        value = dmPython.objectvar(conn, object_type)
        value.setvalue([7, values])
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()

        cur.execute(f"SELECT V FROM {table}")
        fetched = cur.fetchone()[0]
        cur.close()
        closed = True
        assert fetched.getvalue() == [7, values]
    finally:
        conn.rollback()
        if not closed:
            cur.close()
        with conn.cursor() as cleanup:
            for kind, name in reversed(created):
                if kind == "TABLE":
                    drop_table(cleanup, name)
                else:
                    cleanup.execute(f"DROP TYPE {name}")
        conn.commit()
