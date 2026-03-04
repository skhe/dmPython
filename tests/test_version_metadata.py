from __future__ import annotations

import importlib.util
from pathlib import Path
import re

import dmPython


ROOT = Path(__file__).resolve().parents[1]


def _project_version() -> str:
    text = (ROOT / "pyproject.toml").read_text(encoding="utf-8")
    match = re.search(r'^\s*version\s*=\s*"([^"]+)"\s*$', text, re.MULTILINE)
    assert match is not None
    return match.group(1)


def _load_smoke_script_module():
    script_path = ROOT / "scripts" / "test_connection.py"
    spec = importlib.util.spec_from_file_location("test_connection_script", script_path)
    assert spec is not None
    assert spec.loader is not None
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def test_module_version_matches_project_metadata():
    assert dmPython.version == _project_version()


def test_smoke_script_uses_dynamic_expected_version():
    module = _load_smoke_script_module()
    assert module.get_expected_version() == _project_version()

    script_text = (ROOT / "scripts" / "test_connection.py").read_text(encoding="utf-8")
    assert 'dmPython.version == "2.5.30"' not in script_text
