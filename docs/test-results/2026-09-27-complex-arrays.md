# Complex object and array regression (2026-09-27)

The bridge now describes user-defined object elements inside VARRAY values
through `ALL_TYPES`, without requiring access to `SYS.SYSOBJECTS`. It also
binds nested array pointers correctly and returns errors for malformed complex
elements instead of terminating the Python process.

The first GitHub ARM run exposed an additional fetch bug: the C extension
passes an uninitialized nested object handle slot. The bridge now checks for
a live object handle before reusing that slot; the complete CI rerun is the
acceptance gate for this Linux-specific path.

Real DM8 ARM verification:

- GB18030: complete suite **191 passed**, 6 non-database tests deselected.
- UTF8: object and array suite **12 passed**.
- New cases cover nullable fields in nested objects, VARRAY elements that are
  objects (including a null object and `DECIMAL(30,8)`), and nested VARRAYs
  containing high-precision decimals and null elements.

The vendored Go tests also exercise malformed nested elements and the modern
SSL CA/hostname validation path. Cross-schema object arrays and other DM8
server versions still need dedicated regression environments.
