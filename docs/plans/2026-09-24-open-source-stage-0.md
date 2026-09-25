# 开源发布阶段 0：范围与发布前置条件

更新于 2026-09-25。目标是先确定包身份、支持边界、额外引入的 Go 驱动的许可状态和可重复的真实数据库测试环境，再推进自动发布到 PyPI。此阶段不创建公开 PyPI 版本。当前优先级见[项目路线图](../ROADMAP.md)。

## 执行状态

| 事项 | 状态 | 负责人 | 完成标准 |
| --- | --- | --- | --- |
| 包名与版本规则 | 已确定，待首次发布复核 | 维护者 | 使用独立分发名 `dmPython-macOS`；发布前再次确认 PyPI 名称可用并核对版本未被占用 |
| 平台与 Python 范围 | 已落实到元数据和文档 | 维护者 | 构建目标为 macOS 14+ ARM64、CPython 3.9–3.13；真实数据库行为以测试报告为准 |
| 源码包内容 | 已修复并设 CI 检查 | 维护者 | sdist 不含本地 DPI 头文件，包含内置 Go 模块的 `go.mod/go.sum` |
| 内置 Go 驱动的许可 | 未证实可再分发；当前产物暂停发布 | 维护者 | 找到覆盖该 Go 驱动源码、修改版及编译产物的许可证或许可说明，或改用许可明确的实现；[核查记录](2026-09-24-go-driver-license-research.md) |
| 隔离测试库 | 本机官方基线和 GitHub CI 均已就绪；本机试用许可将到期 | 维护者 | [OrbStack ARM 实例首轮基线](../test-results/2026-09-25-orb-arm-baseline.md) 64 项通过、无残留表；[GitHub ARM 完整回归](../test-results/2026-09-25-github-arm-ci.md) 66 项通过；本机官方镜像许可 2026-10-09 到期前需续用有效测试介质 |

## 1. 包身份与版本

- PyPI 分发名沿用仓库元数据中的 `dmPython-macOS`；Python 导入名仍为 `dmPython`。这与官方 `dmPython` 的导入名相同，用户应在独立虚拟环境中安装，避免两个分发包争用同一个模块。
- 2026-09-24 查询时，PyPI 的 `dmPython` 项目已存在，`dmPython-macOS` 查询返回 404。404 仅表示当时未找到项目，不能视为名称已保留。首次发布前用 TestPyPI/PyPI 的实际上传结果最终确认。
- 版本独立于官方项目递增。当前版本为 `2.5.32`，下一次公开候选版本拟为 `2.5.33`；只有通过发布门槛后才修改版本和创建标签。不得覆盖已发布的同版本文件。
- PyPI 发布由 GitHub Actions 的可信发布流程承担；发布工作流和具体触发规则放在后续阶段实施。

## 2. 支持边界

- 预编译 wheel 的目标：macOS 14+、ARM64、CPython 3.9–3.13。`pyproject.toml` 与 `setup.py` 的 Python 约束保持一致。
- 构建和导入通过，只证明该组合可安装并加载。CPython 3.9–3.13 的现有真实库用例已有[逐版本报告](../test-results/2026-09-25-five-python-release-rehearsal.md)；不同 DM8 服务端版本、SQLAlchemy/Django 兼容性仍需另行验证。
- 项目暂按 Beta 标识。生产 SLA 和厂商认证不在当前承诺范围内。

## 3. 源码许可与附加组件

