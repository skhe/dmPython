import os
import dmPython


def conn_params():
    return {
        "server": os.getenv("DM_HOST", "localhost"),
        "port": int(os.getenv("DM_PORT", "5236")),
        "user": os.getenv("DM_USER", "SYSDBA"),
        "password": os.getenv("DM_PASSWORD", "SYSDBA001"),
    }


def main() -> None:
    table = "SKH70_BULK"
    rows = [(i, f"user_{i}", i * 10) for i in range(1, 1001)]

    conn = dmPython.connect(**conn_params())
    cur = conn.cursor()

    cur.execute(f"DROP TABLE IF EXISTS {table}")
    cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, name VARCHAR(100), score INT)")

    cur.executemany(f"INSERT INTO {table}(id, name, score) VALUES (?, ?, ?)", rows)
    conn.commit()

    cur.execute(f"SELECT COUNT(*) FROM {table}")
    print("inserted rows:", cur.fetchone()[0])

    cur.execute(f"DROP TABLE IF EXISTS {table}")
    conn.commit()
    cur.close()
    conn.close()


if __name__ == "__main__":
    main()
