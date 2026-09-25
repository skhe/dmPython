"""Create an isolated test account in a fresh DM8 CI container."""

import os
import json
import platform
import time
from pathlib import Path

import dmPython


host = os.environ.get("DM_TEST_HOST", "127.0.0.1")
port = int(os.environ.get("DM_TEST_PORT", "15236"))
admin_password = os.environ["DM_CI_ADMIN_PASSWORD"]
test_password = os.environ["DM_TEST_PASSWORD"]

if not (test_password.isalnum() and len(test_password) >= 12):
    raise ValueError("DM_TEST_PASSWORD must be alphanumeric and at least 12 characters")

deadline = time.monotonic() + 240
while True:
    try:
        conn = dmPython.connect(
            user="SYSDBA", password=admin_password, server=host, port=port
        )
        break
    except dmPython.Error:
        if time.monotonic() >= deadline:
            raise RuntimeError("DM8 did not become ready within 240 seconds") from None
        time.sleep(3)

try:
    cur = conn.cursor()
    cur.execute(f'CREATE USER DMPYTEST IDENTIFIED BY "{test_password}"')
    cur.execute("GRANT RESOURCE TO DMPYTEST")
    conn.commit()
finally:
    conn.close()

test_conn = dmPython.connect(
    user="DMPYTEST", password=test_password, server=host, port=port
)
try:
    cur = test_conn.cursor()
    cur.execute("SELECT 1")
    assert cur.fetchone() == (1,)
    cur.execute("SELECT ID_CODE FROM V$INSTANCE")
    server_id_code = cur.fetchone()[0]
finally:
    test_conn.close()
Path("dm-ci-environment.json").write_text(
    json.dumps(
        {
            "python": platform.python_version(),
            "machine": platform.machine(),
            "driver": dmPython.version,
            "server_id_code": server_id_code,
        },
        indent=2,
    ) + "\n",
    encoding="utf-8",
)
print("DM8 test account ready")
