"""Verify already-open Python connections across a real DM8 HA takeover.

Run only against a disposable two-node watcher group with a confirmation
monitor. This script kills the primary's dmwatcher and dmserver processes.
The service name must list both endpoints with LOGIN_MODE=1.
"""

from __future__ import annotations

import os
import subprocess
import time
import uuid
from decimal import Decimal

import dmPython


BEFORE = Decimal("12345678901234567890.12345678")
AFTER = Decimal("98765432109876543210.87654321")


def connect(*, autocommit: bool, standby: bool = False):
    options = {
        "user": os.environ["DM_HA_USER"],
        "password": os.environ["DM_HA_PASSWORD"],
        "server": (
            os.environ["DM_HA_STANDBY_HOST"]
            if standby
            else os.environ["DM_HA_SERVICE_NAME"]
        ),
        "autoCommit": (
            dmPython.DSQL_AUTOCOMMIT_ON if autocommit else dmPython.DSQL_AUTOCOMMIT_OFF
        ),
        "login_timeout": 3000,
        "connection_timeout": 3,
    }
    if standby:
        options["port"] = int(os.environ.get("DM_HA_STANDBY_PORT", "5236"))
    else:
        options["dmsvc_path"] = os.environ["DM_HA_SERVICE_PATH"]
    return dmPython.connect(**options)


def instance(conn):
    with conn.cursor() as cur:
        cur.execute("SELECT INSTANCE_NAME, MODE$, STATUS$ FROM V$INSTANCE")
        return tuple(cur.fetchone())


def amount_at(conn, table: str, row_id: int):
    with conn.cursor() as cur:
        cur.execute(
            f"SELECT CAST(AMOUNT AS VARCHAR(80)) FROM {table} WHERE ID=?", (row_id,)
        )
        row = cur.fetchone()
    return None if row is None else Decimal(row[0])


def wait_until(check, description: str, seconds: int = 90):
    deadline = time.monotonic() + seconds
    last_error = None
    while time.monotonic() < deadline:
        try:
            result = check()
            if result:
                return result
        except dmPython.Error as exc:
            last_error = exc
        time.sleep(1)
    raise AssertionError(f"timed out waiting for {description}: {last_error}")


def stop_primary(container: str):
    subprocess.run(
        [
            "docker",
            "exec",
            container,
            "bash",
            "-c",
            "pkill -9 -x dmwatcher && pkill -9 -x dmserver",
        ],
        check=True,
        stdout=subprocess.DEVNULL,
    )


def main():
    container = os.environ["DM_HA_PRIMARY_CONTAINER"]
    table = "DMPY_HA_OPEN_" + uuid.uuid4().hex[:8].upper()
    with (
        connect(autocommit=True) as held,
        connect(autocommit=True, standby=True) as standby,
    ):
        old_name, old_mode, old_status = instance(held)
        new_name, new_mode, new_status = instance(standby)
        assert (old_mode, old_status) == ("PRIMARY", "OPEN")
        assert (new_mode, new_status) == ("STANDBY", "OPEN")
        assert old_name != new_name

        with held.cursor() as cur:
            cur.execute(
                f"CREATE TABLE {table} (ID INT PRIMARY KEY, AMOUNT DECIMAL(30,8))"
            )
            cur.execute(f"INSERT INTO {table} VALUES (?, ?)", (1, BEFORE))

        try:
            wait_until(
                lambda: amount_at(standby, table, 1) == BEFORE,
                "exact decimal on standby",
                seconds=20,
            )
            tx = connect(autocommit=False)
            try:
                assert instance(tx)[0] == old_name
                with tx.cursor() as cur:
                    cur.execute(f"INSERT INTO {table} VALUES (?, ?)", (2, AFTER))
                assert amount_at(standby, table, 2) is None

                stop_primary(container)
                wait_until(
                    lambda: instance(standby) == (new_name, "PRIMARY", "OPEN"),
                    "standby promotion",
                )
                assert amount_at(standby, table, 1) == BEFORE
                assert amount_at(standby, table, 2) is None

                try:
                    tx.commit()
                except dmPython.Error:
                    pass
                else:
                    raise AssertionError(
                        "lost transaction reported a successful commit"
                    )
                assert amount_at(standby, table, 2) is None

                errors = 0

                def held_connection_recovered():
                    nonlocal errors
                    try:
                        return instance(held) == (new_name, "PRIMARY", "OPEN")
                    except dmPython.Error:
                        errors += 1
                        return False

                wait_until(
                    held_connection_recovered, "held autocommit connection recovery"
                )
                with held.cursor() as cur:
                    cur.execute(f"INSERT INTO {table} VALUES (?, ?)", (3, AFTER))
                assert amount_at(held, table, 3) == AFTER
                assert amount_at(standby, table, 3) == AFTER
                print(
                    f"promoted {new_name}; held connection recovered after {errors} error(s)"
                )
                print("lost transaction rejected; DECIMAL(30,8) remained exact")
            finally:
                tx.close()
        finally:
            # The disposable HA database is retained for inspection if the
            # takeover fails, but remove the test table when a primary exists.
            try:
                with connect(autocommit=True, standby=True) as cleanup:
                    if instance(cleanup)[1] == "PRIMARY":
                        with cleanup.cursor() as cur:
                            cur.execute(f"DROP TABLE {table}")
            except dmPython.Error:
                pass


if __name__ == "__main__":
    main()
