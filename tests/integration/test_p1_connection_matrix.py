"""Common connection forms and options against a real DM database."""

from __future__ import annotations

import os
import socket
import subprocess
import sys
import threading
import uuid
from contextlib import contextmanager

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


@contextmanager
def ipv6_proxy(conn_params):
    """Forward one IPv6 loopback connection to the IPv4 test database."""
    listener = socket.socket(socket.AF_INET6, socket.SOCK_STREAM)
    listener.settimeout(6)
    listener.bind(("::1", 0))
    listener.listen(1)
    peers = []

    def forward(source, target):
        try:
            while data := source.recv(65536):
                target.sendall(data)
        except OSError:
            pass
        finally:
            try:
                target.shutdown(socket.SHUT_WR)
            except OSError:
                pass

    def serve():
        client, _ = listener.accept()
        upstream = socket.create_connection((conn_params["server"], conn_params["port"]))
        peers.extend((client, upstream))
        directions = [
            threading.Thread(target=forward, args=(client, upstream), daemon=True),
            threading.Thread(target=forward, args=(upstream, client), daemon=True),
        ]
        for direction in directions:
            direction.start()
        for direction in directions:
            direction.join()

    worker = threading.Thread(target=serve, daemon=True)
    worker.start()
    try:
        yield listener.getsockname()[1]
    finally:
        listener.close()
        for peer in peers:
            peer.close()
        worker.join(timeout=1)


@pytest.mark.parametrize("address", ["server", "host", "dsn"])
def test_connection_address_forms(conn_params, address):
    params = {"user": conn_params["user"], "password": conn_params["password"]}
    host = conn_params["server"]
    port = conn_params["port"]
    if address == "dsn":
        params["dsn"] = f"{host}:{port}"
    else:
        params[address] = host
        params["port"] = port
    with dmPython.connect(**params) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)
        assert conn.dsn == f"{host}:{port}"


@pytest.mark.parametrize("address", ["server", "host", "dsn"])
def test_bracketed_ipv6_address_forms(conn_params, address):
    with ipv6_proxy(conn_params) as port:
        params = {"user": conn_params["user"], "password": conn_params["password"]}
        if address == "dsn":
            params["dsn"] = f"[::1]:{port}"
        else:
            params[address] = "[::1]"
            params["port"] = port
        with dmPython.connect(**params, login_timeout=2000) as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT 1")
                assert cur.fetchone() == (1,)


def test_password_with_dsn_delimiters(conn_params):
    admin_password = os.environ.get("DM_CI_ADMIN_PASSWORD")
    container = os.environ.get("DM_BFILE_TEST_CONTAINER")
    if not admin_password or not container:
        pytest.skip("special password regression requires a DM test container and admin password")

    name = f"DMPYSPEC_{uuid.uuid4().hex[:8].upper()}"
    password = "DmPyA1?/#&%+@:"
    admin_params = {**conn_params, "user": "SYSDBA", "password": admin_password}
    with dmPython.connect(**admin_params) as admin:
        with admin.cursor() as cur:
            try:
                # The Python statement scanner sees '?' in quoted DDL as a bind marker.
                # Use the database's SQL client to create this temporary account.
                script = (
                    f"conn SYSDBA/{admin_password}@127.0.0.1:5236\n"
                    f'CREATE USER {name} IDENTIFIED BY "{password}";\n'
                    f"GRANT RESOURCE TO {name};\nexit\n"
                )
                subprocess.run(
                    ["docker", "exec", "-i", container, "/opt/dmdbms/bin/disql", "/nolog"],
                    input=script,
                    text=True,
                    capture_output=True,
                    check=True,
                )
                cur.execute("SELECT COUNT(*) FROM DBA_USERS WHERE USERNAME = ?", (name,))
                assert cur.fetchone() == (1,)
                with dmPython.connect(**{**conn_params, "user": name, "password": password}) as test_conn:
                    with test_conn.cursor() as test_cur:
                        test_cur.execute("SELECT 1")
                        assert test_cur.fetchone() == (1,)
            finally:
                cur.execute("SELECT COUNT(*) FROM DBA_USERS WHERE USERNAME = ?", (name,))
                if cur.fetchone() == (1,):
                    cur.execute(f"DROP USER {name}")
                    admin.commit()


