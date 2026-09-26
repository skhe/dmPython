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

An additional stop-primary trial did not prove automatic takeover. The
monitor's confirmation mode remained active; after roughly 35 seconds, the
second node still reported `STANDBY, OPEN`. The replicated high-precision
value remained readable there. Automatic promotion and reconnect after
promotion need a separately configured failover regression.
