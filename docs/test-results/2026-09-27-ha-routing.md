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
checks single-node behavior and build compatibility. MPP routing and UKey
authentication remain unverified.

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
the two phases. Existing open connections were not tested for transparent
reconnection; the verified behavior is a **new connection** after takeover.

The service configuration used two reachable endpoints:

```ini
DMPY_HA=(primary-host:5236,standby-host:5236)
[DMPY_HA]
LOGIN_MODE=1
EP_SELECTION=1
SWITCH_TIMES=1
SWITCH_INTERVAL=0
```
