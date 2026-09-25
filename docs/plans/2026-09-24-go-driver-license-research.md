# 达梦 Go 驱动公开许可核查

核查日期：2026-09-24，2026-09-25 复核。范围是本项目内置的 `gitee.com/chunanyong/dm` Go 驱动，以及达梦公开提供的 Go 驱动源码。本文记录可复核的事实；不将“可下载、可用于开发”推断为“可向 PyPI 再分发”。

## 结论

截至核查时，**没有找到由达梦发布、明确允许第三方将 Go 驱动源码、修改版或编译产物再分发到 PyPI 的公开许可文本**。达梦官方旧版 Go 驱动压缩包内没有 `LICENSE`、`COPYING`、`NOTICE` 或其他授权文件，源码带有达梦的版权和 `All rights reserved` 声明。这不能证明所有版本都没有许可条款：项目采用的 8.1.4.170 版官方安装包尚未取得并检查。

**当前发布决定：** 本项目现有 sdist 和 wheel 都包含该 Go 驱动的源码或编译产物。在取得覆盖这些分发方式的许可依据，或移除/替换该驱动之前，不将这些产物上传 PyPI/TestPyPI。这个决定基于“尚未证实可以再分发”，并非断言任何人使用该驱动都违法。

达梦[官方 FAQ](https://eco.dameng.com/document/dm/zh-cn/faq/faq-go-new.html)在“Github 上的代码和 gitee 上的代码没有同步更新么?”一节说明：Go 驱动由数据库安装包提供，Git 上的代码不是达梦官方上传和维护的；最新驱动应从安装包的 `drivers/go` 获取。因此，第三方 GitHub/Gitee 镜像上的许可证或仓库可见性，不能单独证明达梦对原始驱动源码授予了相同许可。

## 官方来源与包内证据

| 来源 | 查到的事实 | 对许可判断的限度 |
| --- | --- | --- |
| 达梦[Go 驱动 FAQ](https://eco.dameng.com/document/dm/zh-cn/faq/faq-go-new.html) | 明确区分官方安装包与非官方维护的 Git 仓库，并称最新驱动在安装包 `drivers/go` 内 | 说明来源；未给出再分发许可 |
| 达梦[GO_DM 开发文档](https://eco.dameng.com/document/dm/zh-cn/start/GO_DM_NEW.html) | 指导用户把安装目录下的 Go 驱动解压并复制到本机 Go `src` 目录 | 指导本地开发使用；未说明第三方公开再分发 |
| 达梦[安装及卸载文档](https://eco.dameng.com/document/dm/zh-cn/pm/install-uninstall.html) | 把驱动列为安装组件；图形化安装时要求阅读并接受安装协议，否则无法继续 | 证实安装协议是需要核查的独立材料；公开页面没有给出协议正文，不能据此推定再分发权限 |
| 达梦[官方 Gitee 组织](https://gitee.com/DamengDB) | 公开项目列表含 `dmPython`，未见 Go 驱动项目 | 仅作为官方仓库范围的旁证；不能证明其他渠道不存在许可 |
| 达梦域名上的[Go 驱动公开下载包](https://download.dameng.com/eco/adapter/resource/go/dm-go-driver.zip) | 2026-09-24 返回 196799 字节 ZIP，HTTP `Last-Modified: Wed, 09 Aug 2023 05:46:53 GMT`；外层含 `go/dm-go-driver.zip` | 可下载不等于获得再分发许可，且该包不是项目所用版本 |
| 上述官方 ZIP 内层的 `dm/VERSION` 和 `dm/a.go` | `VERSION` 为 `8.1.3.12`、日期 `2023.04.17`；`a.go` 顶部写有 `Copyright (c) 2000-2018, 达梦数据库有限公司. All rights reserved.` | 81 个内层条目中未发现许可或声明文件；不能据此推断新版本内容 |

外层 ZIP 的 SHA-256 为 `5d10cc42bbe8e4c7e6d9031d5306caff57b6da6e918868b5d7f70fc678063599`；内层 `go/dm-go-driver.zip` 的 SHA-256 为 `1c9a3ffd76b8650eb68e743f11465d8dea24f8397cc30d6d65e1d0e12b0781d2`。这些摘要用于识别本次下载的具体文件；公开地址将来可能更新。

复核下载包文件清单与版本的方法：

```sh
curl -fsSL https://download.dameng.com/eco/adapter/resource/go/dm-go-driver.zip -o /tmp/dm-go-driver-official.zip
unzip -l /tmp/dm-go-driver-official.zip
unzip -p /tmp/dm-go-driver-official.zip go/dm-go-driver.zip > /tmp/dm-go-driver-inner.zip
unzip -l /tmp/dm-go-driver-inner.zip
unzip -p /tmp/dm-go-driver-inner.zip dm/VERSION
unzip -p /tmp/dm-go-driver-inner.zip dm/a.go | head -6
```

本项目内置驱动的 [`p.go`](../../dpi_bridge/third_party/chunanyong_dm/p.go) 标记版本为 `8.1.4.170`、SVN 号 `43114`，而上面的官方包是 `8.1.3.12`。两者不能混为同一发布物。

## 本项目采用的 v1.8.22 镜像

- 本项目的 [`dpi_bridge/go.mod`](../../dpi_bridge/go.mod) 指向 `gitee.com/chunanyong/dm v1.8.22`，并用本地副本替换。该版本的 [Gitee 标签页](https://gitee.com/chunanyong/dm/tags)将 `v1.8.22` 对应到达梦 `8.1.4.170`。
- 从 [Go 官方模块代理的 v1.8.22 压缩包](https://proxy.golang.org/gitee.com/chunanyong/dm/@v/v1.8.22.zip)取得的 78 个条目中，没有 `LICENSE`、`LICENCE`、`COPYING` 或 `NOTICE` 文件；`README.md` 未写许可条款，`a.go` 保留了达梦版权及 `All rights reserved` 声明。该压缩包 SHA-256 为 `cf507684132bc6d7264026a640ee117ed2da49208870a6637ef3ea0e8e34e686`。
- [Gitee 仓库主页](https://gitee.com/chunanyong/dm)标注未指定许可证。[Go Packages 页面](https://pkg.go.dev/gitee.com/chunanyong/dm)对更新的 `v1.8.23` 显示 `License: None detected`，并注明这不是法律意见。这是未检测到公开许可的旁证，不能单独证明许可不存在。

复核 v1.8.22 文件清单的方法：

```sh
curl -fsSL https://proxy.golang.org/gitee.com/chunanyong/dm/@v/v1.8.22.zip -o /tmp/dm-go-v1.8.22.zip
unzip -Z1 /tmp/dm-go-v1.8.22.zip
unzip -p /tmp/dm-go-v1.8.22.zip 'gitee.com/chunanyong/dm@v1.8.22/README.md'
```

这不是只在开发环境使用的依赖：[`MANIFEST.in`](../../MANIFEST.in)把内置 Go 源码放进 sdist，[wheel 工作流](../../.github/workflows/build-wheels.yml)把 Go 桥接库编译为 dylib 并随 wheel 打包。因此需分别核对源码包和二进制包的公开分发条件。

## 尚未核实

1. 尚未取得包含 Go 驱动 8.1.4.170 的**达梦官方安装包**，因此不能确认该版本的包内是否另附许可、NOTICE 或安装协议。上述公开下载地址只提供 2023 年旧包；[官方 FAQ](https://eco.dameng.com/document/dm/zh-cn/faq/faq-go-new.html)也指出最新版本应从数据库安装包获取。
2. 尚未找到达梦官方针对 Go 驱动 8.1.4.170 作出的公开许可声明，明确覆盖第三方发布源码、修改源码及将其编译进 Python wheel 的情形。
3. 本核查未评估完整数据库安装包的最终用户协议对驱动分发的具体约束；取得对应安装包后应连同 `drivers/go` 目录和安装协议一起检查。

在这些范围澄清之前，本项目的 Mulan PSL v2 只可作为 dmPython 源码的许可依据，不能自动扩展到另行引入的达梦 Go 驱动。
