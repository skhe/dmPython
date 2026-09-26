"""Check that database restart never reports a lost transaction as committed.

Run only against a disposable DM8 container. The script restarts that container.
"""

from __future__ import annotations

import os
import subprocess
import time
import uuid
from decimal import Decimal

import dmPython


AMOUNT = Decimal("12345678901234567890.12345678")


def connect(*, autocommit: bool):
    return dmPython.connect(
        user=os.environ["DM_RESTART_USER"],
        password=os.environ["DM_RESTART_PASSWORD"],
        server=os.environ.get("DM_RESTART_HOST", "127.0.0.1"),
        port=int(os.environ["DM_RESTART_PORT"]),
        autoCommit=(
            dmPython.DSQL_AUTOCOMMIT_ON
            if autocommit
            else dmPython.DSQL_AUTOCOMMIT_OFF
        ),
        login_timeout=3000,
        connection_timeout=3,
    )


def wait_for_database():
    deadline = time.monotonic() + 120
    while True:
        try:
            return connect(autocommit=True)
        except dmPython.Error:
            if time.monotonic() >= deadline:
                raise RuntimeError("DM8 did not become ready within 120 seconds") from None
            time.sleep(2)


def count_rows(conn, table: str) -> int:
    with conn.cursor() as cur:
        cur.execute(f"SELECT COUNT(*) FROM {table}")
        return cur.fetchone()[0]


def main():
    container = os.environ["DM_RESTART_CONTAINER"]
    table = "DMPY_RESTART_" + uuid.uuid4().hex[:8].upper()
    admin = wait_for_database()
    try:
        with admin.cursor() as cur:
            cur.execute(f"CREATE TABLE {table} (ID INT PRIMARY KEY, V DECIMAL(30,8))")
        tx = connect(autocommit=False)
        try:
            with tx.cursor() as cur:
                cur.execute(f"INSERT INTO {table} VALUES (?, ?)", (1, AMOUNT))
            assert count_rows(admin, table) == 0, "uncommitted row was visible"

            subprocess.run(
                ["docker", "restart", container],
                check=True,
                stdout=subprocess.DEVNULL,
            )
            with wait_for_database() as reader:
                assert count_rows(reader, table) == 0, "uncommitted row survived restart"
                try:
                    tx.commit()
                except dmPython.Error:
                    pass
                else:
                    raise AssertionError("commit reported success for a lost transaction")
                assert count_rows(reader, table) == 0

            deadline = time.monotonic() + 30
            errors = 0
            while True:
                try:
                    assert count_rows(admin, table) == 0
                    break
                except dmPython.Error:
                    errors += 1
                    if time.monotonic() >= deadline:
                        raise AssertionError("existing autocommit connection did not recover") from None
                    time.sleep(2)
            print(f"lost transaction rejected; existing autocommit connection recovered after {errors} error(s)")
        finally:
            tx.close()
    finally:
        with wait_for_database() as cleanup:
            with cleanup.cursor() as cur:
                cur.execute(f"DROP TABLE {table}")
        admin.close()


if __name__ == "__main__":
    main()
