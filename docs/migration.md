# 从官方 dmPython 迁移到 macOS 版本

## 目标读者

- 当前使用官方 `dmPython`，希望在 macOS ARM64 运行
- 代码层面尽量保持 DB-API 使用方式不变

## 兼容性结论

在多数业务代码里，迁移只需要替换安装来源，`import dmPython` 与 `connect()/cursor()/execute()` 调用方式保持一致。

## 迁移步骤

1. 卸载旧包并安装 macOS wheel

```bash
pip uninstall -y dmPython dmPython-macOS
pip install dmPython_macOS-<version>-cp312-cp312-macosx_14_0_arm64.whl
```

2. 验证运行时版本

```bash
python - <<'PY'
import dmPython
print(dmPython.version)
print(dmPython.buildtime)
PY
```

3. 回归关键路径

- 建连与断连
- 事务提交/回滚
- 批量写入
- LOB 读写
- 存储过程调用

## 差异与注意事项

- 平台定位：本 fork 的发布目标是 macOS ARM64。
- 底层实现：使用 Go DPI bridge 替代上游依赖的专有 `libdmdpi`。
- 连接池：驱动本身不提供独立 `SessionPool` 对象，建议应用层连接池（见 [连接池示例](examples/connection-pool.md)）。
- 未支持接口：`Cursor.parse()`、`Cursor.arrayvar()`、`Cursor.bindnames()` 当前返回 `NotSupportedError`。

## 常见迁移问题

- `ImportError`：确认 wheel Python ABI 与本地 Python 版本匹配。
- 建连失败：确认 `server/port/user/password`、网络连通性、数据库监听配置。
- 字符编码问题：检查 `local_code`、`lang_id` 参数设置。
