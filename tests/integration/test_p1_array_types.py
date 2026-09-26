"""User-defined VARRAY values against a real DM database."""

from __future__ import annotations

import os
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


def test_varray_of_objects_preserves_null_members(conn, table_name_factory, drop_table):
    object_type = table_name_factory("DMPY_ARRAY_ITEM")
    array_type = table_name_factory("DMPY_OBJECT_ARRAY")
    table = table_name_factory("DMPY_OBJECT_ARRAY_TAB")
    values = [
        [Decimal("12345678901234567890.12345678"), "汉字"],
        None,
        [None, "emoji 😀"],
    ]
    cur = conn.cursor()
    created = []
    try:
        cur.execute(f"CREATE TYPE {object_type} AS OBJECT (AMOUNT DECIMAL(30,8), LABEL VARCHAR(20))")
        created.append(("TYPE", object_type))
        cur.execute(f"CREATE TYPE {array_type} AS VARRAY(3) OF {object_type}")
        created.append(("TYPE", array_type))
        cur.execute(f"CREATE TABLE {table} (V {array_type})")
        created.append(("TABLE", table))
        conn.commit()

        value = dmPython.objectvar(conn, array_type)
        value.setvalue(values)
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()

        cur.execute(f"SELECT V FROM {table}")
        assert cur.fetchone()[0].getvalue() == values
    finally:
        conn.rollback()
        with conn.cursor() as cleanup:
            for kind, name in reversed(created):
                if kind == "TABLE":
                    drop_table(cleanup, name)
                else:
                    cleanup.execute(f"DROP TYPE {name}")
        conn.commit()
        cur.close()


def test_varray_of_varray_keeps_decimal_elements(conn, table_name_factory, drop_table):
    inner_type = table_name_factory("DMPY_NEST_INNER")
    outer_type = table_name_factory("DMPY_NEST_OUTER")
    table = table_name_factory("DMPY_NEST_ARRAY_TAB")
    values = [[Decimal("12345678901234567890.12345678"), None], [Decimal("0.00000001")]]
    cur = conn.cursor()
    created = []
    try:
        cur.execute(f"CREATE TYPE {inner_type} AS VARRAY(2) OF DECIMAL(30,8)")
        created.append(("TYPE", inner_type))
        cur.execute(f"CREATE TYPE {outer_type} AS VARRAY(2) OF {inner_type}")
        created.append(("TYPE", outer_type))
        cur.execute(f"CREATE TABLE {table} (V {outer_type})")
        created.append(("TABLE", table))
        conn.commit()

        value = dmPython.objectvar(conn, outer_type)
        value.setvalue(values)
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()

        cur.execute(f"SELECT V FROM {table}")
        assert cur.fetchone()[0].getvalue() == values
    finally:
        conn.rollback()
        with conn.cursor() as cleanup:
            for kind, name in reversed(created):
                if kind == "TABLE":
                    drop_table(cleanup, name)
                else:
                    cleanup.execute(f"DROP TYPE {name}")
        conn.commit()
        cur.close()


def test_cross_schema_varray_of_objects(conn, conn_params, table_name_factory):
    admin_password = os.environ.get("DM_CI_ADMIN_PASSWORD")
    if not admin_password:
        pytest.skip("cross-schema type regression requires the admin test password")

    object_type = table_name_factory("DMPY_SHARED_ITEM")
    array_type = table_name_factory("DMPY_SHARED_ARRAY")
    table = table_name_factory("DMPY_SHARED_TAB")
    local_table = table_name_factory("DMPY_REF_SHARED")
    values = [[Decimal("12345678901234567890.12345678"), "汉字"], None]
    admin = dmPython.connect(**{**conn_params, "user": "SYSDBA", "password": admin_password})
    created = []
    local_created = []
    try:
        with admin.cursor() as cur:
            cur.execute(f"CREATE TYPE {object_type} AS OBJECT (AMOUNT DECIMAL(30,8), LABEL VARCHAR(20))")
            created.append(("TYPE", object_type))
            cur.execute(f"CREATE TYPE {array_type} AS VARRAY(2) OF {object_type}")
            created.append(("TYPE", array_type))
            cur.execute(f"CREATE TABLE {table} (V {array_type})")
            created.append(("TABLE", table))
            cur.execute(f"GRANT EXECUTE ON {object_type} TO {conn_params['user']}")
            cur.execute(f"GRANT EXECUTE ON {array_type} TO {conn_params['user']}")
            cur.execute(f"GRANT SELECT, INSERT ON {table} TO {conn_params['user']}")
        admin.commit()

        # A same-named type in the reader schema must not shadow SYSDBA's type.
        with conn.cursor() as cur:
            cur.execute(f"CREATE TYPE {object_type} AS OBJECT (FLAG INTEGER)")
            local_created.append(("TYPE", object_type))
            cur.execute(f"CREATE TYPE {array_type} AS VARRAY(2) OF {object_type}")
            local_created.append(("TYPE", array_type))
            cur.execute(f"CREATE TABLE {local_table} (V SYSDBA.{array_type})")
            local_created.append(("TABLE", local_table))
        conn.commit()

        value = dmPython.objectvar(conn, array_type, schema="SYSDBA")
        assert value.type.schema == "SYSDBA"
        value.setvalue(values)
        with conn.cursor() as cur:
            cur.execute(f"INSERT INTO SYSDBA.{table} VALUES (?)", (value,))
            cur.execute(f"INSERT INTO {local_table} VALUES (?)", (value,))
            conn.commit()
            for source in (f"SYSDBA.{table}", local_table):
                cur.execute(f"SELECT V FROM {source}")
                fetched = cur.fetchone()[0]
                assert fetched.type.schema == "SYSDBA"
                assert fetched.getvalue() == values
    finally:
        conn.rollback()
        with conn.cursor() as cur:
            for kind, name in reversed(local_created):
                cur.execute(f"DROP {kind} {name}")
        conn.commit()
        with admin.cursor() as cur:
            for kind, name in reversed(created):
                cur.execute(f"DROP {kind} {name}")
        admin.commit()
        admin.close()
