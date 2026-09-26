# SSL connection regression (2026-09-27)

## Local ARM verification

- Official DM8 ARM container with `ENABLE_ENCRYPT=1`: four SSL tests passed.
- The exact ARM development image pinned by CI, started with working directory
  `/opt/dmdbms/bin` and `ENABLE_ENCRYPT=1`: the same four tests passed.
- The image fails at startup with `SSL encrypt fail!` when left at its default
  working directory, `/home/dmdba`; the SSL CI job sets the working directory.
- The four checks cover encrypted login and query, an SSL path containing spaces
  and `&`, a wrong server-certificate pin, and a missing pin.
- Full real-database suite on the GB18030 instance: 188 passed, 6 deselected.
  The UTF8 instance lacks the admin credential required by four container-based
  cases; 184 passed, 4 skipped, 6 deselected there. GitHub CI supplies the
  required credential for both database encodings.

## Modern server certificate

An isolated ARM DM8 instance used a short-lived test CA, a server certificate
with `127.0.0.1` in its IP SAN, and a matching SYSDBA client certificate.
Without a `server-cert.pem` pin in the client directory, the ARM macOS Python
3.10 extension connected and queried successfully. Connecting as `localhost`
was rejected by hostname verification, and replacing the trusted CA with an
unrelated CA was rejected as an unknown authority. All three real-database
checks in `tests/ssl_modern` passed locally. The GitHub ARM SSL job now runs
the same checks after replacing the bundled certificates in its isolated
instance.

## Remaining scope

The ARM macOS Python 3.10 extension also connected to the SSL-enabled DM8
instance with the bundled RSA client key re-encrypted in traditional PEM
format. `ssl_pwd="test+ssl&pwd 123"` succeeded and executed a query; a missing
or wrong password failed. An OpenSSL-generated PBES2/AES-256-CBC PKCS#8 RSA
client key also connected and queried with `ssl_pwd="test+pkcs8&pwd 123"`;
missing and wrong passwords failed. All six SSL tests passed against the local
ARM DM8 instance. The password without `ssl_path` was rejected on the ordinary
DM8 instance. Because the PKCS#8 dependency updates `golang.org/x/text`, the
type matrix and Unicode CLOB regressions were rerun against local UTF8 and
GB18030 DM8 instances: 68 passed on each. UKey login remains unsupported.