| 组件 | 当前证据 | 发布前动作 |
| --- | --- | --- |
| 官方 dmPython 的 C/Python 源码 | 上游仓库明确标注 Mulan PSL v2；本仓库保留了许可声明 | 按该开源许可证保留声明与来源，并在发布包内附完整许可文本；无需另行索要这部分源码的授权 |
| DPI 头文件 | 上游说明构建时从 DM 安装或驱动目录取得；本地 `dpi_include/` 不受 Git 跟踪 | 检查头文件不进入 sdist、wheel 和公开构建日志；若 SDK 条款另有限制，再按条款处理 |
| `gitee.com/chunanyong/dm` v1.8.22 | 本仓库内置并修改源码；Gitee 页面和该版本模块包均未找到许可证，源码带有达梦版权声明；达梦[官方 FAQ](https://eco.dameng.com/document/dm/zh-cn/faq/faq-go-new.html)说明 Git 仓库非官方维护 | 参阅[核查记录](2026-09-24-go-driver-license-research.md)；取得适用的许可说明或替换实现，并核对编译进 wheel 的桥接库 |

**门槛：** dmPython 源码已有明确开源许可证。当前不能直接把该许可证推及另一个仓库的 Go 驱动；在其许可状态或替代实现确定前，暂缓将包含该 Go 驱动的 sdist/wheel 上传 PyPI/TestPyPI。现有 GitHub Release 不构成 Go 驱动许可的证据。若找到适用的正式许可文件，应以文件条款为准，无需默认要求另行联系权利人。

本仓库目前的 `LICENSE` 文件是 Mulan PSL v2 的许可声明和链接，并非完整条款；正式发布包还需附上完整许可文本。Mulan PSL v2 允许按条款分发修改后的源码和可执行文件，同时要求向接收者提供许可证副本并保留有关声明。

构建发现：原始 `MANIFEST.in` 在本地生成的 sdist 中意外带入 4 个 DPI 头文件，并遗漏内置 Go 模块的 `go.mod/go.sum`。现已修正清单，并在工作流中检查源包内容。公开发布前仍需检查最终 wheel 的实际文件列表和动态库来源。

## 4. 隔离测试库准备

1. 选择专用 DM8 测试实例或与生产隔离的数据库，记录服务端版本、字符集、操作系统、连接地址和快照/重建方式。CI 运行器必须可达；不要把生产库用作回归目标。
2. 建立专用的非 `SYSDBA` 测试账号，仅授予测试所需的建表、读写、删表权限。凭据保存在 GitHub Actions Secrets 或隔离的本地环境中，不写入仓库。
3. 用 `DM_TEST_HOST`、`DM_TEST_PORT`、`DM_TEST_USER`、`DM_TEST_PASSWORD` 配置连接。测试已改为必须显式提供这四项，子进程测试也沿用相同配置。GitHub PR 在临时 DM8 容器内运行 P0/P1 真实库测试；主分支、定时、手动完整回归和发布标签运行全部 `requires_dm` 用例。各门槛均使用 `--require-dm`，缺库、断连、失败或跳过都不能通过。
4. 现有集成测试使用随机表名并尝试清理，但异常退出可能留表。测试账号只允许操作专用 schema；每次运行前后检查该 schema 中的 `DMPY_TEST_%` 等测试表，按保留期限清理并记录结果。不要在共享 schema 中执行自动清理。
5. 验收时保存一次连接探测、一次 P0/P1/P2 完整回归和测试前后残留表数量。若 CI 因缺少 Secrets 跳过集成测试，阶段 0 仍未完成。

## 退出条件与后续顺序

阶段 0 完成需要同时满足：包名复核、平台元数据一致、源码包内容检查通过、内置 Go 驱动的许可状态或替代方案明确、隔离测试库及非管理员账号就绪、完整真实数据库回归有记录。当前本机隔离库和 GitHub ARM 完整回归均已有记录；Go 驱动分发权利仍未明确，本机官方试用许可将于 2026-10-09 到期，首次发布前还需复核包名和最终产物。因此阶段 0 不能标为完成。

Go 驱动许可沟通与真实库测试环境按[并行推进方案](2026-09-25-parallel-license-and-regression.md)同时进行。测试覆盖与故障分类、GitHub Actions 构建/真实库门禁可先完成；只有分发权利和回归门槛同时满足后，才进入 TestPyPI、PyPI 可信发布与回滚/撤销方案。每一步的放行证据应进入发布检查表和对应 GitHub Actions 运行记录。

## 核查来源

- [官方 dmPython 仓库](https://github.com/DamengDB/dmPython)
- [官方 dmPython 许可证](https://github.com/DamengDB/dmPython/blob/main/LICENSE)
- [Mulan PSL v2 完整条款](https://spdx.org/licenses/MulanPSL-2.0.html)
- [Gitee 上的 Go 驱动仓库](https://gitee.com/chunanyong/dm)
- [PyPI 官方 dmPython 项目](https://pypi.org/project/dmpython/)
