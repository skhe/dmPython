# OrbStack 官方 DM8 ARM 镜像首轮基线

运行日期：2026-09-25。此记录针对本机未提交的工作区，Git HEAD 为 `5329dec6de0e8f267c9892ef5be29bd33609d808`；不能将结果视为已发布版本或 GitHub CI 的通过证明。

## 数据库与隔离

- 来源：[达梦社区提供的达梦官方域名 ARM 镜像链接](https://eco.dameng.com/community/question/ff0127e0ff8a14d2df03fa4398512d0d)，下载归档名 `dm8_20250924_HWarm_kylin10_sp1_64_rq_ent_8.1.4.80_pack29.tar`，SHA-256 `f08359576241c17110446aa3e19794650b95add43840a5da9d5070240b1e53c8`。
- 导入后镜像：`dm8:dm8_20250924_rev288894_HWarm_kylin10_64`，`linux/arm64`，镜像 ID `sha256:eb5f243c8f4322596d0eac611c0fa2d8cd8a03a5d3f69889f3549a530354b210`。
- 容器：`dmpython-dm8-arm-baseline`；数据卷：`dmpython-dm8-arm-baseline`；仅映射 `127.0.0.1:15236` 到容器 `5236`。实例名 `DMPYTEST`，UTF-8 初始化标志 `UNICODE_FLAG=1`，页大小 16 KB。
- 服务端 `ID_CODE`：`--03134284368-20250821-288894-20149 Pack29`；默认试用许可到期日 `2026-10-09`，到期前需要换新官方介质或取得有效测试授权。
- 测试账号 `DMPYTEST`，仅授予 `RESOURCE` 角色；账号拥有独立 schema。凭据仅存于本机 `/private/tmp/dmpython-dm8-arm-baseline.env`，权限 `0600`，不进入 Git。账号建表、插入、查询、删表已单独验证。

## 驱动与测试

- 在 macOS ARM64 上用 Go 1.26.5、CPython 3.10.20 从当前工作区重新构建 `libdmdpi.dylib` 和 `dmPython.cpython-310-darwin.so`。驱动报告版本 `2.5.32`，连接后的 `SELECT 1` 返回 `(1,)`。
- 命令：`DYLD_LIBRARY_PATH="$PWD/dpi_bridge" PYTHONPATH="$PWD" python -m pytest -q --require-dm -m requires_dm tests`，环境变量从上述本地凭据文件载入。结果：**62 passed、0 failed、0 skipped、2 deselected**，12.678 秒。
- 再执行 `python -m pytest -q --require-dm tests`：**64 passed、0 failed、0 skipped**，12.687 秒，覆盖 P0/P1/P2、旧集成用例和两项本地检查。
- 两次 JUnit 报告分别保存在 `/private/tmp/dmpython-baseline-2026-09-25.xml`、`/private/tmp/dmpython-baseline-full-2026-09-25.xml`。回归后查询 `DMPYTEST.USER_TABLES`，残留表数为 **0**。
- Pytest 报告了 `asyncio_default_fixture_loop_scope` 未识别的配置警告；本次无测试失败。当前验证仅覆盖 CPython 3.10 和这一版 DM8 服务端，尚未证明 3.9–3.13 的真实库行为或不同服务端版本兼容性。

## 后续

本地 Orb 数据库不能直接作为 GitHub 托管 runner 的测试地址；接入 CI 需要安全连通方案。公开发布还取决于内置 Go 驱动的分发许可，和本次数据库测试通过是两个独立门槛。
