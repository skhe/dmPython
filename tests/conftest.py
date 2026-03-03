from __future__ import annotations

import os
import subprocess
import sys
import uuid
from pathlib import Path
from typing import Callable

import pytest

import dmPython


ROOT_DIR = Path(__file__).resolve().parents[1]
DYLIB_DIR = ROOT_DIR / "dpi_bridge"

_DM_AVAILABLE: bool | None = None
_DM_ERROR: str | None = None
_DEFAULT_BOUNDARY_SIZES = "0,1,2,255,256,1023,1024,2047,2048,4095,4096,4097,8191,8192,8193,16384"


def dm_conn_params() -> dict[str, object]:
    return {
        "user": os.getenv("DM_TEST_USER", "SYSDBA"),
        "password": os.getenv("DM_TEST_PASSWORD", "SYSDBA001"),
        "server": os.getenv("DM_TEST_HOST", "localhost"),
        "port": int(os.getenv("DM_TEST_PORT", "5237")),
    }


def _probe_dm() -> tuple[bool, str]:
    params = dm_conn_params()
    try:
        conn = dmPython.connect(**params)
        cur = conn.cursor()
        cur.execute("SELECT 1")
        cur.fetchone()
        cur.close()
        conn.close()
        return True, ""
    except Exception as exc:  # pragma: no cover - probe helper
        return False, str(exc)


def pytest_runtest_setup(item: pytest.Item) -> None:
    global _DM_AVAILABLE, _DM_ERROR

    if item.get_closest_marker("requires_dm") is None:
        return

    if _DM_AVAILABLE is None:
        _DM_AVAILABLE, _DM_ERROR = _probe_dm()

    if not _DM_AVAILABLE:
        pytest.skip(f"DM not available: {_DM_ERROR}")


@pytest.fixture(scope="session")
def conn_params() -> dict[str, object]:
    return dm_conn_params()


@pytest.fixture()
def conn(conn_params):
    c = dmPython.connect(**conn_params)
    yield c
    c.close()


@pytest.fixture()
def cursor(conn):
    cur = conn.cursor()
    yield cur
    cur.close()


@pytest.fixture(scope="session")
def stress_rows() -> int:
    return int(os.getenv("DM_TEST_STRESS_ROWS", "2000"))


@pytest.fixture(scope="session")
def stress_workers() -> int:
    return int(os.getenv("DM_TEST_STRESS_WORKERS", "4"))


@pytest.fixture(scope="session")
def boundary_sizes() -> list[int]:
    raw = os.getenv("DM_TEST_BOUNDARY_SIZES", _DEFAULT_BOUNDARY_SIZES)
    values: list[int] = []
    for item in raw.split(","):
        text = item.strip()
        if not text:
            continue
        value = int(text)
        if value < 0:
            raise ValueError(f"DM_TEST_BOUNDARY_SIZES contains negative size: {value}")
        values.append(value)
    if not values:
        raise ValueError("DM_TEST_BOUNDARY_SIZES produced an empty size list")
    return values


@pytest.fixture(scope="session")
def churn_loops() -> int:
    return int(os.getenv("DM_TEST_CHURN_LOOPS", "120"))


@pytest.fixture(scope="session")
def gc_loops() -> int:
    return int(os.getenv("DM_TEST_GC_LOOPS", "150"))


@pytest.fixture()
def table_name_factory() -> Callable[[str], str]:
    def _mk(prefix: str = "DMPY_TEST") -> str:
        return f"{prefix}_{uuid.uuid4().hex[:10].upper()}"

    return _mk


@pytest.fixture()
def drop_table() -> Callable[[object, str], None]:
    def _drop(cur, table_name: str) -> None:
        try:
            cur.execute(f"DROP TABLE {table_name}")
        except Exception:
            pass

    return _drop


@pytest.fixture()
def run_in_subprocess(conn_params) -> Callable[[str, int], subprocess.CompletedProcess[str]]:
    def _run(code: str, timeout: int = 60) -> subprocess.CompletedProcess[str]:
        env = os.environ.copy()
        env["DM_TEST_HOST"] = str(conn_params["server"])
        env["DM_TEST_PORT"] = str(conn_params["port"])
        env["DM_TEST_USER"] = str(conn_params["user"])
        env["DM_TEST_PASSWORD"] = str(conn_params["password"])

        dylib_path = str(DYLIB_DIR)
        existing = env.get("DYLD_LIBRARY_PATH", "")
        env["DYLD_LIBRARY_PATH"] = f"{dylib_path}:{existing}" if existing else dylib_path

        return subprocess.run(
            [sys.executable, "-X", "faulthandler", "-c", code],
            capture_output=True,
            text=True,
            env=env,
            timeout=timeout,
        )

    return _run
