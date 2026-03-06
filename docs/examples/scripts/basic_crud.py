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
    table = "SKH70_BASIC_CRUD"
    conn = dmPython.connect(**conn_params())
    cur = conn.cursor()

    cur.execute(f"DROP TABLE IF EXISTS {table}")
    cur.execute(f"CREATE TABLE {table} (id INT PRIMARY KEY, name VARCHAR(100), score INT)")

    cur.execute(f"INSERT INTO {table}(id, name, score) VALUES (?, ?, ?)", (1, "alice", 88))
    conn.commit()

    cur.execute(f"SELECT id, name, score FROM {table} WHERE id = ?", (1,))
    print("after insert:", cur.fetchone())

    cur.execute(f"UPDATE {table} SET score = ? WHERE id = ?", (95, 1))
    conn.commit()

    cur.execute(f"SELECT id, name, score FROM {table} WHERE id = ?", (1,))
    print("after update:", cur.fetchone())

    cur.execute(f"DELETE FROM {table} WHERE id = ?", (1,))
    conn.commit()

    cur.execute(f"SELECT COUNT(*) FROM {table}")
    print("after delete:", cur.fetchone())

    cur.execute(f"DROP TABLE IF EXISTS {table}")
    conn.commit()
    cur.close()
    conn.close()


if __name__ == "__main__":
    main()
