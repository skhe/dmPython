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

## Remaining scope

The local test uses DM8's bundled legacy certificate without SAN, so it
exercises exact certificate pinning. CA and hostname verification for modern
SAN certificates is implemented but needs a server with a modern certificate
for an end-to-end regression. `ssl_pwd` and UKey login remain unsupported.
