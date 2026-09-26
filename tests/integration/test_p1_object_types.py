"""User-defined object values against a real DM database."""

from __future__ import annotations

from decimal import Decimal

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


def test_object_roundtrip_keeps_decimal_and_unicode(conn, table_name_factory, drop_table):
    type_name = table_name_factory("DMPY_OBJ")
    table = table_name_factory("DMPY_OBJ_TAB")
    amount = Decimal("12345678901234567890.12345678")
    cur = conn.cursor()
    created_type = False
    created_table = False
    closed = False
    try:
        cur.execute(
            f"CREATE TYPE {type_name} AS OBJECT "
            "(ID INTEGER, AMOUNT DECIMAL(30,8), LABEL VARCHAR(20))"
        )
        created_type = True
        cur.execute(f"CREATE TABLE {table} (V {type_name})")
        created_table = True
        conn.commit()

        value = dmPython.objectvar(conn, type_name)
        assert value.type.name == type_name
        assert len(value.type.attributes) == 3
        assert value.valuecount == 3
        value.setvalue([7, amount, "汉字"])
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()

        cur.execute(f"SELECT V FROM {table}")
        fetched = cur.fetchone()[0]
        cur.close()
        closed = True
        assert fetched.getvalue() == [7, amount, "汉字"]
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


def test_nested_object_roundtrip(conn, table_name_factory, drop_table):
    child = table_name_factory("DMPY_CHILD")
    parent = table_name_factory("DMPY_PARENT")
    table = table_name_factory("DMPY_NEST_TAB")
    amount = Decimal("12345678901234567890.12345678")
    cur = conn.cursor()
    created_types = []
    created_table = False
    closed = False
    try:
        cur.execute(f"CREATE TYPE {child} AS OBJECT (AMOUNT DECIMAL(30,8), LABEL VARCHAR(20))")
        created_types.append(child)
        cur.execute(f"CREATE TYPE {parent} AS OBJECT (ID INTEGER, ITEM {child})")
        created_types.append(parent)
        cur.execute(f"CREATE TABLE {table} (V {parent})")
        created_table = True
        conn.commit()

        value = dmPython.objectvar(conn, parent)
        value.setvalue([7, [amount, "汉字"]])
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()

        cur.execute(f"SELECT V FROM {table}")
        fetched = cur.fetchone()[0]
        cur.close()
        closed = True
        assert fetched.getvalue() == [7, [amount, "汉字"]]
    finally:
        conn.rollback()
        if not closed:
            cur.close()
        with conn.cursor() as cleanup:
            if created_table:
                drop_table(cleanup, table)
            for type_name in reversed(created_types):
                cleanup.execute(f"DROP TYPE {type_name}")
        conn.commit()


def test_unknown_object_type_reports_error(conn, table_name_factory):
    type_name = table_name_factory("DMPY_MISSING")
    with pytest.raises(dmPython.DatabaseError, match="not visible"):
        dmPython.objectvar(conn, type_name)
