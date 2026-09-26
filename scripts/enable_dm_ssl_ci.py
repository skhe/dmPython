"""Enable mandatory SSL on the isolated DM8 CI instance before its restart."""

import os
import time

import dmPython


deadline = time.monotonic() + 240
while True:
    try:
        conn = dmPython.connect(
            user="SYSDBA",
            password=os.environ["DM_CI_ADMIN_PASSWORD"],
            server=os.environ["DM_SSL_TEST_HOST"],
            port=int(os.environ["DM_SSL_TEST_PORT"]),
        )
        break
    except dmPython.Error:
        if time.monotonic() >= deadline:
            raise RuntimeError("DM8 did not become ready within 240 seconds") from None
        time.sleep(3)

try:
    with conn.cursor() as cur:
        cur.execute("SP_SET_PARA_VALUE(2, 'ENABLE_ENCRYPT', 1)")
    conn.commit()
finally:
    conn.close()
