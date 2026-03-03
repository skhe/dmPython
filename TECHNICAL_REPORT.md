# 让达梦数据库跑在 Mac 上：dmPython-macOS 的诞生与开源实践

> 一个用 Go 桥接层替代专有 C 库、让达梦数据库 Python 驱动原生运行在 macOS ARM64 上的技术故事。

## 1. 背景：一个被遗忘的平台

### 达梦数据库与 dmPython

达梦数据库（DM8）是国产关系型数据库的代表产品，广泛应用于政企领域。[dmPython](https://github.com/DamengDB/dmPython) 是其官方 Python 驱动，遵循 Python DB-API 2.0 规范，底层是约 19000 行 C 代码的扩展模块，通过 DPI（Dameng Programming Interface）库与数据库通信。

### 问题在哪？

dmPython 的 C 代码通过动态链接调用 `libdmdpi.so`（Linux）或 `dmdpi.dll`（Windows）——这是达梦官方提供的**闭源专有库**。

**macOS 没有对应的 `libdmdpi.dylib`。**

这意味着：
- macOS 上无法编译 dmPython
- macOS 上无法 `pip install` 任何 dmPython wheel
- 使用 Mac 做开发的工程师，如果项目用了达梦数据库，只能在 Linux 虚拟机或远程服务器上调试

对于日益增长的 macOS（尤其是 Apple Silicon）开发者群体来说，这是一个实际的痛点。

## 2. 解题思路：把“缺库问题”拆成“兼容层工程”

### 先定义约束，再选方案

这个项目不是简单“让代码跑起来”，而是一个多约束优化问题。核心约束有四条：

1. **兼容性约束**：上层 `dmPython` C 扩展（约 19K 行）尽量不改，避免 fork 长期分叉。
2. **合规约束**：达梦 DPI 头文件属于专有资产，仓库不能直接公开分发。
3. **交付约束**：目标是最终用户一条 `pip install` 即可安装，而不是“先装一堆前置环境”。
4. **平台约束**：必须原生支持 macOS ARM64，不依赖 Linux 虚拟机或 Rosetta 绕行方案。

基于这些约束，架构上可选路径其实不多：

- 路线 A：重写 dmPython C 扩展。技术可行，但维护成本最高，与上游同步最困难。
- 路线 B：直接在 Python 层改驱动协议栈。改动面过大，风险从 ABI 层转移到行为层。
- 路线 C：保留 C 扩展，替换其依赖的 `libdmdpi`。改动边界最清晰，最符合“最小侵入”原则。

最终选择路线 C：把问题收敛为“实现一个与 DPI ABI 兼容的 `libdmdpi.dylib`”。

### 方案核心：以 DPI ABI 为边界的 Go 桥接层

达梦 Go 驱动 [dm](https://gitee.com/chunanyong/dm) 是纯 Go 网络协议实现，天然跨平台。项目将其作为底层能力，通过 Go `-buildmode=c-shared` 暴露 C 符号，向上伪装成 `libdmdpi.dylib`：

```
Python App
  ↓
dmPython (官方 C Extension，零改动复用)
  ↓  调用 DPI C API
libdmdpi.dylib (Go bridge, c-shared)
  ↓  调用 Go dm driver
DM wire protocol (TCP)
  ↓
DM8 Server
```

这种分层的关键价值是“把变化锁在桥接层”：上游 C 代码保持稳定，平台适配与协议实现由 Go 层承担。

### 关键实现机制（项目内真实落地）

`dpi_bridge/` 按功能拆分为连接、语句、取数、绑定、诊断、LOB、元数据等模块，核心机制如下：

- **句柄模型对齐**：`handle.go` 维护 `uintptr -> Go 对象` 的句柄池，对外表现为 C 可识别的 `void*` 句柄，保证 dmPython 原有句柄生命周期可复用。
- **语句执行语义兼容**：`dpi_stmt.go` 负责 `prepare/exec/attr`，并通过缓存结果行的方式支持 `dpi_row_count` 等依赖“已知行数”的调用路径。
- **类型与内存布局转换**：`dpi_fetch.go` 把 Go 值（如 `string`、`time.Time`、数值）写入 DPI 约定的 C 结构体（如 `dpi_timestamp_t`、`dpi_numeric_t`），同时维护 `indPtr/actLenPtr` 等长度与空值信息。
- **错误诊断回传**：`dpi_diag.go` 将 Go 侧错误统一映射为 DPI 诊断信息，保证上层仍通过 `dpi_get_diag_rec` 等标准接口拿到错误详情。

### 兼容边界与工程取舍

该桥接层优先覆盖 dmPython 主路径（连接、SQL 执行、结果读取、事务、常见元数据与 LOB）；对象/BFILE 等复杂特性当前以“显式返回未支持错误”为策略，而不是静默行为偏差。  
这种取舍让系统在“可用性优先”与“行为可解释性”之间取得平衡，也为后续增量补齐能力留下明确路线。

## 3. CI/CD 实现：从可构建到可发布的自动化链路

### 触发策略与发布闸门

`.github/workflows/build-wheels.yml` 采用单文件双阶段设计：

- `pull_request` 到 `main`：执行完整构建与校验，但不发布。
- `push tags: v*`：先构建，再进入 release 阶段发布。
- `workflow_dispatch`：支持人工重跑与应急发布。

发布闸门由两个条件共同控制：`release` job 依赖 `build` 成功（`needs: build`），且仅在 tag 引用下触发（`if: startsWith(github.ref, 'refs/tags/v')`）。

### Build 阶段：矩阵并行产出 wheel

`build` job 在 `macos-14`（ARM64 runner）上执行，Python 版本矩阵为 `3.9~3.13`，每个版本独立产出 wheel。关键步骤如下：

1. `actions/checkout` 拉取源码。
2. `setup-go@v5` 安装 Go 1.21，并通过 `cache-dependency-path: dpi_bridge/go.sum` 命中子目录依赖缓存。
3. `setup-python@v5` 安装矩阵 Python。
4. 从 `DPI_HEADERS_TAR_B64` secret 解码专有头文件到 `dpi_include/`。
5. 编译 Go 桥接库：`go build -buildmode=c-shared -o libdmdpi.dylib`，并通过 `install_name_tool -id @rpath/libdmdpi.dylib` 修正动态库标识。
6. 构建 wheel：设置 `MACOSX_DEPLOYMENT_TARGET=14.0` 与 `_PYTHON_HOST_PLATFORM=macosx-14.0-arm64`，再执行 `DMPYTHON_SKIP_GO_BUILD=1 python -m build --wheel`（避免重复编译 Go）。
7. `delocate-wheel` 将 `libdmdpi.dylib` 内嵌进 wheel，形成可分发产物。
8. 在临时虚拟环境安装 wheel 并执行 `import dmPython` 作为最小可用性验证。
9. 通过 `actions/upload-artifact` 上传每个 Python 版本的 wheel。

### Release 阶段：聚合产物并发布

`release` job 下载前序矩阵产物（`pattern: wheel-*`, `merge-multiple: true`），随后调用：

```bash
gh release create "$TAG_NAME" --generate-notes dist_fixed/*.whl
```

实现“一次 tag -> 自动生成 GitHub Release + 附带全部 wheel”。

### CI 里踩过的坑与固定方案

| 问题 | 原因 | 固化方案 |
|------|------|----------|
| 专有头文件不能入库 | 合规要求 | Secret（Base64 压缩包）注入，流水线临时解码 |
| wheel 标签不稳定 | 默认平台推断可能混入非目标架构 | 显式设置 `MACOSX_DEPLOYMENT_TARGET` 与 `_PYTHON_HOST_PLATFORM` |
| Go 缓存未命中 | `go.sum` 位于子目录 | 指定 `cache-dependency-path: dpi_bridge/go.sum` |
| release 阶段找不到仓库上下文 | `gh release` 需要 git 元数据 | release job 重新 `checkout` |

### 产物形态与当前覆盖边界

tag 发布后会生成 5 个 ARM64 wheel（对应 Python 3.9~3.13），命名形态如下：

```
dmPython_macOS-2.5.30-cp39-cp39-macosx_14_0_arm64.whl
dmPython_macOS-2.5.30-cp310-cp310-macosx_14_0_arm64.whl
dmPython_macOS-2.5.30-cp311-cp311-macosx_14_0_arm64.whl
dmPython_macOS-2.5.30-cp312-cp312-macosx_14_0_arm64.whl
dmPython_macOS-2.5.30-cp313-cp313-macosx_14_0_arm64.whl
```

当前 CI 的验证粒度是“构建成功 + 可安装 + 可导入”。它能有效拦截打包与链接问题，但还未覆盖真实数据库集成测试；这也是下一阶段最值得补强的质量门禁。

## 4. 开源项目规范化

将项目从"能用"提升到"规范的开源项目"，一次性完成了以下工作：

### 4.1 元数据补全

- **GitHub 仓库**：设置 description、homepage、topics（dameng, dm8, database, python, db-api, macos, driver）
- **pyproject.toml**：添加 authors、project.urls、Python 3.9-3.13 classifiers
- **setup.py**：同步添加 author、url、project_urls

### 4.2 标准开源文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `LICENSE` | 修复 | 拼写错误（KIDN→KIND）、更新年份（2017-2026）、去掉方括号 |
| `README.md` | 重写 | 纯英文 + CI/License/Python/Platform badges |
| `README_zh.md` | 新建 | 中文文档迁移，许可证从错误的 "PSF License" 改为 "Mulan PSL v2" |
| `CHANGELOG.md` | 重命名+格式化 | `ChangeLogs.md` → `CHANGELOG.md`，转为 [Keep a Changelog](https://keepachangelog.com/) 格式 |
| `CONTRIBUTING.md` | 新建 | 开发环境、构建命令、PR 流程、代码风格 |
| `.gitignore` | 补充 | `.DS_Store`、`*.pyc`、`.pytest_cache/`、`dist_ci/` |

### 4.3 仓库卫生

- 清理已合并的 `improve-readme` 分支（本地 + 远程）
- 检查版本历史：无专有头文件、二进制文件或秘钥泄漏
- 确认 upstream remote 配置正确，便于未来同步上游更新

## 5. 技术亮点与经验总结

### 5.1 Go 作为"万能胶水"的价值

Go 语言在这个项目中展现了独特价值：

- **纯 Go 网络协议实现**：达梦 Go 驱动不依赖 C 库，天然跨平台
- **`c-shared` 编译模式**：Go 代码可以编译为 C 兼容的动态库，暴露标准 C 函数符号
- **CGo 双向互操作**：Go 函数可以接收 C 指针参数，也可以回写 C 结构体内存

这使得"用 Go 重写一个 C 库的实现"成为可行且高效的方案。

### 5.2 "不改上游代码"的 Fork 策略

我们刻意保持 dmPython 的 19000 行 C 代码**零修改**。好处是：

- 上游发布新版本时，可以直接 `git merge upstream/main`
- 不需要理解和维护 C 代码的内部逻辑
- Bug 修复和新功能自动继承

代价是 Go 桥接层必须**精确实现** DPI 头文件声明的所有函数签名和内存布局，没有偷懒的余地。

### 5.3 Wheel 打包的平台标签陷阱

macOS wheel 的平台标签直接影响 `pip install` 的兼容性判断。我们遇到的问题是：

```
# 错误：混入非目标架构标签，pip 在 ARM64 上可能拒绝安装
dmPython_macOS-2.5.30-cp312-cp312-macosx_10_9_x86_64.macosx_14_0_arm64.whl

# 正确：纯 ARM64 标签
dmPython_macOS-2.5.30-cp312-cp312-macosx_14_0_arm64.whl
```

解决方法是在构建时通过环境变量明确指定目标平台：

```bash
MACOSX_DEPLOYMENT_TARGET=14.0 _PYTHON_HOST_PLATFORM=macosx-14.0-arm64 python -m build --wheel
```

### 5.4 Secrets 管理：专有头文件的处理

DPI 头文件（`DPI.h`、`DPItypes.h` 等）是达梦的专有文件，不能公开分发。我们的方案：

1. `.gitignore` 排除 `dpi_include/` 目录
2. 将头文件 tar.gz + Base64 编码后存入 GitHub Secrets (`DPI_HEADERS_TAR_B64`)
3. CI 中解码到临时目录，构建完成后自动清理

这样既保护了专有文件，又实现了完全自动化的 CI 构建。

## 6. 项目数据

| 指标 | 数值 |
|------|------|
| C 源码（复用上游） | ~19,000 行 |
| Go 桥接层（新写） | ~4,700 行 |
| 支持 Python 版本 | 3.9, 3.10, 3.11, 3.12, 3.13 |
| 支持平台 | macOS ARM64 (Apple Silicon) |
| CI 构建时间 | ~1 分钟（5 版本并行） |
| Wheel 大小 | ~8 MB（内嵌 libdmdpi.dylib） |
| 依赖 | 零运行时依赖 |

## 7. 产生的价值

1. **填补平台空白**：macOS ARM64 开发者首次获得可直接 `pip install` 的达梦 Python 驱动
2. **开发体验提升**：不再需要 Linux 虚拟机或远程服务器来调试涉及达梦数据库的 Python 代码
3. **Go 桥接模式的验证**：证明了"用 Go 重实现 C 库接口"这一技术路线的可行性，可推广到其他缺乏跨平台支持的数据库驱动
4. **开源最佳实践**：从 CI 自动构建、Release 自动发布，到标准的开源文件结构，提供了一个小型开源项目的完整范本
5. **与上游共存**：零修改 fork 策略确保可以持续跟进上游更新，不会分裂社区

## 8. 未来方向

- [ ] 发布到 PyPI，支持 `pip install dmPython-macOS`
- [ ] 补充自动化测试套件（连接 Docker 达梦实例）
- [ ] 探索 Linux ARM64 支持（同样缺少官方 `libdmdpi.so`）
- [ ] 性能基准测试：Go 桥接层 vs 原生 DPI 库的开销对比
- [ ] 向上游提议合并 Go 桥接方案，惠及更多平台

---

**项目地址**：https://github.com/skhe/dmPython

**上游项目**：https://github.com/DamengDB/dmPython

**许可证**：Mulan PSL v2
