# Real DM8 primary/standby routing (2026-09-27)

An isolated two-node DM8 watcher group and monitor ran in local Orb, using the
2025-09-24 ARM image. `V$INSTANCE` reported `PRIMARY, OPEN` and `STANDBY, OPEN`
with the same OGUID. Both endpoints accepted direct Python connections.

The image's `DOCKER_DMWATCHER` startup script omitted the `TIME_ZONE` template
substitution. The temporary containers replaced `ENV_21` with `+08:00` before
running the image startup script. This workaround was confined to the local
test containers.

In the cluster network, a Go driver probe found that `rwSeparate=1` sent a
read-only query to the standby, while `rwSeparate=4` failed by dialing `:0`.
The server did not provide a standby address in the mode 4 login response. The
driver now falls back to its existing valid-standby metadata query when that
address is absent.

After the fix, `scripts/verify_dm_ha.py` built and ran with the ARM Linux
Python extension. With autocommit enabled and 0% primary reads:

| Mode | Query destination |
| --- | --- |
| `rwseparate=0` | Primary (`GRP453932_DW1`) |
| `rwseparate=1` | Standby (`GRP453932_DW2`) |
| `rwseparate=4` | Standby (`GRP453932_DW2`) |

The script then inserted `12345678901234567890.12345678` into a
`DECIMAL(30,8)` column through mode 4. A direct standby connection read the
same value after replication, and the script removed its test table.

To repeat the check, provide `DM_HA_PRIMARY_HOST`, `DM_HA_STANDBY_HOST`,
`DM_HA_USER`, and `DM_HA_PASSWORD` to `scripts/verify_dm_ha.py`. Both hosts
must be reachable from the process running the driver. Optional
`DM_HA_PRIMARY_PORT` and `DM_HA_STANDBY_PORT` default to 5236. The account
needs permission to create and drop a table on the primary.

This HA check currently runs locally: the GitHub hosted runners cannot reach
the user's Orb network. The regular GitHub ARM real-database matrix still
checks single-node behavior and build compatibility. MPP has since been
verified separately; UKey authentication remains unverified.

## Automatic takeover and new connections

An initial trial stopped the entire primary container. After roughly 35
seconds, the second node still reported `STANDBY, OPEN`, and the monitor could
no longer resolve the stopped container's hostname. That trial did not prove
takeover. Confirmation mode was correctly enabled: [the DM monitor manual](https://eco.dameng.com/document/dm/zh-cn/pm/data-watch-monitor.html)
requires a confirmation monitor for automatic takeover, while [the watcher
configuration manual](https://eco.dameng.com/document/dm/zh-cn/pm/configuration-description)
defines `DW_MODE=AUTO` as automatic fault switching.

The follow-up kept the primary container and its network identity running,
but stopped both `dmserver` and `dmwatcher` inside it. Before the failure, a
`DECIMAL(30,8)` value of `12345678901234567890.12345678` had reached the
standby. The monitor log recorded `start to auto takeover` and a successful
`ALTER DATABASE PRIMARY`; within the 20-second observation, the standby
reported `PRIMARY, OPEN`. A new ARM macOS Python 3.10 connection through a
two-endpoint `dm_svc.conf` service with `LOGIN_MODE=1` reached the promoted
node. It read the original value exactly and wrote and read
`98765432109876543210.87654321` exactly.

The former primary then rejoined as `STANDBY, OPEN`. The same procedure was
repeated in the opposite direction with `scripts/verify_dm_failover.py`:
`prepare` checked the service route and standby replication; after stopping
the current primary's database and watcher, `verify` connected to the newly
promoted primary, checked both exact decimal values, and removed its test
table. Both phases passed. The script requires a unique `--table` name,
`DM_HA_USER`, `DM_HA_PASSWORD`, `DM_HA_SERVICE_NAME`, and
`DM_HA_SERVICE_PATH`; `prepare` also needs `DM_HA_STANDBY_HOST` and optional
`DM_HA_STANDBY_PORT`. Pass the standby instance name printed by `prepare` as
`--expected-new-primary` to `verify`. The operator triggers the fault between
the two phases. This two-phase check verified a **new connection** after
takeover; the already-open case was tested separately below.

The service configuration used two reachable endpoints:

```ini
DMPY_HA=(primary-host:5236,standby-host:5236)
[DMPY_HA]
LOGIN_MODE=1
EP_SELECTION=1
SWITCH_TIMES=1
SWITCH_INTERVAL=0
```

## Existing connection after a single-node restart

An independent, disposable ARM DM8 instance was restarted while Python 3.10
held an open autocommit connection. The same Python connection queried again
after one visible communication error. A second experiment inserted an exact
`DECIMAL(30,8)` value in manual-commit mode and verified that a separate
connection could not see it. Before the bridge fix, a database restart made
`commit()` return success even though the row was lost. Manual transactions now
pin one physical connection; `commit()` reports an error after that connection
is lost, and a fresh connection confirms the row was not persisted. The
repeatable check is `scripts/verify_dm_restart_transaction.py`; it also runs
in a dedicated GitHub ARM CI job. Local full real-database regression with a
dedicated test user passed 193 cases.

## Existing connections across a real primary/standby takeover

The follow-up used a new isolated two-node watcher group and confirmation
monitor in local Orb, with the same ARM image
`dm8:dm8_20250924_rev288894_HWarm_kylin10_64` (image ID
`sha256:eb5f243c8f4322596d0eac611c0fa2d8cd8a03a5d3f69889f3549a530354b210`).
The primary was `GRP453932_DW1`, and the standby was `GRP453932_DW2`. The
service listed both endpoints with `LOGIN_MODE=1`. The image's missing
`TIME_ZONE` substitution was repaired only inside these temporary containers.

`scripts/verify_dm_open_connection_failover.py` kept an autocommit service
connection and an uncommitted manual-transaction service connection open on
the primary. It wrote `12345678901234567890.12345678` to a `DECIMAL(30,8)`
column and confirmed the exact value on the standby. It then inserted an
uncommitted second row and confirmed that the standby could not see it. The
script killed `dmwatcher` and `dmserver` inside the disposable primary
container, preserving the container's network identity. The monitor logged
`AUTO TAKEOVER GRP453932_DW2` and `auto takeover success` at 04:36:25–27
local time. The standby became `PRIMARY, OPEN`.

The old transaction's `commit()` raised `dmPython.Error`; the lost row remained
absent. The *same* already-open autocommit Python connection reached the
promoted primary without a visible query error in this run, then wrote and
read `98765432109876543210.87654321` exactly. The promoted primary also read
the original replicated value exactly. The verifier removed its table.

After the former primary rejoined as `STANDBY, OPEN`, the same verifier ran in
the reverse direction. The monitor logged automatic takeover by
`GRP453932_DW1` at 04:39:57–58. The already-open service connection reached
that promoted node with zero visible query errors; the lost manual transaction
again raised on `commit()`, and both exact decimal checks passed.

To repeat this check against a disposable HA group, set `DM_HA_USER`,
`DM_HA_PASSWORD`, `DM_HA_SERVICE_NAME`, `DM_HA_SERVICE_PATH` (directory
containing `dm_svc.conf`), `DM_HA_STANDBY_HOST`, optional
`DM_HA_STANDBY_PORT`, and `DM_HA_PRIMARY_CONTAINER`. The service must route
only to the primary; the standby address must connect directly. Run the script
from a Python environment with the local extension and bridge library. It
stops the named primary's database and watcher processes. This local HA check
is not yet a GitHub-hosted CI gate because the runner cannot reach Orb.
