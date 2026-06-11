# Python

IoM Python SDK 是一个现代异步客户端库，用于与 Malice Network C2 框架进行交互。

> 完整文档和示例请参考：[Python SDK](https://github.com/chainreactors/IoM-python)

## 快速开始

### 安装

```bash
git clone https://github.com/chainreactors/IoM-python.git
cd IoM-python
pip install -e .
```

### 最小示例

```python
import asyncio
from IoM import MaliceClient
from IoM.proto.modulepb import Request

async def main():
    # 连接服务器
    client = MaliceClient.from_config_file("client.auth")

    async with client:
        # 获取会话
        await client.update_sessions()
        session_id = list(client.cached_sessions.keys())[0]
        session = await client.sessions.get_session(session_id)

        # 执行命令
        task = await session.whoami(Request(name="whoami"))
        result = await client.wait_task_finish(task)
        print(f"结果: {result.spite.response.output}")

asyncio.run(main())
```

## 核心概念

SDK 的设计完全对应 IoM 的架构概念，详细架构说明请参考 [核心概念](../../concept.md)。

### Client - 客户端

对应 [Client 架构](../../client/)，SDK 通过 `MaliceClient` 类与 Server 进行 gRPC 通讯。

**初始化方式：**

```python
# 从配置文件加载（推荐）
client = MaliceClient.from_config_file("client.auth")

# 手动配置
from IoM import ClientConfig
config = ClientConfig(
    host="127.0.0.1",
    port=5004,
    operator="admin",
    ca_certificate="...",
    certificate="...",
    private_key="..."
)
client = MaliceClient(config)
```

### Session - 会话

对应 [Session](../../client/console.md)，代表一个已连接的 Implant。

```python
# 更新会话列表
await client.update_sessions()

# 获取会话操作器
session = await client.sessions.get_session(session_id)
```

### Task - 任务

对应 [Task](../../client/console.md)，IoM 使用基于任务的异步执行模型。

```python
# 执行命令返回 Task
task = await session.whoami(Request(name="whoami"))

# 等待任务完成
result = await client.wait_task_finish(task)
```

### Spite - 通讯消息

对应 [核心概念](../../concept.md) 中的数据面说明，SDK 自动处理 Spite 的序列化和反序列化。

```python
# Request/Response 都是 Spite 的 body
from IoM.proto.modulepb import Request, ExecRequest
from IoM.proto.clientpb import Empty

# 通用请求
task = await session.whoami(Request(name="whoami"))

# 特定模块请求
task = await session.execute(ExecRequest(path="/bin/bash", args=["-c", "ls"]))

# 服务器操作
basic = await client.get_basic(Empty())
```

## 常用操作

### 系统信息

```python
# 获取当前用户
task = await session.whoami(Request(name="whoami"))
result = await client.wait_task_finish(task)

# 获取当前目录
task = await session.pwd(Request(name="pwd"))

# 列出进程
task = await session.ps(Request(name="ps"))
```

### 文件操作

```python
# 列出目录
task = await session.ls(Request(name="ls", input="/tmp"))

# 读取文件
task = await session.cat(Request(name="cat", input="/etc/passwd"))
```

### 命令执行

```python
from IoM.proto.modulepb import ExecRequest

# 执行系统命令
task = await session.execute(ExecRequest(
    path="/bin/bash",
    args=["-c", "ls -la"]
))
result = await client.wait_task_finish(task)
```

## 动态 API

SDK 自动转发所有 133 个 gRPC 方法，对应 [Server RPC](../../server/internals.md)。

```python
# 会话操作自动注入 session_id
task = await session.任意模块名(Request(...))

# 服务器操作
result = await client.任意RPC方法(对应的请求对象)

# 查看可用方法
print(dir(session))
print(dir(client))
```

## 错误处理

```python
from IoM.exceptions import MaliceError, ConnectionError

try:
    async with client:
        task = await session.whoami(Request(name="whoami"))
        result = await client.wait_task_finish(task)
except ConnectionError:
    print("服务器连接失败")
except MaliceError as e:
    print(f"执行错误: {e}")
```

## 最佳实践

1. **使用上下文管理器** - 确保连接正确关闭
   ```python
   async with client:
       # 执行操作
   ```

2. **复用 Client 实例** - 避免频繁创建连接
   ```python
   client = MaliceClient.from_config_file("client.auth")
   async with client:
       for i in range(100):
           await execute_command(session)
   ```

3. **检查会话状态** - 执行前确认会话存活
   ```python
   if session_info.is_alive:
       task = await session.whoami(...)
   ```

4. **设置超时** - 避免长时间等待
   ```python
   result = await client.wait_task_finish(task, timeout=30)
   ```

## 相关资源

- [Python SDK 源码](https://github.com/chainreactors/IoM-python)
- [示例代码](https://github.com/chainreactors/IoM-python/tree/master/examples)
- [核心概念](../../concept.md)
- [Proto 协议定义](https://github.com/chainreactors/proto)