def test_service_name_from_explicit_config_directory(conn_params, tmp_path):
    service = f"DMPYSVC_{uuid.uuid4().hex[:8].upper()}"
    config_dir = tmp_path / "dm config&test"
    config_dir.mkdir()
    (config_dir / "dm_svc.conf").write_text(
        f"{service}={conn_params['server']}:{conn_params['port']}\n",
        encoding="utf-8",
    )
    with dmPython.connect(
        user=conn_params["user"],
        password=conn_params["password"],
        server=service,
        dmsvc_path=str(config_dir),
        login_timeout=2000,
    ) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_service_name_fails_over_after_handshake_failure(conn_params, tmp_path):
    service = f"DMPYFAIL_{uuid.uuid4().hex[:8].upper()}"
    rejected = threading.Event()
    listener = socket.socket()
    listener.settimeout(5)
    listener.bind(("127.0.0.1", 0))
    listener.listen(1)

    def reject_first_endpoint():
        try:
            client, _ = listener.accept()
            rejected.set()
            client.close()
        except OSError:
            pass

    worker = threading.Thread(target=reject_first_endpoint, daemon=True)
    worker.start()
    (tmp_path / "dm_svc.conf").write_text(
        f"{service}=(127.0.0.1:{listener.getsockname()[1]},"
        f"{conn_params['server']}:{conn_params['port']})\n"
        f"[{service}]\nEP_SELECTION=1\nSWITCH_TIMES=1\nSWITCH_INTERVAL=0\n",
        encoding="utf-8",
    )
    try:
        with dmPython.connect(
            user=conn_params["user"],
            password=conn_params["password"],
            server=service,
            dmsvc_path=str(tmp_path),
            login_timeout=3000,
        ) as conn:
            with conn.cursor() as cur:
                cur.execute("SELECT 1")
                assert cur.fetchone() == (1,)
        assert rejected.is_set(), "first endpoint was never attempted"
    finally:
        listener.close()
        worker.join(timeout=1)


@pytest.mark.parametrize("cursorclass", [dmPython.TupleCursor, dmPython.DictCursor])
def test_cursorclass_result_shape(conn_params, cursorclass):
    with dmPython.connect(**conn_params, cursorclass=cursorclass) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1 AS PROBE")
            row = cur.fetchone()
            if cursorclass == dmPython.DictCursor:
                assert row == {"PROBE": 1}
            else:
                assert row == (1,)


def test_schema_option_selects_test_schema(conn_params):
    with dmPython.connect(**conn_params, schema=conn_params["user"]) as conn:
        assert str(conn.current_schema).upper() == str(conn_params["user"]).upper()


@pytest.mark.parametrize("autocommit", [dmPython.DSQL_AUTOCOMMIT_OFF, dmPython.DSQL_AUTOCOMMIT_ON])
def test_autocommit_connect_option(conn_params, autocommit):
    with dmPython.connect(**conn_params, autoCommit=autocommit) as conn:
        assert int(conn.autocommit) == int(autocommit)
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_autocommit_on_persists_without_explicit_commit(
    conn, conn_params, table_name_factory, drop_table
):
    table = table_name_factory("DMPY_AUTOCOMMIT")
    cur = conn.cursor()
    try:
        cur.execute(f"CREATE TABLE {table} (v INTEGER)")
        conn.commit()
        with dmPython.connect(**conn_params, autoCommit=dmPython.DSQL_AUTOCOMMIT_ON) as writer:
            with writer.cursor() as writer_cur:
                writer_cur.execute(f"INSERT INTO {table} VALUES (1)")
        cur.execute(f"SELECT COUNT(*) FROM {table}")
        assert cur.fetchone() == (1,)
    finally:
        drop_table(cur, table)
        conn.commit()
        cur.close()


def test_isolation_connect_option(conn_params):
    with dmPython.connect(**conn_params, txn_isolation=dmPython.ISO_LEVEL_READ_COMMITTED) as conn:
        assert int(conn.txn_isolation) == int(dmPython.ISO_LEVEL_READ_COMMITTED)


