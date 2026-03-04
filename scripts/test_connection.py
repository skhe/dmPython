"""dmPython 连接达梦数据库测试脚本。

测试内容：
1. 基本连接
2. 简单查询 (SELECT 1)
3. DDL — 建表、插入、查询、删除
4. 参数化查询
5. 事务回滚
6. 游标属性
7. 连接属性
"""
import sys
import traceback
import os
from pathlib import Path

import dmPython

# 连接参数 — docker 容器 dm8_test 映射到宿主机 5237
CONN_PARAMS = {
    "user": os.getenv("DM_TEST_USER", "SYSDBA"),
    "password": os.getenv("DM_TEST_PASSWORD", "SYSDBA001"),
    "server": os.getenv("DM_TEST_HOST", "localhost"),
    "port": int(os.getenv("DM_TEST_PORT", "5237")),
}

TEST_TABLE = "DMPYTHON_TEST_TBL"
passed = 0
failed = 0


def run_test(name, func):
    global passed, failed
    try:
        func()
        print(f"  [PASS] {name}")
        passed += 1
    except Exception as e:
        print(f"  [FAIL] {name}: {e}")
        traceback.print_exc()
        failed += 1


def _read_project_version() -> str:
    pyproject = Path(__file__).resolve().parents[1] / "pyproject.toml"
    for line in pyproject.read_text(encoding="utf-8").splitlines():
        stripped = line.strip()
        if stripped.startswith("version = "):
            value = stripped.split("=", 1)[1].strip()
            return value.strip('"')
    raise RuntimeError("Cannot read project version from pyproject.toml")


def get_expected_version() -> str:
    return _read_project_version()


def test_version():
    assert hasattr(dmPython, "version"), "缺少 version 属性"
    expected = get_expected_version()
    assert dmPython.version == expected, f"版本不匹配: got={dmPython.version}, expected={expected}"


def test_connect():
    conn = dmPython.connect(**CONN_PARAMS)
    assert conn is not None
    conn.close()


def test_select_one():
    conn = dmPython.connect(**CONN_PARAMS)
    cur = conn.cursor()
    cur.execute("SELECT 1")
    row = cur.fetchone()
    assert row == (1,), f"期望 (1,), 实际 {row}"
    cur.close()
    conn.close()


def test_server_info():
    conn = dmPython.connect(**CONN_PARAMS)
    cur = conn.cursor()
    cur.execute("SELECT * FROM V$VERSION")
    rows = cur.fetchall()
    assert len(rows) > 0, "V$VERSION 应返回至少一行"
    for row in rows:
        print(f"    {row}")
    cur.close()
    conn.close()


def test_ddl_and_dml():
    conn = dmPython.connect(**CONN_PARAMS)
    cur = conn.cursor()

    # 清理
    try:
        cur.execute(f"DROP TABLE {TEST_TABLE}")
    except Exception:
        pass

    # 建表
    cur.execute(f"""
        CREATE TABLE {TEST_TABLE} (
            id   INT PRIMARY KEY,
            name VARCHAR(100),
            val  DECIMAL(10, 2)
        )
    """)

    # 插入
    cur.execute(f"INSERT INTO {TEST_TABLE} VALUES (1, '测试数据', 3.14)")
    cur.execute(f"INSERT INTO {TEST_TABLE} VALUES (2, 'dmPython', 2.718)")
    conn.commit()

    # 查询
    cur.execute(f"SELECT * FROM {TEST_TABLE} ORDER BY id")
    rows = cur.fetchall()
    assert len(rows) == 2, f"期望 2 行, 实际 {len(rows)}"
    assert rows[0][1] == "测试数据"
    assert rows[1][1] == "dmPython"

    # 清理
    cur.execute(f"DROP TABLE {TEST_TABLE}")
    conn.commit()
    cur.close()
    conn.close()


def test_parameterized_query():
    conn = dmPython.connect(**CONN_PARAMS)
    cur = conn.cursor()

    try:
        cur.execute(f"DROP TABLE {TEST_TABLE}")
    except Exception:
        pass

    cur.execute(f"CREATE TABLE {TEST_TABLE} (id INT, name VARCHAR(100))")

    # 参数化插入
    cur.execute(f"INSERT INTO {TEST_TABLE} VALUES (?, ?)", (1, "参数化测试"))
    conn.commit()

    # 参数化查询
    cur.execute(f"SELECT name FROM {TEST_TABLE} WHERE id = ?", (1,))
    row = cur.fetchone()
    assert row[0] == "参数化测试", f"期望 '参数化测试', 实际 {row[0]}"

    cur.execute(f"DROP TABLE {TEST_TABLE}")
    conn.commit()
    cur.close()
    conn.close()


def test_rollback():
    conn = dmPython.connect(**CONN_PARAMS)
    conn.autocommit = False
    cur = conn.cursor()

    try:
        cur.execute(f"DROP TABLE {TEST_TABLE}")
    except Exception:
        pass

    cur.execute(f"CREATE TABLE {TEST_TABLE} (id INT)")
    conn.commit()

    cur.execute(f"INSERT INTO {TEST_TABLE} VALUES (999)")
    conn.rollback()

    cur.execute(f"SELECT COUNT(*) FROM {TEST_TABLE}")
    count = cur.fetchone()[0]
    assert count == 0, f"回滚后期望 0 行, 实际 {count}"

    cur.execute(f"DROP TABLE {TEST_TABLE}")
    conn.commit()
    cur.close()
    conn.close()


def test_cursor_description():
    conn = dmPython.connect(**CONN_PARAMS)
    cur = conn.cursor()
    cur.execute("SELECT 1 AS col_a, 'hello' AS col_b")
    assert cur.description is not None
    assert len(cur.description) == 2
    assert cur.description[0][0].upper() == "COL_A"
    assert cur.description[1][0].upper() == "COL_B"
    cur.close()
    conn.close()


def test_connection_attributes():
    conn = dmPython.connect(**CONN_PARAMS)
    # 检查 autocommit 属性
    original = conn.autocommit
    conn.autocommit = True
    assert conn.autocommit == True
    conn.autocommit = original
    conn.close()


def main():
    print(f"dmPython version: {dmPython.version}")
    print(f"Python: {sys.executable}")
    print(f"连接: {CONN_PARAMS['server']}:{CONN_PARAMS['port']}")
    print()

    tests = [
        ("模块版本检查", test_version),
        ("基本连接", test_connect),
        ("SELECT 1", test_select_one),
        ("服务器版本信息", test_server_info),
        ("建表/插入/查询/删表", test_ddl_and_dml),
        ("参数化查询", test_parameterized_query),
        ("事务回滚", test_rollback),
        ("游标 description", test_cursor_description),
        ("连接属性", test_connection_attributes),
    ]

    print(f"运行 {len(tests)} 个测试:")
    for name, func in tests:
        run_test(name, func)

    print()
    print(f"结果: {passed} 通过, {failed} 失败")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
