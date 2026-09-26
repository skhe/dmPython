"""Common SQL type round trips against a real DM database."""

from __future__ import annotations

import datetime as dt
from decimal import Decimal

import pytest


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


@pytest.mark.parametrize(
    ("sql_type", "value", "expected"),
    [
        ("SMALLINT", -32768, -32768),
        ("INTEGER", 2147483647, 2147483647),
        ("BIGINT", 9223372036854775807, 9223372036854775807),
        ("DECIMAL(18, 6)", Decimal("12345.678901"), Decimal("12345.678901")),
        ("FLOAT", 1.25, Decimal("1.25")),
        ("DOUBLE", 1.25, Decimal("1.25")),
        ("BIT", True, True),
        ("VARCHAR(80)", "汉字🚀", "汉字🚀"),
        ("CHAR(4)", "AB", "AB  "),
        ("NCHAR(8)", "中文", "中文      "),
        ("NVARCHAR(32)", "中文🚀", "中文🚀"),
        ("BINARY(4)", b"\x00\xff", b"\x00\xff\x00\x00"),
        ("VARBINARY(16)", b"\x00\x01\xfe\xff", b"\x00\x01\xfe\xff"),
        ("DATE", dt.date(2024, 2, 29), dt.date(2024, 2, 29)),
        ("TIME", dt.time(23, 59, 58), "23:59:58"),
        (
            "TIMESTAMP",
            dt.datetime(2024, 2, 29, 23, 59, 58, 123456),
            "2024-02-29 23:59:58.123456",
        ),
    ],
)
def test_scalar_type_roundtrip(conn, table_name_factory, drop_table, sql_type, value, expected):
    table = table_name_factory("DMPY_TYPE")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v {sql_type})")
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()
        cur.execute(f"SELECT v FROM {table}")
        actual = cur.fetchone()[0]
        if sql_type in {"DECIMAL(18, 6)", "FLOAT", "DOUBLE"}:
            assert Decimal(str(actual)) == expected
        elif sql_type == "TIME":
            assert str(actual).split()[-1].split(".")[0] == expected
        elif sql_type == "TIMESTAMP":
            assert str(actual) == expected
        else:
            assert actual == expected
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()


@pytest.mark.parametrize("sql_type", ["INTEGER", "DECIMAL(18, 6)", "VARCHAR(80)", "VARBINARY(16)", "DATE", "TIMESTAMP"])
def test_typed_null_roundtrip(conn, table_name_factory, drop_table, sql_type):
    table = table_name_factory("DMPY_NULL")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v {sql_type})")
        cur.execute(f"INSERT INTO {table} VALUES (?)", (None,))
        conn.commit()
        cur.execute(f"SELECT v FROM {table}")
        assert cur.fetchone() == (None,)
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()
