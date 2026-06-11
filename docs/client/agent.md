# Agent 集成

IoM 通过 MCP（Model Context Protocol）和 Agent 命令体系，实现 LLM Agent 对 C2 的自动化操控。

## 概述

AI Agent 集成包含三个层面：

| 层面 | 机制 | 说明 |
|------|------|------|
| **接入层** | MCP Server | 将 IoM 能力暴露给外部 AI Agent |
| **控制层** | Agent 命令 | 管理 Agent 会话和 LLM Provider |
| **操作层** | chat / skill / schema / tool_call | 对 Agent 发起自然语言任务、执行 Skill、查看和调用桥接工具 |

## MCP 服务

MCP（Model Context Protocol）是连接 IoM 与外部 AI Agent 的标准协议。

### 启动 MCP

```bash
./iom --auth admin.auth --mcp 127.0.0.1:5005
```

也可以与 daemon 模式组合使用：

```bash
./iom --auth admin.auth --daemon --mcp 127.0.0.1:5005
```

### 暴露内容

MCP 服务会注册通用工具和 Client 命令接口：

- **Tools** ：`execute_command`、`execute_lua`、`search_commands`、`get_history`，以及从 Cobra 命令树生成的命令 Tool
- **Resources** ：从命令树生成的只读 Resource，用于查询 session、listener、artifact 等常用状态

### 支持的客户端

任何兼容 MCP 协议的客户端均可接入：

- Claude Code / Claude Desktop
- 任何支持 SSE 协议的 MCP Client

## Agent 命令

### chat

与 AI Agent 进行交互式对话：

```bash
chat "list current directory"     # 向当前 Agent 会话发送自然语言任务
```

对自托管 Agent 会话，可通过参数临时覆盖本地 `config ai` 中的模型或 Provider 偏好：

```bash
chat -m gpt-4o "do a network scan"
chat -p deepseek "enumerate running processes"
```

### tapping

监听 Agent 的交互过程，实时查看 Agent 的思考和工具调用：

```bash
tapping                     # 监听当前 Agent 会话
```

输出格式包括：

- `◀ REQ` — 发送给 LLM 的请求
- `▶ RSP` — LLM 的响应（包含 text / ⚡tool 调用）
- `↩ result` — 工具执行结果

### skill

使用预定义的 prompt 模板执行标准化操作：

```bash
skill list                  # 列出可用的 skill
skill recon                 # 执行侦查 skill
skill creds "AWS keys"      # 执行凭证收集，附带参数
```

!!! tip "内置 Skill"
    Skill 会按 `./skills/`、`~/.config/malice/skills/`、内置资源的优先级查找。同名 skill 以本地文件优先。

    当前内置 skill 包括：`recon`、`creds`、`exfil`、`privesc`、`persist`、`portscan`、`cleanup`。

    详见 [Agent Skills 提案](../experiments/proposal-agent-skills.md)。

### schema

查看当前 Agent bridge 已观察到的工具 schema：

```bash
schema
```

这通常用于确认桥接 Agent 暴露了哪些工具，例如 Bash、Read、Write、WebFetch、Grep 等。

### tool_call

向当前 Agent 会话注入一次工具调用：

```bash
tool_call Bash '{"command":"id"}'
tool_call Read '{"file_path":"/etc/passwd"}'
```

工具名和参数需要匹配 `schema` 返回的定义。

## AI 偏好配置

通过 `config ai` 管理 Client 本地 AI 偏好：

```bash
config ai                   # 查看当前 AI 偏好
config ai enable --provider openai --model gpt-5.4
```

Agent `chat` / `skill` 会把本地 `config ai` 中的 provider/model 作为偏好传给 Server；实际 endpoint、api key、proxy 由 Server 端 `server.llm` 解析。旧的 `ask` / `analyze` 命令仍可使用本地 endpoint/api key。

## 相关文档

- [命令行系统](console.md) — MCP / LocalRPC 服务启动
- [Agent Skills 提案](../experiments/proposal-agent-skills.md) — Skill 系统设计
- [SDK 文档](../development/sdk/) — 编程方式接入 IoM