def test_host_and_server_cannot_both_be_set(conn_params):
    with pytest.raises(dmPython.NotSupportedError, match="host or server"):
        dmPython.connect(**conn_params, host=conn_params["server"])


def test_port_rejects_non_numeric_value(conn_params):
    with pytest.raises((ValueError, TypeError, dmPython.Error)):
        dmPython.connect(**{**conn_params, "port": object()})


@pytest.mark.parametrize(
    "option",
    [
        {"connection_timeout": 5},
        {"login_timeout": 5000},
        {"compress_msg": 0},
        {"use_stmt_pool": 1},
    ],
)
def test_optional_setting_connects_and_queries(conn_params, option):
    with dmPython.connect(**conn_params, **option) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_connection_timeout_options_are_reported(conn_params):
    with dmPython.connect(
        **conn_params,
        login_timeout=2000,
        connection_timeout=2,
        app_name="dmpython & matrix+1",
    ) as conn:
        assert conn.login_timeout == 2000
        assert conn.connection_timeout == 2
        assert conn.app_name == "dmpython & matrix+1"
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_timeout_defaults_match_dm_interface(conn_params):
    with dmPython.connect(**conn_params) as conn:
        assert conn.login_timeout == 5000
        assert conn.connection_timeout == 0


def test_login_timeout_interrupts_unresponsive_handshake():
    code = """
import dmPython
import sys
import time

start = time.monotonic()
try:
    dmPython.connect(user="probe", password="probe", server="127.0.0.1",
                     port=int(sys.argv[1]), login_timeout=1000)
except dmPython.Error:
    print(time.monotonic() - start)
else:
    raise AssertionError("unresponsive server accepted a connection")
"""
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        listener.listen()
        listener.settimeout(4)
        child = subprocess.Popen(
            [sys.executable, "-c", code, str(listener.getsockname()[1])],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env=os.environ.copy(),
        )
        try:
            peer, _ = listener.accept()
            with peer:
                stdout, stderr = child.communicate(timeout=4)
        finally:
            if child.poll() is None:
                child.kill()
                child.communicate()

    assert child.returncode == 0, stderr
    assert float(stdout.strip()) < 3


@pytest.mark.parametrize("statement", ["direct", "prepared", "select_for_update"])
def test_connection_timeout_limits_sql_execution(
    conn, table_name_factory, drop_table, statement
):
    table = table_name_factory("DMPY_TIMEOUT")
    cur = conn.cursor()
    code = """
import dmPython
import os
import sys
import time

conn = dmPython.connect(
    user=os.environ["DM_TEST_USER"],
    password=os.environ["DM_TEST_PASSWORD"],
    server=os.environ["DM_TEST_HOST"],
    port=int(os.environ["DM_TEST_PORT"]),
    connection_timeout=1,
)
cur = conn.cursor()
start = time.monotonic()
try:
    if sys.argv[2] == "select_for_update":
        cur.execute(f"SELECT v FROM {sys.argv[1]} WHERE id=1 FOR UPDATE")
        cur.fetchone()
    elif sys.argv[2] == "prepared":
        cur.execute(f"UPDATE {sys.argv[1]} SET v=? WHERE id=1", (3,))
    else:
        cur.execute(f"UPDATE {sys.argv[1]} SET v=3 WHERE id=1")
except dmPython.Error:
    print(time.monotonic() - start)
else:
    raise AssertionError("blocked update unexpectedly completed")
cur.execute("SELECT 1")
assert cur.fetchone() == (1,)
"""
    try:
        cur.execute(f"CREATE TABLE {table} (id INTEGER PRIMARY KEY, v INTEGER)")
        cur.execute(f"INSERT INTO {table} VALUES (1, 1)")
        conn.commit()
        cur.execute(f"UPDATE {table} SET v=2 WHERE id=1")

        child = subprocess.Popen(
            [sys.executable, "-c", code, table, statement],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env=os.environ.copy(),
        )
        try:
            stdout, stderr = child.communicate(timeout=6)
        finally:
            if child.poll() is None:
                child.kill()
                child.communicate()
        assert child.returncode == 0, stderr
        assert 0.5 < float(stdout.strip()) < 4
    finally:
        conn.rollback()
        drop_table(cur, table)
        conn.commit()
        cur.close()
