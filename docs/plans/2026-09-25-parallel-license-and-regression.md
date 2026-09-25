# 并行推进 Go 驱动许可与真实库回归

更新于 2026-09-25。两条工作线可以同时进行；构建和测试不等待许可答复，公开分发须同时通过许可与回归门槛。

## A. Go 驱动分发权利

联系对象是 `gitee.com/chunanyong/dm` 的维护者和达梦官方，而不是已经按 Mulan PSL v2 开源的 Python 驱动上游。向双方提供本仓库链接、所用版本 `v1.8.22`、本仓库修改内容及发布形态，要求书面确认：

1. Go 驱动源码和本仓库修改版能否在公开仓库及 sdist 中分发；
2. 编译后的 Go 桥接库能否随 macOS ARM64 wheel 在 GitHub Releases 和 PyPI 中分发；
3. 应保留哪些版权声明、许可证、NOTICE，以及是否有版本或用途限制。

记录答复的原始链接或文件与适用版本。若得不到明确授权，替换依赖并重新构建、回归；不能把官方 dmPython 的许可证扩展到这个单独引入的 Go 仓库。GitHub 仓库变量 `DMPYTHON_RELEASE_ENABLED` 在权利明确之前保持未设置。

## B. 真实数据库回归

先用 OrbStack 上的独立 ARM Linux 环境运行达梦官方开发/试用安装包或官方 ARM 镜像，固定安装包版本与 SHA-256、数据库版本、字符集、大小写设置和端口。官方文档要求安装包的 CPU 与系统版本匹配。Docker 适合有官方 ARM 镜像的情况；只有安装包时优先用独立 Ubuntu ARM 虚拟机。先跑单实例，不引入 Kubernetes。

已从达梦官方域名下载 ARM64 容器归档，在 OrbStack 启动独立实例；[首轮基线记录](../test-results/2026-09-25-orb-arm-baseline.md)包含校验值、服务端版本、测试结果和许可到期日。原有 `if010/dameng:latest` 是第三方 `linux/amd64` 镜像，本轮未使用，也不作为发布基线。

测试账号限定在专用 schema，使用 `DM_TEST_HOST`、`DM_TEST_PORT`、`DM_TEST_USER`、`DM_TEST_PASSWORD`。依次验证连接与版本、P0、P1、P2、崩溃防护；保存 JUnit 报告、服务端版本和残留测试表数量。每个失败都记录最小 SQL、驱动版本和服务端版本。

## CI 与发布门槛

- PR：GitHub 托管 ARM Linux runner 拉取摘要固定的社区开发镜像，启动临时数据库并执行 P0/P1；缺库、失败、跳过均不能通过。同仓库 PR 成功后构建五个 macOS ARM wheel；外部 fork PR 因拿不到 DPI 头文件 Secret 暂时只跑真实库，不把 wheel 视为已验证。
- main、定时/手动完整回归：运行全部 `requires_dm` 用例，并校验 JUnit 报告非空、无失败和跳过。
- 发布标签：`real-dm` 在真实库上执行全部 `requires_dm` 测试；只有它和五个 wheel 构建均通过，且 `DMPYTHON_RELEASE_ENABLED=true`，才创建 GitHub Release。PyPI 仍需单独建立可信发布流程，并以同样门槛约束。
- 当前发布回归固定在 CPython 3.10；五个 Python 版本只完成了 wheel 安装/导入检查。真实库稳定后逐步扩展到 3.9–3.13，并记录不同 DM8 服务端版本的兼容结果。
- 本机 OrbStack 官方镜像仅用于开发和对照验证；CI 在托管 ARM Linux runner 内使用固定摘要的第三方开发镜像新建临时数据库，无需本机连通性。达梦下载域名对 GitHub runner 返回 HTTP 403，所以不能在托管 runner 上直接使用官方归档。macOS wheel 从既有仓库 Secret 取得 DPI 头文件；仓库不上传镜像或头文件。详见 [CI 说明](../ci.md)。

## 本轮进度和下一步

- 已增加 `--require-dm`：没有选中真实库测试、缺连接参数或连不上数据库时，回归命令会失败。
- 已让标签发布依赖真实库回归；许可未明确时发布变量默认关闭。
- 已建隔离 ARM 数据库和非管理员账号；首轮本机完整回归 64 项通过，残留表数为 0。[证据与限制](../test-results/2026-09-25-orb-arm-baseline.md)。新增行为测试后，macOS ARM 本机 68 项通过；[GitHub 托管 ARM 完整回归](https://github.com/skhe/dmPython/actions/runs/36089633170) 66 项真实库用例通过，五个 macOS ARM wheel 构建和安装检查通过。发布仍等待 Go 驱动的分发权利明确。
