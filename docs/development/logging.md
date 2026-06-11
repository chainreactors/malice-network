# 日志格式规范

## 概述

Server debug 日志使用面向人工排查的短文本格式，目标是快速看清时间、级别、模块和关键字段。

## 控制台格式

```text
[05.13 21:36:28] DBG crypto.wrap - encryption_configs_count=2
[05.13 21:36:28] DBG crypto.peek - raw_first_9_bytes=761bc3124f3ab80c51
[05.13 21:36:28] DBG pipeline.tcp - accept pipeline=tcp remote=192.168.239.182:54774
[05.13 21:36:28] DBG connection - close session=2165f494174b03af7d10a0e54d70e9b8 raw=801492628 reason="EOF"
```

字段含义：

| 字段 | 说明 |
|------|------|
| `[05.13 21:36:28]` | 本地时间，精确到秒 |
| `DBG` | 日志级别，另有 `INF`、`WRN`、`ERR`、`IMP` |
| `crypto.wrap` | 模块名，使用点号表达子模块 |
| `-` | 模块和内容的分隔符 |
| `key=value` | 稳定字段，便于 grep 和脚本处理 |

## 级别映射

| 级别 | 原始含义 | 使用场景 |
|------|----------|----------|
| `DBG` | debug | 调试细节 |
| `INF` | info | 普通运行事件 |
| `WRN` | warn | 可恢复的异常或风险 |
| `ERR` | error | 失败或错误 |
| `IMP` | important | 重要状态变化 |

## 字段保留原则

日志优化只调整展示顺序和命名，不减少现有排查字段。

- 连接日志保留完整 `session`、`raw`、`remote`、`reason`
- 加密日志保留 `encryption[index].type` 和 `encryption[index].key`
- 协议探测日志保留原始 9 字节、解密结果、cryptor index 和错误信息
- 大对象日志可继续按大小降级为 `bytes=<n>`，避免控制台输出过长

## 编写规则

推荐格式：

```go
logs.Log.Debugf("module.name - action field=%s count=%d", value, count)
```

避免格式：

```go
logs.Log.Debugf("debug ModuleName: field=%s", value)
logs.Log.Debugf("module did something %s", value)
```

## 启用方式

```bash
./malice-server --debug
```

`--debug` 会启用 `DBG` 级别，并使用秒级时间戳。普通模式仍保持较少输出。

## 延迟分析

秒级时间适合人工排查大多数 beacon、pipeline、task 时间线问题：

```bash
grep "event.task" server.log
grep "receive_spite_request" server.log
grep "pipeline.tcp - accept" server.log
```

如果需要毫秒级性能分析，应单独加入耗时字段，例如：

```text
[05.13 21:36:28] DBG download - chunk_retry chunk=4 retry=2 elapsed_ms=128
```

## 相关文件

| 文件 | 说明 |
|------|------|
| `server/cmd/server/log_config.go` | Server debug 日志格式配置 |
| `server/cmd/server/options.go` | `--debug` 启用入口 |
| `server/internal/core/pipeline.go` | 加密封装日志 |
| `server/internal/stream/peekconn.go` | 协议探测日志 |
| `server/internal/core/connection.go` | 连接生命周期日志 |
