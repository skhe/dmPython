"""Verify local and global login against a real two-node DM MPP cluster.

Set DM_MPP_USER, DM_MPP_PASSWORD, DM_MPP_EP1_HOST, and DM_MPP_EP2_HOST.
DM_MPP_EP1_PORT and DM_MPP_EP2_PORT default to 5236. The account must be
able to create and drop a distributed table.
"""

from __future__ import annotations

import os
import uuid
from decimal import Decimal

import dmPython


HIGH = Decimal("12345678901234567890.12345678")


def connect(host: str, port: int, mode: int):
    return dmPython.connect(
        user=os.environ["DM_MPP_USER"],
        password=os.environ["DM_MPP_PASSWORD"],
        server=host,
        port=port,
        mpp_login=mode,
        autoCommit=dmPython.DSQL_AUTOCOMMIT_ON,
        login_timeout=3000,
    )


def rows(conn, table: str):
    with conn.cursor() as cur:
        cur.execute(f"SELECT ID, CAST(AMOUNT AS VARCHAR(80)) FROM {table} ORDER BY ID")
        return {row_id: Decimal(amount) for row_id, amount in cur.fetchall()}


def local_instance(conn):
    with conn.cursor() as cur:
        cur.execute("SELECT INSTANCE_NAME FROM V$INSTANCE")
        return cur.fetchone()[0]


def main():
    endpoints = (
        (os.environ["DM_MPP_EP1_HOST"], int(os.environ.get("DM_MPP_EP1_PORT", "5236"))),
        (os.environ["DM_MPP_EP2_HOST"], int(os.environ.get("DM_MPP_EP2_PORT", "5236"))),
    )
    table = "DMPY_MPP_" + uuid.uuid4().hex[:8].upper()
    expected = {i: HIGH if i % 2 else -HIGH for i in range(1, 13)}
    expected[12] = Decimal("0.00000001")

    with connect(*endpoints[0], dmPython.DSQL_MPP_LOGIN_GLOBAL) as global_conn:
        with global_conn.cursor() as cur:
            cur.execute(
                f"CREATE TABLE {table} (ID INT PRIMARY KEY, AMOUNT DECIMAL(30,8)) "
                "DISTRIBUTED BY HASH(ID)"
            )
        try:
            with global_conn.cursor() as cur:
                cur.executemany(
                    f"INSERT INTO {table} VALUES (?, ?)", list(expected.items())
                )

            local_rows = []
            instance_names = []
            for host, port in endpoints:
                with connect(host, port, dmPython.DSQL_MPP_LOGIN_GLOBAL) as conn:
                    assert rows(conn, table) == expected
                with connect(host, port, dmPython.DSQL_MPP_LOGIN_LOCAL) as conn:
                    instance_names.append(local_instance(conn))
                    local_rows.append(rows(conn, table))

            assert instance_names[0] != instance_names[1], instance_names
            assert local_rows[0] and local_rows[1], "both EPs must store rows"
            assert local_rows[0].keys().isdisjoint(local_rows[1].keys())
            assert local_rows[0] | local_rows[1] == expected
            print(
                f"MPP global: {len(expected)} exact rows from both endpoints; "
                f"local: {instance_names[0]}={len(local_rows[0])}, "
                f"{instance_names[1]}={len(local_rows[1])}"
            )
        finally:
            with global_conn.cursor() as cur:
                cur.execute(f"DROP TABLE {table}")


if __name__ == "__main__":
    main()
