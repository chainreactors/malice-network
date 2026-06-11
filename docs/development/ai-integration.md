# AI 集成指南

本文档介绍如何通过 MCP 和 SDK 将 IoM 能力集成到 AI Agent 中。

## 集成方式

IoM 当前仓库内提供两类一线 AI 集成方式：

| 方式 | 机制 | 适用场景 |
|------|------|---------|
| **Client MCP** | Client 内置 MCP Server | Claude Desktop、MCP 兼容 Agent |
| **SDK as Tool** | Python/TypeScript SDK 封装为 AI Tool | LangChain、OpenAI Function、自定义 Agent |

## Client MCP

Client 内置 MCP (Model Context Protocol) Server，可以让 AI 通过 MCP 协议完整使用 IoM 的全部功能。

### 启动

```bash
./iom --auth admin.auth --daemon --mcp 127.0.0.1:5005
```

也可以在 Client settings 中启用 `mcp_enable` / `mcp_addr`。命令行参数优先于配置文件。

### Claude Desktop 配置

```json
{
  "mcpServers": {
    "IoM": {
      "type": "sse",
      "url": "http://127.0.0.1:5005/mcp/sse"
    }
  }
}
```

### 使用场景

- 智能渗透测试：AI 自主分析目标并执行渗透测试
- 自动化响应：结合 AI 决策和 IoM 执行能力
- 交互式操作：通过自然语言控制 C2 框架

详细的 Agent 命令（chat / tapping / skill / schema / tool_call）见 [AI Agent 集成](../client/agent.md)。

## SDK as AI Tool

通过 Python/TypeScript SDK，可以将 IoM 的 RPC 封装为 AI Tool。

### 作为 LangChain Tool

```python
from langchain.tools import Tool
from IoM import MaliceClient
from IoM.proto.modulepb import Request

class IoMTool:
    def __init__(self, auth_file: str):
        self.client = MaliceClient.from_config_file(auth_file)
        self.session = None

    async def setup(self):
        await self.client.update_sessions()
        session_id = list(self.client.cached_sessions.keys())[0]
        self.session = await self.client.sessions.get_session(session_id)

    async def execute_command(self, command: str) -> str:
        task = await self.session.execute(Request(name="execute", input=command))
        result = await self.client.wait_task_finish(task)
        return result.spite.response.output

iom_tool = IoMTool("client.auth")
tool = Tool(
    name="IoM_Execute",
    func=lambda cmd: asyncio.run(iom_tool.execute_command(cmd)),
    description="Execute command on remote session"
)
```

### 作为 OpenAI Function

```python
functions = [{
    "name": "execute_command",
    "description": "Execute system command on remote target",
    "parameters": {
        "type": "object",
        "properties": {
            "command": {"type": "string", "description": "Command to execute"}
        },
        "required": ["command"]
    }
}]
```

### 作为 MCP Server（Python）

```python
from mcp.server import Server
from IoM import MaliceClient

app = Server("iom-mcp-server")

@app.call_tool()
async def execute_iom_command(command: str) -> str:
    client = MaliceClient.from_config_file("client.auth")
    async with client:
        await client.update_sessions()
        session_id = list(client.cached_sessions.keys())[0]
        session = await client.sessions.get_session(session_id)
        task = await session.execute(Request(name="execute", input=command))
        result = await client.wait_task_finish(task)
        return result.spite.response.output
```

## 外部 FFI 集成

FFI 示例属于外部 implant 项目能力，不在当前 malice-network 仓库内实现。需要跨语言调用 implant 能力时，应以对应外部项目的 FFI 文档和示例为准；本仓库侧主要负责 Client MCP、LocalRPC/SDK 与 Server RPC 能力暴露。

## 相关文档

- [AI Agent 集成](../client/agent.md) — Client 侧 Agent 命令详解
- [SDK 文档](sdk/) — 多语言 SDK 使用指南
- [命令行系统](../client/console.md) — MCP/LocalRPC 启动方式
