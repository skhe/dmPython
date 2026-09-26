"""Real SSL handshake checks against a dedicated encrypted DM8 instance."""

import os
import shutil
import time

import dmPython
import pytest


@pytest.fixture(scope="module")
def ssl_params():
    names = (
        "DM_SSL_TEST_HOST",
        "DM_SSL_TEST_PORT",
        "DM_SSL_TEST_USER",
        "DM_SSL_TEST_PASSWORD",
        "DM_SSL_TEST_PATH",
    )
    missing = [name for name in names if not os.getenv(name)]
    if missing:
        pytest.fail(f"SSL database environment is incomplete: {', '.join(missing)}")
    params = {
        "server": os.environ["DM_SSL_TEST_HOST"],
        "port": int(os.environ["DM_SSL_TEST_PORT"]),
        "user": os.environ["DM_SSL_TEST_USER"],
        "password": os.environ["DM_SSL_TEST_PASSWORD"],
        "ssl_path": os.environ["DM_SSL_TEST_PATH"],
    }
    deadline = time.monotonic() + 180
    while True:
        try:
            conn = dmPython.connect(**params)
            conn.close()
            return params
        except dmPython.Error:
            if time.monotonic() >= deadline:
                pytest.fail("SSL database did not become ready within 180 seconds")
            time.sleep(3)


def test_encrypted_connection_and_query(ssl_params):
    with dmPython.connect(**ssl_params) as conn:
        assert conn.ssl_path == ssl_params["ssl_path"]
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_ssl_path_with_spaces_and_ampersand(ssl_params, tmp_path):
    cert_dir = tmp_path / "ssl files & certs"
    shutil.copytree(ssl_params["ssl_path"], cert_dir)
    with dmPython.connect(**{**ssl_params, "ssl_path": str(cert_dir)}) as conn:
        assert conn.ssl_path == str(cert_dir)
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            assert cur.fetchone() == (1,)


def test_wrong_server_certificate_pin_is_rejected(ssl_params, tmp_path):
    cert_dir = tmp_path / "wrong-pin"
    shutil.copytree(ssl_params["ssl_path"], cert_dir)
    shutil.copyfile(cert_dir / "client-cert.pem", cert_dir / "server-cert.pem")
    with pytest.raises(dmPython.Error, match="does not match server-cert.pem pin"):
        dmPython.connect(**{**ssl_params, "ssl_path": str(cert_dir)})


def test_missing_server_certificate_pin_is_rejected(ssl_params, tmp_path):
    cert_dir = tmp_path / "missing-pin"
    shutil.copytree(ssl_params["ssl_path"], cert_dir)
    (cert_dir / "server-cert.pem").unlink()
    with pytest.raises(dmPython.Error, match="requires server-cert.pem pin"):
        dmPython.connect(**{**ssl_params, "ssl_path": str(cert_dir)})
