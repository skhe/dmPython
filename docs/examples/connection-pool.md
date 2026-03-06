# 连接池

驱动本身没有独立 `SessionPool` 类型时，可以在应用层实现简单连接池。

## 运行方式

```bash
python docs/examples/scripts/connection_pool.py
```

## 示例代码

```python
# docs/examples/scripts/connection_pool.py
import os
from contextlib import contextmanager
from queue import LifoQueue

import dmPython


class SimpleConnectionPool:
    def __init__(self, min_size: int = 1, max_size: int = 5):
        self.min_size = min_size
        self.max_size = max_size
        self._created = 0
        self._idle = LifoQueue(maxsize=max_size)
        for _ in range(min_size):
            self._idle.put(self._new_conn())

    def _conn_params(self):
        return {
            "server": os.getenv("DM_HOST", "localhost"),
            "port": int(os.getenv("DM_PORT", "5236")),
            "user": os.getenv("DM_USER", "SYSDBA"),
            "password": os.getenv("DM_PASSWORD", "SYSDBA001"),
        }

    def _new_conn(self):
        self._created += 1
        return dmPython.connect(**self._conn_params())

    @contextmanager
    def acquire(self):
        conn = None
        try:
            if not self._idle.empty():
                conn = self._idle.get()
            elif self._created < self.max_size:
                conn = self._new_conn()
            else:
                conn = self._idle.get()
            yield conn
        finally:
            if conn is not None:
                self._idle.put(conn)

    def closeall(self):
        while not self._idle.empty():
            conn = self._idle.get()
            conn.close()


def main() -> None:
    pool = SimpleConnectionPool(min_size=2, max_size=4)

    with pool.acquire() as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 1")
            print("conn-1:", cur.fetchone())

    with pool.acquire() as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 2")
            print("conn-2:", cur.fetchone())

    pool.closeall()


if __name__ == "__main__":
    main()
```
