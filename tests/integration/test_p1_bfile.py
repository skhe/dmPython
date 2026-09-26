"""BFILE locator reads against a real DM database and container file."""

from __future__ import annotations

import os
import subprocess
import uuid

import pytest

import dmPython


pytestmark = [pytest.mark.requires_dm, pytest.mark.p1_contract]


@pytest.fixture(scope="module")
def bfile_resource():
    container = os.environ.get("DM_BFILE_TEST_CONTAINER")
    admin_password = os.environ.get("DM_CI_ADMIN_PASSWORD")
    if not container or not admin_password:
        pytest.skip("BFILE regression requires a DM container and admin test password")

    suffix = uuid.uuid4().hex[:8].upper()
    directory = f"DMPYBF_{suffix}"
    filename = f"dmpython-bfile-{suffix.lower()}.bin"
    path = f"/tmp/{filename}"
    content = bytes(range(256)) * 157
    subprocess.run(
        ["docker", "exec", "-i", container, "sh", "-c", f"cat > {path}"],
        input=content,
        check=True,
    )

    admin = None
    created = False
    try:
        admin = dmPython.connect(
            user="SYSDBA",
            password=admin_password,
            server=os.environ["DM_TEST_HOST"],
            port=int(os.environ["DM_TEST_PORT"]),
        )
        with admin.cursor() as cur:
            cur.execute(f"CREATE DIRECTORY {directory} AS '/tmp'")
            created = True
            cur.execute(f"GRANT READ ON DIRECTORY {directory} TO {os.environ['DM_TEST_USER']}")
        admin.commit()
        yield directory, filename, content
    finally:
        if admin is not None:
            if created:
                with admin.cursor() as cur:
                    cur.execute(f"DROP DIRECTORY {directory}")
                admin.commit()
            admin.close()
        subprocess.run(["docker", "exec", container, "rm", "-f", path], check=True)


def test_bfile_size_and_binary_reads(conn, bfile_resource):
    directory, filename, content = bfile_resource
    with conn.cursor() as cur:
        cur.execute(f"SELECT BFILENAME('{directory}','{filename}')")
        value = cur.fetchone()[0]

    assert value.size() == len(content)
    assert value.read() == content
    assert value.read(offset=15999, amount=20) == content[15998:16018]
    assert value.read(offset=len(content) + 1, amount=10) == b""


def test_bfile_locator_insert_and_fetch(conn, bfile_resource, table_name_factory, drop_table):
    directory, filename, content = bfile_resource
    table = table_name_factory("DMPY_BFILE")
    cur = conn.cursor()
    created = False
    try:
        cur.execute(f"CREATE TABLE {table} (V BFILE)")
        created = True
        conn.commit()
        cur.execute(f"SELECT BFILENAME('{directory}','{filename}')")
        value = cur.fetchone()[0]
        cur.execute(f"INSERT INTO {table} VALUES (?)", (value,))
        conn.commit()

        cur.execute(f"SELECT V FROM {table}")
        fetched = cur.fetchone()[0]
        assert fetched.size() == len(content)
        assert fetched.read() == content
    finally:
        conn.rollback()
        if created:
            drop_table(cur, table)
            conn.commit()
        cur.close()


def test_bfile_read_after_connection_close_reports_error(bfile_resource):
    directory, filename, _ = bfile_resource
    separate = dmPython.connect(
        user=os.environ["DM_TEST_USER"],
        password=os.environ["DM_TEST_PASSWORD"],
        server=os.environ["DM_TEST_HOST"],
        port=int(os.environ["DM_TEST_PORT"]),
    )
    with separate.cursor() as cur:
        cur.execute(f"SELECT BFILENAME('{directory}','{filename}')")
        value = cur.fetchone()[0]
    separate.close()

    with pytest.raises(ValueError, match="cursor or connection is closed"):
        value.read()
