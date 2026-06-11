# Server 日志时间戳优化

## 概述

Server debug 日志现在使用秒级时间戳和短级别名称，减少控制台噪声，同时保留原有排查字段。

## 输出示例

```text
[05.13 21:36:28] DBG crypto.wrap - encryption_configs_count=2
[05.13 21:36:28] DBG crypto.peek - parser=auto cryptors=2
[05.13 21:36:28] DBG event.task - Task 9 task_finish
[05.13 21:36:28] ERR connection - close session=2165f494174b03af7d10a0e54d70e9b8 raw=801492628 reason="EOF"
```

## 设计选择

- 时间精确到秒，便于阅读和对齐控制台输出
- 级别使用 `DBG/INF/WRN/ERR/IMP`，替代 `[debug]/[+]/[-]`
- 模块使用 `module.submodule - action fields`
- 字段使用 `key=value`，方便 grep 和脚本分析

## 使用方式

```bash
./malice-server --debug
```

`--debug` 会启用 debug level 并应用新的 server 日志格式。

## 延迟排查

常用筛选：

```bash
grep "receive_spite_request" server.log
grep "event.task" server.log
grep "pipeline.tcp - accept" server.log
```

如果某个具体路径需要毫秒级分析，优先在业务日志里加入 `elapsed_ms=<n>` 字段，而不是把所有日志时间戳改回毫秒级。

## 相关文件

| 文件 | 说明 |
|------|------|
| `server/cmd/server/log_config.go` | 日志格式配置 |
| `server/cmd/server/options.go` | debug 启用入口 |
| `docs/development/logging.md` | 完整日志格式规范 |
