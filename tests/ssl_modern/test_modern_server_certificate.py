"""Real DM8 regression for CA and SAN verification after cert replacement."""

import os
import shutil
import time
from pathlib import Path

import dmPython
import pytest


@pytest.fixture(scope="module")
def modern_params():
    names = (
        "DM_SSL_MODERN_HOST",
        "DM_SSL_MODERN_PORT",
        "DM_SSL_MODERN_USER",
        "DM_SSL_MODERN_PASSWORD",
        "DM_SSL_MODERN_PATH",
        "DM_SSL_MODERN_BAD_CA",
    )
    missing = [name for name in names if not os.getenv(name)]
    if missing:
        pytest.fail(f"modern SSL environment is incomplete: {', '.join(missing)}")
    params = {
        "server": os.environ["DM_SSL_MODERN_HOST"],
        "port": int(os.environ["DM_SSL_MODERN_PORT"]),
        "user": os.environ["DM_SSL_MODERN_USER"],
        "password": os.environ["DM_SSL_MODERN_PASSWORD"],
        "ssl_path": os.environ["DM_SSL_MODERN_PATH"],
    }
    assert not (Path(params["ssl_path"]) / "server-cert.pem").exists()
    deadline = time.monotonic() + 90
    while True:
        try:
            with dmPython.connect(**params):
                return params
        except dmPython.Error:
            if time.monotonic() >= deadline:
                pytest.fail("DM8 did not accept the modern certificate within 90 seconds")
            time.sleep(2)


def test_trusted_ip_san_connects(modern_params):
    with dmPython.connect(**modern_params) as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_wrong_hostname_is_rejected(modern_params):
    with pytest.raises(dmPython.Error, match=r"x509:.*localhost"):
        dmPython.connect(**{**modern_params, "server": "localhost"})


def test_untrusted_ca_is_rejected(modern_params, tmp_path):
    cert_dir = tmp_path / "untrusted-ca"
    shutil.copytree(modern_params["ssl_path"], cert_dir)
    shutil.copyfile(os.environ["DM_SSL_MODERN_BAD_CA"], cert_dir / "ca-cert.pem")
    with pytest.raises(dmPython.Error, match="unknown authority"):
        dmPython.connect(**{**modern_params, "ssl_path": str(cert_dir)})
