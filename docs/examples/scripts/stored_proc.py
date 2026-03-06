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
    conn = dmPython.connect(**conn_params())
    cur = conn.cursor()

    cur.execute(
        """
CREATE OR REPLACE PROCEDURE SKH70_PROC(p_in INT, p_out OUT INT)
AS
BEGIN
    p_out := p_in * 2;
END;
"""
    )

    cur.execute(
        """
CREATE OR REPLACE FUNCTION SKH70_FUNC(p_in INT)
RETURN INT
AS
BEGIN
    RETURN p_in + 100;
END;
"""
    )
    conn.commit()

    proc_result = cur.callproc("SKH70_PROC", [21, None])
    func_result = cur.callfunc("SKH70_FUNC", [23])

    print("callproc:", proc_result)
    print("callfunc:", func_result)

    cur.close()
    conn.close()


if __name__ == "__main__":
    main()
