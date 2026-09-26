"""Verify read/write routing against a real DM primary and standby.

Set DM_HA_PRIMARY_HOST, DM_HA_STANDBY_HOST, DM_HA_USER, and DM_HA_PASSWORD.
The account must be able to query V$INSTANCE and create/drop a test table.
"""

from __future__ import annotations

import os
import time
import uuid
from decimal import Decimal

import dmPython


def connect(host: str, port: int, **options):
    return dmPython.connect(
        user=os.environ["DM_HA_USER"],
        password=os.environ["DM_HA_PASSWORD"],
        server=host,
        port=port,
        autoCommit=dmPython.DSQL_AUTOCOMMIT_ON,
        login_timeout=2000,
        **options,
    )


def instance(conn):
    with conn.cursor() as cur:
        cur.execute("SELECT INSTANCE_NAME, MODE$, STATUS$ FROM V$INSTANCE")
        return cur.fetchone()


def main():
    primary_host = os.environ["DM_HA_PRIMARY_HOST"]
    standby_host = os.environ["DM_HA_STANDBY_HOST"]
    primary_port = int(os.environ.get("DM_HA_PRIMARY_PORT", "5236"))
    standby_port = int(os.environ.get("DM_HA_STANDBY_PORT", "5236"))
    amount = Decimal("12345678901234567890.12345678")
    table = "DMPY_HA_" + uuid.uuid4().hex[:8].upper()

    with connect(primary_host, primary_port) as primary, connect(
        standby_host, standby_port
    ) as standby:
        primary_name, primary_mode, primary_status = instance(primary)
        standby_name, standby_mode, standby_status = instance(standby)
        assert (primary_mode, primary_status) == ("PRIMARY", "OPEN")
        assert (standby_mode, standby_status) == ("STANDBY", "OPEN")

        for mode, expected in (
            (dmPython.DSQL_RWSEPARATE_OFF, primary_name),
            (dmPython.DSQL_RWSEPARATE_ON, standby_name),
            (dmPython.DSQL_RWSEPARATE_ON2, standby_name),
        ):
            with connect(
                primary_host, primary_port, rwseparate=mode, rwseparate_percent=0
            ) as routed:
                actual = instance(routed)[0]
                assert actual == expected, (mode, actual, expected)
                print(f"rwseparate={mode}: {actual}")

        with primary.cursor() as cur:
            cur.execute(f"CREATE TABLE {table} (ID INT, AMOUNT DECIMAL(30,8))")
        try:
            with connect(
                primary_host,
                primary_port,
                rwseparate=dmPython.DSQL_RWSEPARATE_ON2,
                rwseparate_percent=0,
            ) as routed:
                with routed.cursor() as cur:
                    cur.execute(f"INSERT INTO {table} VALUES (?, ?)", (1, amount))

            deadline = time.monotonic() + 10
            while True:
                try:
                    with standby.cursor() as cur:
                        cur.execute(f"SELECT CAST(AMOUNT AS VARCHAR(80)) FROM {table} WHERE ID=1")
                        row = cur.fetchone()
                    if row is not None:
                        assert Decimal(row[0]) == amount
                        print("standby replicated DECIMAL(30,8) exactly")
                        break
                except dmPython.Error:
                    # The table may not be visible until the DDL reaches the standby.
                    pass
                if time.monotonic() >= deadline:
                    raise AssertionError("write did not reach standby within 10 seconds")
                time.sleep(0.2)
        finally:
            with primary.cursor() as cur:
                cur.execute(f"DROP TABLE {table}")


if __name__ == "__main__":
    main()
