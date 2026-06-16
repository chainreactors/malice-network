# AI 集成指南

本文档介绍如何通过 MCP 和 SDK 将 IoM 能力集成到 AI Agent 中。

## 集成方式

IoM 提供三种 AI 集成方式：

| 方式 | 机制 | 适用场景 |
|------|------|---------|
| **Client MCP** | Client 内置 MCP Server | Claude Desktop、MCP 兼容 Agent |
| **SDK as Tool** | Python/TypeScript SDK 封装为 AI Tool | LangChain、OpenAI Function、自定义 Agent |
| **FFI** | Malefic-Win-Kit DLL 跨语言调用 | 自定义 Implant 开发、多语言集成 |

## Client MCP

Client 内置 MCP (Model Context Protocol) Server，可以让 AI 通过 MCP 协议完整使用 IoM 的全部功能。

### 启动

```bash
./iom --mcp 127.0.0.1:4999
```

也可以在 Client settings 中启用 `mcp_enable` / `mcp_addr`。命令行参数优先于配置文件。

### MCP Client 配置

优先使用 Streamable HTTP 接入：

```json
{
  "mcpServers": {
    "IoM": {
      "type": "http",
      "url": "http://127.0.0.1:4999/mcp"
    }
  }
}
```

旧版只支持 SSE 的 MCP Client 仍可使用 legacy endpoint：

```json
{
  "mcpServers": {
    "IoM": {
      "type": "sse",
      "url": "http://127.0.0.1:4999/mcp/sse"
    }
  }
}
```

### 使用场景

- 智能渗透测试：AI 自主分析目标并执行渗透测试
- 自动化响应：结合 AI 决策和 IoM 执行能力
- 交互式操作：通过自然语言控制 C2 框架

详细的 Agent 命令（chat / poison / tapping / skill）见 [AI Agent 集成](../client/agent.md)。

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

## FFI 集成

Malefic-Win-Kit 将 Windows 攻击能力封装为标准 C ABI 的 DLL，支持任意语言通过 FFI 调用。

### 主要能力

- **PE 执行**: `RunPE`, `PELoader`, `InlinePE`, `RunSacrifice`
- **反射加载**: `ReflectiveLoader`, `MaleficLoadLibrary`
- **代码注入**: `RunShellcode`, `ApcLoaderInline`
- **高级功能**: `BOF`, `CSharpUtils`, `PowershellUtils`
- **EDR 绕过**: `EDRBypassUtils_REFRESH_DLL`, `EDRBypassUtils_BLOCK_DLL`

### 示例（Python）

```python
import ctypes

dll = ctypes.CDLL("malefic_win_kit.dll")

class RawString(ctypes.Structure):
    _fields_ = [
        ("data", ctypes.POINTER(ctypes.c_uint8)),
        ("len", ctypes.c_size_t),
        ("capacity", ctypes.c_size_t)
    ]

dll.RunPE.restype = RawString
result = dll.RunPE(
    b"C:\\Windows\\System32\\notepad.exe",
    (ctypes.c_uint8 * len(pe_data))(*pe_data),
    len(pe_data), b"--help", 0, True, False
)
output = ctypes.string_at(result.data, result.len)
dll.SafeFreePipeData(result.data)
```

### 支持的语言

| 语言 | 示例 |
|------|------|
| C | [c/](https://github.com/chainreactors/malefic/tree/master/examples/ffi/c) |
| Go | [go/](https://github.com/chainreactors/malefic/tree/master/examples/ffi/go) |
| Rust | [rust/](https://github.com/chainreactors/malefic/tree/master/examples/ffi/rust) |
| Python | [python/](https://github.com/chainreactors/malefic/tree/master/examples/ffi/python) |
| C# | [csharp/](https://github.com/chainreactors/malefic/tree/master/examples/ffi/csharp) |

## 相关文档

- [AI Agent 集成](../client/agent.md) — Client 侧 Agent 命令详解
- [SDK 文档](sdk/) — 多语言 SDK 使用指南
- [命令行系统](../client/console.md) — MCP/LocalRPC 启动方式
