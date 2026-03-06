# LOB 大对象操作

## 运行方式

```bash
python docs/examples/scripts/lob_handling.py
```

## 示例代码

```python
# docs/examples/scripts/lob_handling.py
import os
import dmPython


def conn_params():
    return {
        "server": os.getenv("DM_HOST", "localhost"),
        "port": int(os.getenv("DM_PORT", "5236")),
        "user": os.getenv("DM_USER", "SYSDBA"),
        "password": os.getenv("DM_PASSWORD", "SYSDBA001"),
    }


def _read_lob(val):
    if hasattr(val, "read"):
        return val.read()
    return val


def main() -> None:
    table = "SKH70_LOB"
    text = "达梦 LOB 示例" * 500
    data = b"DMLOB" * 500

    conn = dmPython.connect(**conn_params())
    cur = conn.cursor()

    cur.execute(f"DROP TABLE IF EXISTS {table}")
    cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, c CLOB, b BLOB)")

    cur.execute(f"INSERT INTO {table}(id, c, b) VALUES (?, ?, ?)", (1, text, data))
    conn.commit()

    cur.execute(f"SELECT c, b FROM {table} WHERE id = ?", (1,))
    c_val, b_val = cur.fetchone()

    c_content = _read_lob(c_val)
    b_content = _read_lob(b_val)
    print("clob length:", len(c_content))
    print("blob length:", len(b_content))

    cur.execute(f"DROP TABLE IF EXISTS {table}")
    conn.commit()
    cur.close()
    conn.close()


if __name__ == "__main__":
    main()
```
