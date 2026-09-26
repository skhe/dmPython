"""Common SQL type round trips against a real DM database."""

from __future__ import annotations

import datetime as dt
import re
from decimal import Decimal

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]

_TZ_PLUS_0530 = dt.timezone(dt.timedelta(hours=5, minutes=30))
_TZ_MINUS_0400 = dt.timezone(-dt.timedelta(hours=4))
_AWARE_TIME = dt.time(12, 34, 56, 123456, tzinfo=_TZ_PLUS_0530)
_AWARE_TIMESTAMP = dt.datetime(2024, 2, 29, 12, 34, 56, 123456, tzinfo=_TZ_PLUS_0530)


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


@pytest.mark.parametrize(
    "sql_type",
    [
        "INTEGER", "DECIMAL(18, 6)", "VARCHAR(80)", "VARBINARY(16)",
        "DATE", "TIMESTAMP", "INTERVAL DAY(9) TO SECOND(6)",
        "INTERVAL YEAR(9) TO MONTH",
    ],
)
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


@pytest.mark.parametrize(
    ("value", "expected"),
    [
        (Decimal("12345678901234567890.12345678"), Decimal("12345678901234567890.12345678")),
        ("12345678901234567890.12345678", Decimal("12345678901234567890.12345678")),
        (Decimal("-12345678901234567890.12345678"), Decimal("-12345678901234567890.12345678")),
        ("1234567890123456789012345678E-8", Decimal("12345678901234567890.12345678")),
    ],
    ids=["decimal", "text", "negative", "scientific"],
)
def test_high_precision_decimal_parameter_preserves_fraction(
    conn, table_name_factory, drop_table, value, expected
):
    table = table_name_factory("DMPY_DECIMAL")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v DECIMAL(30, 8))")
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()
        cur.execute(f"SELECT CAST(v AS VARCHAR(80)) FROM {table}")
        assert Decimal(cur.fetchone()[0]) == expected
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()


@pytest.mark.parametrize(
    ("sql_type", "value", "expected", "parse"),
    [
        (
            "TIME(6) WITH TIME ZONE",
            _AWARE_TIME,
            _AWARE_TIME,
            dt.time.fromisoformat,
        ),
        (
            "TIMESTAMP WITH TIME ZONE",
            _AWARE_TIMESTAMP,
            _AWARE_TIMESTAMP,
            dt.datetime.fromisoformat,
        ),
        (
            "TIME(6) WITH TIME ZONE",
            "12:34:56.123456 -04:00",
            dt.time(12, 34, 56, 123456, tzinfo=_TZ_MINUS_0400),
            dt.time.fromisoformat,
        ),
        (
            "TIMESTAMP WITH TIME ZONE",
            "2024-02-29 12:34:56.123456 -04:00",
            dt.datetime(2024, 2, 29, 12, 34, 56, 123456, tzinfo=_TZ_MINUS_0400),
            dt.datetime.fromisoformat,
        ),
    ],
    ids=["aware-time", "aware-timestamp", "text-time", "text-timestamp"],
)
def test_timezone_time_roundtrip_preserves_instant(
    conn, table_name_factory, drop_table, sql_type, value, expected, parse
):
    table = table_name_factory("DMPY_TZ")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v {sql_type})")
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()
        cur.execute(f"SELECT v, CAST(v AS VARCHAR(80)) FROM {table}")
        direct, cast = cur.fetchone()
        for actual in (direct, cast):
            normalized = str(actual).replace(" +", "+").replace(" -", "-")
            parsed = parse(normalized)
            if isinstance(expected, dt.time):
                assert parsed.utcoffset() is not None
                assert (
                    dt.datetime.combine(dt.date(2024, 1, 1), parsed) - parsed.utcoffset()
                ).time() == (
                    dt.datetime.combine(dt.date(2024, 1, 1), expected) - expected.utcoffset()
                ).time()
            else:
                assert parsed == expected
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()


@pytest.mark.parametrize(
    "value",
    [
        dt.timedelta(days=1, hours=2, minutes=3, seconds=4, microseconds=123456),
        -dt.timedelta(seconds=1, microseconds=1),
        -dt.timedelta(days=100000, microseconds=1),
        dt.timedelta(0),
    ],
    ids=["positive-fraction", "negative-fraction", "large-negative", "zero"],
)
def test_day_second_interval_timedelta_roundtrip(conn, table_name_factory, drop_table, value):
    table = table_name_factory("DMPY_INTERVAL_DT")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v INTERVAL DAY(9) TO SECOND(6))")
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()
        cur.execute(f"SELECT v FROM {table}")
        assert cur.description[0][1] is dmPython.INTERVAL
        assert cur.fetchone() == (value,)
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()


def test_year_month_interval_text_roundtrip(conn, table_name_factory, drop_table):
    table = table_name_factory("DMPY_INTERVAL_YM")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v INTERVAL YEAR(9) TO MONTH)")
        cur.execute(f"INSERT INTO {table} VALUES (?)", ("INTERVAL '01-02' YEAR(9) TO MONTH",))
        conn.commit()
        cur.execute(f"SELECT v FROM {table}")
        assert cur.description[0][1] is dmPython.YEAR_MONTH_INTERVAL
        assert re.search(r"'0*1-0*2'", cur.fetchone()[0])
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()
