# Server 内部机制

本文档说明 Server 的核心内部机制，包括 RPC 通信、数据持久化和辅助服务。

## RPC 通信机制

### 三类 RPC 服务

Server 通过 gRPC 暴露三类服务，分别面向不同的调用方：

| 服务 | 调用方 | 职责 | Proto 定义 |
|------|--------|------|-----------|
| **MaliceRPC** | Client | Session/Task/Build/Profile/Module 等全部操作 | `services/clientrpc/service.proto` |
| **RootRPC** | 本地管理 | Operator 管理、Listener 增删（仅 localhost） | `client/rootpb/root.proto` |
| **ListenerRPC** | Listener | Pipeline 注册、SpiteStream/JobStream 双向流 | `services/listenerrpc/service.proto` |

### mTLS 认证

所有 RPC 连接均使用 mTLS（双向 TLS）认证：

1. Server 启动时生成 Root CA 证书
2. 每个 Operator（Client/Listener）获得由 CA 签发的客户端证书
3. 连接时从证书中提取 CN（Common Name）和指纹进行身份验证
4. `.auth` 文件包含客户端证书 + CA 证书 + 连接信息

### 授权机制

认证通过后，Server 通过 `AuthzRule` 表控制不同角色可调用的 RPC 方法：

| 角色 | 说明 |
|------|------|
| **admin** | 完全权限，localhost 自动识别 |
| **operator** | 标准操作权限 |
| **listener** | 仅 ListenerRPC 权限 |

`AuthzRule` 支持方法级的 allow/deny 规则，支持通配符匹配（如 `/clientrpc.MaliceRPC/*`）。

### 拦截器链

每个 RPC 调用经过三层拦截器：

```
请求 → logInterceptor → auditInterceptor → authInterceptor → Handler
```

- **logInterceptor** ：记录所有 RPC 调用到日志
- **auditInterceptor** ：按 Session 记录请求/响应详情（受 `audit` 级别控制）
- **authInterceptor** ：执行 mTLS 认证 + RBAC 授权

### Listener 双向流

Listener 与 Server 之间通过两条 gRPC Stream 通信：

| Stream | 方向 | 用途 |
|--------|------|------|
| **JobStream** | Server → Listener | Pipeline 启停控制（Start/Stop/Register） |
| **SpiteStream** | 双向 | Implant 命令下发与结果回传 |

## 数据持久化机制

### 数据库

Server 支持两种数据库后端：

| 后端 | 适用场景 | 说明 |
|------|---------|------|
| **SQLite** | 默认，单节点部署 | 文件数据库，无需额外依赖 |
| **PostgreSQL** | 多节点、高并发 | 需要独立数据库服务 |

基于 GORM ORM，启动时自动执行数据库迁移。

### 数据模型

| 模型 | 持久化内容 |
|------|-----------|
| **Session** | Implant 会话记录（ID、Pipeline、Listener、存活状态、运行时上下文） |
| **Task** | 任务记录（Session 关联、类型、进度、执行时间） |
| **Profile** | 构建配置快照（名称、参数、Pipeline 绑定） |
| **Pipeline** | Pipeline 注册信息（类型、端口、加密、TLS） |
| **Artifact** | 构建产物记录（名称、Target、Source、状态、路径） |
| **Operator** | 操作员信息（名称、指纹、角色、证书、撤销状态） |
| **Certificate** | TLS 证书存储 |
| **AuthzRule** | 角色授权规则 |
| **Context** | 操作上下文数据（截图、凭证、下载等） |
| **WebsiteContent** | Website Pipeline 托管内容 |

### Context 目录

除数据库外，Server 使用文件系统存储大体积运行时数据：

```
contexts/<sessionID>/
├── tasks/          # 任务结果（protobuf Spite 序列化）
├── requests/       # 请求记录（审计级别 > 1 时）
├── screenshots/    # 截图
├── downloads/      # 下载文件
├── keylogger/      # 键盘记录
├── media/          # 媒体文件
└── cache/          # 缓存数据
```

### 状态恢复

Server 重启时自动恢复运行时状态：

1. **Session 恢复** ：从数据库加载存活的 Session，重建内存对象
2. **Pipeline 恢复** ：从数据库加载已启用的 Pipeline，通过 JobCtrl 重新启动
3. **Website 恢复** ：重新启动已注册的 Website Pipeline

## 审计机制

### 审计级别

通过 `server.audit` 配置控制审计详细程度：

| 级别 | 记录内容 |
|------|---------|
| 1 | 任务执行结果（Task + Spite） |
| > 1 | 额外记录原始请求（requests 目录） |

### 审计数据来源

- **数据库** ：Task 元数据（ID、时间、描述）
- **文件系统** ：`contexts/<sessionID>/tasks/` 下的 protobuf 文件
- **格式** ：每个审计条目包含 Task 上下文、命令内容、请求/响应、时间戳

### 日志轮转

RPC 日志（`auth.log`、`rpc.log`）每日零点自动轮转。

## 通知机制

Server 通过 EventBroker 订阅事件，满足条件时分发到外部通知渠道。

### 支持的渠道

| 渠道 | 认证方式 |
|------|---------|
| **Telegram** | Bot API Key + Chat ID |
| **DingTalk** | Token + Secret |
| **Lark** | Webhook URL |
| **ServerChan** | API URL |
| **PushPlus** | Token + Topic + Channel |

### 事件流

```
Implant 上线 / 任务完成 / ...
    ↓ publish
EventBroker
    ↓ event.IsNotify == true
Notifier
    ↓ dispatch
Telegram / DingTalk / Lark / ...
```

配置见 [Server 配置参考 - 消息通知配置](index.md#消息通知配置)。

## LLM 代理机制

Server 为 Client 侧的 AI Agent 提供 LLM Provider 桥接服务。

### Provider 解析优先级

1. 请求级参数
2. Server config 中的 provider 配置（`server.llm.providers.<name>`）
3. 全局 LLM 配置（`server.llm`）
4. 环境变量（`BRIDGE_<PROVIDER>_API_KEY`、`BRIDGE_<PROVIDER>_BASE_URL`）
5. 预设默认值

### 预设 Provider

| Provider | 默认 Endpoint |
|----------|--------------|
| OpenAI | `https://api.openai.com/v1` |
| OpenRouter | `https://openrouter.ai/api/v1` |
| DeepSeek | `https://api.deepseek.com/v1` |
| Groq | `https://api.groq.com/openai/v1` |
| Moonshot | `https://api.moonshot.cn/v1` |

### 调用流程

```
Client (chat/skill)
    ↓ MaliceRPC
Server LLM Proxy
    ↓ POST /chat/completions
LLM Provider API
    ↓ response
Server → Client
```

配置见 [Server 配置参考 - LLM Provider 配置](index.md#llm-provider-配置)。

## 实现位置

| 目录 | 职责 |
|------|------|
| `server/rpc/` | RPC Handler、中间件、认证 |
| `server/internal/db/` | 数据库适配、模型定义 |
| `server/internal/core/` | Session/Task/Event/Pipeline 运行时 |
| `server/internal/audit/` | 审计日志 |
| `server/internal/notify/` | 通知分发 |
| `server/internal/llm/` | LLM Provider 代理 |
| `server/internal/configs/` | 配置加载 |
| `external/IoM-go/generate/proto/` | Proto 定义 |

## 相关文档

- [Server 配置参考](index.md) — config.yaml 完整配置
- [系统架构](../concept.md) — 整体架构说明
- [Listener 架构](listeners.md) — Listener/Pipeline 设计
