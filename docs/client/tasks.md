# 任务查询

Client 在 Implant 上下文中通过 `tasks` 查看任务列表，通过 `fetch_task` 读取任务输出，通过 `tasks info` 查看任务请求摘要和原始请求。

## 概览

任务记录分为三类数据：

- `tasks`：任务元数据，例如 task id、类型、进度、创建时间、完成时间、请求摘要。
- request cache：Server 在创建任务时保存的完整请求 `Spite`，用于追溯当时发给 implant 的参数。
- task results：implant 回传的结果分片，用于 `fetch_task` 和结果导出。

默认列表不会返回 raw request 或 result spites。需要导出完整请求或结果时，使用 `tasks info` 的显式 flag。

## 用法

列出当前 session 的活跃任务：

```bash
tasks
```

列出当前 session 的全部历史任务：

```bash
tasks --all
```

查看单个任务的请求摘要：

```bash
tasks info 7
```

导出单个任务的完整 request 和结果：

```bash
tasks info 7 --raw --results --json
```

读取任务输出并按现有 renderer 展示：

```bash
fetch_task 7
```

## 字段

`tasks info` 返回的 `task` 元数据包含：

| 字段 | 含义 |
|------|------|
| `command_summary` | 一行命令摘要，例如 `exec whoami -- /all` |
| `request_summary` | 结构化 JSON 摘要，二进制字段只记录 size 和 sha256 |
| `request_size` | raw request protobuf 字节数 |
| `request_sha256` | raw request protobuf 的 SHA-256 |
| `has_request` | Server 是否保存了 request cache |

## 配置

普通 Task 的默认观察性超时为 60 秒。超过该时间会将 `timeout` 标记为 `true`，但不会中断 implant 上正在执行的任务。

request cache 不需要新增配置，使用现有 context 目录，路径由 Server 的 runtime context path 决定。

## 示例

```bash
tasks info 12 --json
```

输出中会包含任务元数据和请求摘要。若加 `--raw`，输出会额外包含完整的 `rawRequest`；若加 `--results`，输出会额外包含 `results` 数组。
