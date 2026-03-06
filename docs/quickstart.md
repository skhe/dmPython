# 快速开始

## 最小可运行示例

```python
import dmPython

conn = dmPython.connect(
    user="SYSDBA",
    password="SYSDBA001",
    server="localhost",
    port=5236,
)

cur = conn.cursor()
cur.execute("SELECT 1")
print(cur.fetchone())

cur.close()
conn.close()
```

## 使用上下文管理器

```python
import dmPython

with dmPython.connect(user="SYSDBA", password="SYSDBA001", server="localhost", port=5236) as conn:
    with conn.cursor() as cur:
        cur.execute("SELECT SYSTIMESTAMP")
        print(cur.fetchone())
```

## 下一步

- 查看 [API 参考](api-reference.md)
- 查看 [示例目录](examples/basic-crud.md)
