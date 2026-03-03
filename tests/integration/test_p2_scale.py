from __future__ import annotations

import gc
import uuid
from concurrent.futures import ThreadPoolExecutor

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p2_scale]


def test_large_result_fetch_consistency(conn, table_name_factory, drop_table, stress_rows):
    table_name = table_name_factory("DMPY_P2_ROWS")
    cur = conn.cursor()
    rows = max(200, stress_rows)
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, val VARCHAR(64))")
        batch = [(i, f"v{i}") for i in range(1, rows + 1)]
        cur.executemany(f"INSERT INTO {table_name} (id, val) VALUES (?, ?)", batch)
        conn.commit()

        cur.execute(f"SELECT id FROM {table_name} ORDER BY id")
        first = cur.fetchone()
        mid = cur.fetchmany(127)
        tail = cur.fetchall()
        got = [first[0]] + [r[0] for r in mid] + [r[0] for r in tail]
        assert got == list(range(1, rows + 1))
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_multithread_multi_connection_stability(conn_params, table_name_factory, stress_workers):
    table_name = table_name_factory("DMPY_P2_CONCUR")
    setup_conn = dmPython.connect(**conn_params)
    setup_cur = setup_conn.cursor()
    setup_cur.execute(
        f"CREATE TABLE {table_name} (id INT PRIMARY KEY, worker_id INT, val VARCHAR(32))"
    )
    setup_conn.commit()
    setup_cur.close()
    setup_conn.close()

    workers = max(2, stress_workers)
    rows_per_worker = 60

    def worker(worker_id: int) -> int:
        conn = dmPython.connect(**conn_params)
        cur = conn.cursor()
        try:
            for i in range(rows_per_worker):
                pk = worker_id * 100000 + i
                cur.execute(
                    f"INSERT INTO {table_name} (id, worker_id, val) VALUES (?, ?, ?)",
                    (pk, worker_id, f"w{worker_id}_{i}"),
                )
                if i % 20 == 0:
                    cur.execute(f"SELECT COUNT(*) FROM {table_name} WHERE worker_id = ?", (worker_id,))
                    cur.fetchone()
            conn.commit()
            return rows_per_worker
        finally:
            cur.close()
            conn.close()

    with ThreadPoolExecutor(max_workers=workers) as pool:
        inserted = sum(pool.map(worker, range(1, workers + 1)))

    verify_conn = dmPython.connect(**conn_params)
    verify_cur = verify_conn.cursor()
    try:
        verify_cur.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert verify_cur.fetchone()[0] == inserted
    finally:
        try:
            verify_cur.execute(f"DROP TABLE {table_name}")
            verify_conn.commit()
        except Exception:
            pass
        verify_cur.close()
        verify_conn.close()


def test_unicode_and_long_payload_boundary(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P2_UNI")
    cur = conn.cursor()
    try:
        payload = ("中文边界🚀" * 800)[:4000]
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, txt CLOB)")
        cur.execute(f"INSERT INTO {table_name} (id, txt) VALUES (?, ?)", (1, payload))
        conn.commit()
        cur.execute(f"SELECT txt FROM {table_name} WHERE id = 1")
        got = cur.fetchone()[0]
        assert got == payload
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_transaction_visibility_across_connections(conn_params, table_name_factory):
    table_name = table_name_factory("DMPY_P2_TXN")
    conn1 = dmPython.connect(**conn_params)
    conn2 = dmPython.connect(**conn_params)
    cur1 = conn1.cursor()
    cur2 = conn2.cursor()
    try:
        conn1.autocommit = 0
        cur1.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY)")
        conn1.commit()

        cur1.execute(f"INSERT INTO {table_name} (id) VALUES (1)")
        cur2.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert cur2.fetchone()[0] == 0

        conn1.commit()
        cur2.execute(f"SELECT COUNT(*) FROM {table_name}")
        assert cur2.fetchone()[0] == 1
    finally:
        try:
            cur1.execute(f"DROP TABLE {table_name}")
            conn1.commit()
        except Exception:
            pass
        cur2.close()
        cur1.close()
        conn2.close()
        conn1.close()


def test_repeated_connect_cursor_gc_loop(conn_params):
    loops = 120
    for i in range(loops):
        conn = dmPython.connect(**conn_params)
        cur = conn.cursor()
        cur.execute("SELECT 1")
        assert cur.fetchone() == (1,)
        cur.close()
        conn.close()
        if i % 15 == 0:
            gc.collect()
