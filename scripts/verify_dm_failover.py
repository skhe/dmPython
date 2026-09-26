"""Verify a DM primary/standby takeover with two separate invocations.

Run ``prepare`` while both nodes are healthy, stop the primary database and
watcher, then run ``verify`` after the standby has been promoted. The service
name in dm_svc.conf must list both endpoints with LOGIN_MODE=1 (primary only).
"""

from __future__ import annotations

import argparse
import os
import re
import time
from decimal import Decimal

import dmPython


BEFORE_AMOUNT = Decimal("12345678901234567890.12345678")
AFTER_AMOUNT = Decimal("98765432109876543210.87654321")


def connect(server: str, *, port: int | None = None, service: bool = False):
    options = {
        "user": os.environ["DM_HA_USER"],
        "password": os.environ["DM_HA_PASSWORD"],
        "server": server,
        "autoCommit": dmPython.DSQL_AUTOCOMMIT_ON,
        "login_timeout": 3000,
    }
    if service:
        options["dmsvc_path"] = os.environ["DM_HA_SERVICE_PATH"]
    else:
        options["port"] = port
    return dmPython.connect(**options)


def instance(conn):
    with conn.cursor() as cur:
        cur.execute("SELECT INSTANCE_NAME, MODE$, STATUS$ FROM V$INSTANCE")
        return cur.fetchone()


def amount_at(conn, table: str, row_id: int):
    with conn.cursor() as cur:
        cur.execute(f"SELECT CAST(AMOUNT AS VARCHAR(80)) FROM {table} WHERE ID=?", (row_id,))
        row = cur.fetchone()
    return None if row is None else Decimal(row[0])


def prepare(table: str):
    with connect(os.environ["DM_HA_SERVICE_NAME"], service=True) as primary, connect(
        os.environ["DM_HA_STANDBY_HOST"],
        port=int(os.environ.get("DM_HA_STANDBY_PORT", "5236")),
    ) as standby:
        primary_name, primary_mode, primary_status = instance(primary)
        standby_name, standby_mode, standby_status = instance(standby)
        assert (primary_mode, primary_status) == ("PRIMARY", "OPEN")
        assert (standby_mode, standby_status) == ("STANDBY", "OPEN")

        with primary.cursor() as cur:
            cur.execute(f"CREATE TABLE {table} (ID INT PRIMARY KEY, AMOUNT DECIMAL(30,8))")
            cur.execute(f"INSERT INTO {table} VALUES (?, ?)", (1, BEFORE_AMOUNT))

        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            try:
                if amount_at(standby, table, 1) == BEFORE_AMOUNT:
                    print(f"prepared: {primary_name} PRIMARY, {standby_name} STANDBY")
                    print(f"table: {table}; standby has exact pre-failover decimal")
                    return
            except dmPython.Error:
                # The replicated table definition may not be visible yet.
                pass
            time.sleep(0.2)
        raise AssertionError("standby did not receive exact value within 15 seconds")


def verify(table: str, expected_new_primary: str):
    with connect(os.environ["DM_HA_SERVICE_NAME"], service=True) as primary:
        actual = instance(primary)
        assert actual == (expected_new_primary, "PRIMARY", "OPEN"), actual
        assert amount_at(primary, table, 1) == BEFORE_AMOUNT

        if amount_at(primary, table, 2) is None:
            with primary.cursor() as cur:
                cur.execute(f"INSERT INTO {table} VALUES (?, ?)", (2, AFTER_AMOUNT))
        assert amount_at(primary, table, 2) == AFTER_AMOUNT

        with primary.cursor() as cur:
            cur.execute(f"DROP TABLE {table}")
        print(f"verified: connected to promoted primary {expected_new_primary}")
        print("pre- and post-failover DECIMAL(30,8) values were exact")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("phase", choices=("prepare", "verify"))
    parser.add_argument("--table", required=True, help="unique test table name")
    parser.add_argument("--expected-new-primary", help="standby instance name from prepare")
    args = parser.parse_args()

    table = args.table.upper()
    if not re.fullmatch(r"[A-Z][A-Z0-9_]{0,63}", table):
        parser.error("--table must be a simple unquoted SQL identifier")
    if args.phase == "verify" and not args.expected_new_primary:
        parser.error("--expected-new-primary is required for verify")

    if args.phase == "prepare":
        prepare(table)
    else:
        verify(table, args.expected_new_primary)


if __name__ == "__main__":
    main()
