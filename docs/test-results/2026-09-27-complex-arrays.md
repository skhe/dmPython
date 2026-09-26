# Complex object and array regression (2026-09-27)

The bridge now describes user-defined object elements inside VARRAY values
through `ALL_TYPES`, without requiring access to `SYS.SYSOBJECTS`. It also
binds nested array pointers correctly and returns errors for malformed complex
elements instead of terminating the Python process.

Real DM8 ARM verification:

- GB18030: complete suite **191 passed**, 6 non-database tests deselected.
- UTF8: object and array suite **12 passed**.
- New cases cover nullable fields in nested objects, VARRAY elements that are
  objects (including a null object and `DECIMAL(30,8)`), and nested VARRAYs
  containing high-precision decimals and null elements.

The vendored Go tests also exercise malformed nested elements and the modern
SSL CA/hostname validation path. Cross-schema object arrays and other DM8
server versions still need dedicated regression environments.
