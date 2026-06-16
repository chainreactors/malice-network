# AI Agent 集成

IoM 通过 MCP（Model Context Protocol）和 Agent 命令体系，实现 LLM Agent 对 C2 的自动化操控。

## 概述

AI Agent 集成包含三个层面：

| 层面 | 机制 | 说明 |
|------|------|------|
| **接入层** | MCP Server | 将 IoM 能力暴露给外部 AI Agent |
| **控制层** | Agent 命令 | 管理 Agent 会话和 LLM Provider |
| **操作层** | Poison / Skill | 向 Agent 注入 prompt 执行任务 |

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

MCP 服务自动将 Client 命令树注册为标准 MCP 接口：

- **Tools**：每个 Client 命令被封装为可调用的 MCP Tool
- **Resources**：Session 列表、Listener 状态等暴露为可查询的 Resource

### 支持的客户端

任何兼容 MCP 协议的客户端均可接入：

- Claude Code / Claude Desktop
- 支持 Streamable HTTP 的 MCP Client：`http://127.0.0.1:5005/mcp`
- 旧版 SSE MCP Client：`http://127.0.0.1:5005/mcp/sse`

## Agent 命令

### chat

与 AI Agent 进行交互式对话：

```bash
chat                        # 开始交互式 Agent 对话
```

### poison

向 Agent 会话注入 prompt，让 Agent 自动执行操作：

```bash
poison <prompt>             # 注入 prompt 到当前 Agent 会话
```

??? example "Poison 使用示例"
    ```bash
    poison "Collect system information including OS version, users, and network config"
    ```
    Agent 会自动调用 IoM 命令完成信息收集。

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
    Client 内置 7 个核心 skill：`recon`、`creds`、`exfil`、`privesc`、`persist`、`portscan`、`cleanup`。

    详见 [Agent Skills 提案](../experiments/proposal-agent-skills.md)。

## AI Provider 配置

通过 `ai` 命令管理 LLM Provider：

```bash
ai                          # 查看当前 AI Provider 配置
```

Provider 配置通过 Server 端管理，支持 Claude、GPT 等 LLM。

## 相关文档

- [命令行系统](console.md) — MCP / LocalRPC 服务启动
- [Agent Skills 提案](../experiments/proposal-agent-skills.md) — Skill 系统设计
- [SDK 文档](../development/sdk/) — 编程方式接入 IoM
