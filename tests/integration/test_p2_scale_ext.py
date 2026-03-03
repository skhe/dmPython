from __future__ import annotations

import gc
from concurrent.futures import ThreadPoolExecutor

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p2_scale]


def test_fetchmany_arraysize_matrix_consistency(conn, table_name_factory, drop_table, stress_rows):
    table_name = table_name_factory("DMPY_P2_ARR")
    cur = conn.cursor()
    rows = max(512, stress_rows // 2)
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, val VARCHAR(64))")
        cur.executemany(
            f"INSERT INTO {table_name} (id, val) VALUES (?, ?)",
            [(i, f"v{i}") for i in range(1, rows + 1)],
        )
        conn.commit()

        cur.execute(f"SELECT id FROM {table_name} ORDER BY id")
        baseline = [row[0] for row in cur.fetchall()]
        assert baseline == list(range(1, rows + 1))

        for arr in (1, 16, 128, 512):
            c = conn.cursor()
            try:
                c.arraysize = arr
                c.execute(f"SELECT id FROM {table_name} ORDER BY id")
                first = c.fetchone()
                assert first is not None
                second_batch = c.fetchmany(max(1, arr // 2))
                remain = c.fetchall()
                got = [first[0]] + [r[0] for r in second_batch] + [r[0] for r in remain]
                assert got == baseline
            finally:
                c.close()
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()


def test_lob_mixed_read_write_multithread_multi_connection(
    conn_params, table_name_factory, stress_workers
):
    table_name = table_name_factory("DMPY_P2_LOBMT")
    setup_conn = dmPython.connect(**conn_params)
    setup_cur = setup_conn.cursor()
    setup_cur.execute(
        f"CREATE TABLE {table_name} (id INT PRIMARY KEY, worker_id INT, c CLOB, b BLOB)"
    )
    setup_conn.commit()
    setup_cur.close()
    setup_conn.close()

    workers = max(2, stress_workers)
    rows_per_worker = 15

    def worker(worker_id: int) -> int:
        conn = dmPython.connect(**conn_params)
        cur = conn.cursor()
        try:
            for i in range(rows_per_worker):
                pk = worker_id * 100000 + i
                c_payload = (f"w{worker_id}-" + ("中🚀" * 500))[:1800]
                b_payload = bytes([worker_id % 256]) * 2048
                cur.execute(
                    f"INSERT INTO {table_name} (id, worker_id, c, b) VALUES (?, ?, ?, ?)",
                    (pk, worker_id, c_payload, b_payload),
                )
                cur.execute(f"SELECT c, b FROM {table_name} WHERE id = ?", (pk,))
                row = cur.fetchone()
                assert row is not None
                assert isinstance(row[0], str)
                assert isinstance(row[1], (bytes, bytearray))
                assert row[0] == c_payload
                assert bytes(row[1]) == b_payload
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
        assert int(verify_cur.fetchone()[0]) == inserted
    finally:
        try:
            verify_cur.execute(f"DROP TABLE {table_name}")
            verify_conn.commit()
        except Exception:
            pass
        verify_cur.close()
        verify_conn.close()


def test_connection_churn_with_lob_roundtrip(conn_params, table_name_factory):
    table_name = table_name_factory("DMPY_P2_CHURN")
    setup_conn = dmPython.connect(**conn_params)
    setup_cur = setup_conn.cursor()
    setup_cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB, b BLOB)")
    setup_conn.commit()
    setup_cur.close()
    setup_conn.close()

    loops = 90
    try:
        for i in range(1, loops + 1):
            conn = dmPython.connect(**conn_params)
            cur = conn.cursor()
            try:
                c_payload = (f"loop-{i}-" + ("文" * 200))[:1200]
                b_payload = bytes([i % 256]) * 256
                cur.execute(
                    f"INSERT INTO {table_name} (id, c, b) VALUES (?, ?, ?)",
                    (i, c_payload, b_payload),
                )
                conn.commit()
                cur.execute(f"SELECT c, b FROM {table_name} WHERE id = ?", (i,))
                row = cur.fetchone()
                assert row is not None
                assert row[0] == c_payload
                assert bytes(row[1]) == b_payload
                cur.execute(f"DELETE FROM {table_name} WHERE id = ?", (i,))
                conn.commit()
            finally:
                cur.close()
                conn.close()
    finally:
        cleanup_conn = dmPython.connect(**conn_params)
        cleanup_cur = cleanup_conn.cursor()
        try:
            cleanup_cur.execute(f"DROP TABLE {table_name}")
            cleanup_conn.commit()
        except Exception:
            pass
        cleanup_cur.close()
        cleanup_conn.close()


def test_long_gc_loop_with_lob_objects(conn, table_name_factory, drop_table):
    table_name = table_name_factory("DMPY_P2_GCLOB")
    cur = conn.cursor()
    loops = 120
    try:
        cur.execute(f"CREATE TABLE {table_name} (id INT PRIMARY KEY, c CLOB, b BLOB)")
        conn.commit()
        for i in range(1, loops + 1):
            c_payload = ("gc-loop-" + ("中" * 300))[:900]
            b_payload = bytes([i % 256]) * 128
            cur.execute(
                f"INSERT INTO {table_name} (id, c, b) VALUES (?, ?, ?)",
                (i, c_payload, b_payload),
            )
            conn.commit()

            cur.execute(f"SELECT c, b FROM {table_name} WHERE id = ?", (i,))
            row = cur.fetchone()
            assert row is not None
            assert row[0] == c_payload
            assert bytes(row[1]) == b_payload

            cur.execute(f"DELETE FROM {table_name} WHERE id = ?", (i,))
            conn.commit()
            if i % 10 == 0:
                gc.collect()
    finally:
        drop_table(cur, table_name)
        conn.commit()
        cur.close()
