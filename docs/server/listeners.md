# Listener 与 Pipeline 架构

本文档说明 Listener 与 Pipeline 的架构设计、类型体系和核心机制。

操作指南见 [Listener 操作](../operations/listener.md)。

## 架构设计

### Listener 的角色

Listener 是 malice-network 的分布式通信层，与 Server 解耦设计：

- **分布式部署** ：可以独立部署在任意服务器上，不需要与 Server 同机
- **与 Server 解耦** ：通过 gRPC Stream 与 Server 全双工通信，独立运行和故障隔离
- **多 Pipeline 承载** ：每个 Listener 可运行多个不同类型的 Pipeline

### Server 与 Listener 的传输模式

`listeners.transport` 控制 Server 与 Listener 之间的控制/任务通道方向：

| 模式 | 方向 | 适用场景 |
|------|------|----------|
| `reverse` | Listener 主动连接 Server | 默认模式；Listener 能访问 Server 的 gRPC 端口 |
| `forward` | Server 主动连接 Listener | Listener 不能访问 Server，但 Server 能访问 Listener 的 forward 端口 |

`reverse` 是默认值，沿用 `listener.auth` 和 `ListenerRPC` 的 `JobStream` / `SpiteStream`。`forward` 会让 Listener 本地启动一个 forward gRPC 服务，Server 拨入后通过 `ControlStream` 下发 pipeline 控制，通过 `TaskStream` 下发任务并接收 session 事件/任务响应。

两种模式都保持同一个 Root CA 层级。`listener.auth` 只包含 CA 公钥证书和 Listener 自己的小证书私钥，不包含 Root CA 私钥。新生成的 Listener 小证书同时具备 `clientAuth` 和 `serverAuth`：

- `reverse` 模式下，Listener 用 `listener.auth` 的 `cert/key` 作为客户端证书主动连接 Server。
- `forward` 模式下，Listener 用同一份 `cert/key` 作为 forward gRPC server 证书等待 Server 拨入。
- Server 拨入 Listener 时使用 Server 本地 Root CA 签发/保存的 forward client cert，不会读取或复用 Listener 的私钥。
- 双方都用 Root CA 公钥验证对端证书链和证书用途。

### Listener 身份一致性

Listener 的唯一身份来自 mTLS 证书对应的 Operator。`listeners.name`、auth 文件中的 `operator` 和 RPC 中的 `listener_id` 必须完全一致；不同名称的 Listener 不能共用同一份 auth。Server 会同时校验证书 fingerprint、RPC metadata 和请求消息中的 Listener ID，不一致的请求会以 `PermissionDenied` 拒绝，避免同一运行实例在不同 ID 下重复注册并产生 Pipeline 状态分裂。

Listener-only forward 最小示例：

```yaml
listeners:
  enable: true
  name: listener-a
  ip: 10.10.1.20
  transport: forward
  forward:
    listen_host: 0.0.0.0
    listen_port: 5005
```

`listen_*` 是 Listener 端绑定地址。Server 端拨号地址由 Server 侧 `connect_*` 配置或 `listener forward connect --host/--port` 提供。forward 模式要求 Server 到 Listener 的 TCP 连接可达；如果双向都不可达，需要额外 relay/中继。

### forward 动态连接

forward Listener 可以通过配置在 Server 启动时自动连接，也可以由 Client 触发 Server 主动连接：

```bash
listener forward connect listener-a --host 10.10.1.20 --port 5005
listener forward status listener-a
listener forward list
listener forward disconnect listener-a
```

动态连接时 `--host` 必须显式提供，`--port` 默认是 `5005`。Server 只把 `--host` 当作拨号地址，不会从 Operator `Remote` 字段推断 forward 地址，也不会把拨号地址作为证书身份授权依据。

动态连接只负责建立 Server 到 Listener 的控制/任务通道，不会生成或分发证书。被连接的 `listener_id` 必须已经在 Server DB 中有 Listener Operator 记录，通常由 `listener add` / `listener reset` 或服务端初始化流程生成。Server 会读取该记录里的 Listener 证书 fingerprint，并在 mTLS 握手中要求 Listener forward server cert 与该 fingerprint 匹配。

forward 管理 RPC 是 admin-only。普通 operator 即使默认可访问 MaliceRPC，也不能触发 `ConnectForwardListener` / `DisconnectForwardListener` / `GetForwardListenerStatus` / `ListForwardListeners` / `RetireListener`。DB 不新增 transport 字段；正向/反向是运行时连接方向和配置含义，身份校验仍由证书链、EKU 和 Operator fingerprint 决定。forward client 会用 CA、`serverAuth` EKU 和 DB fingerprint 校验 Listener 证书，不要求 Listener 证书的 DNS/IP SAN 必须等于当前拨号 host。

### Listener retire

Server 可以对已连接 Listener 下发 retire 控制命令。retire 会先等待 Listener 返回确认，再清理 Server 侧运行时记录，并默认 revoke 同名 Listener Operator：

```bash
listener retire listener-a --yes
listener retire listener-a --purge-config --purge-auth --yes
listener retire listener-a --no-revoke --yes
```

`--purge-config` 只删除 Listener 进程当前 `-c` 使用的配置文件；`--purge-auth` 只删除 Listener 配置中的 `auth` 文件。两个删除行为都是显式 opt-in，默认不删除本地文件。`--no-revoke` 会保留 Server DB 中同名 Listener Operator 的有效性，默认不建议使用。`--timeout` 控制 Server 等待 Listener retire 确认的秒数。

删除 Listener Operator 只适合处理未连接的身份记录。若同名 reverse Listener 运行时仍处于 active 状态，或 Server 侧仍保持 forward Listener runtime，`RemoveListener` 会返回 `FailedPrecondition` 并保留 DB 中的身份记录。需要先执行 `listener retire`，或对 forward runtime 执行 `listener forward disconnect`，再删除或重置身份。

!!! warning "forward 模式当前限制"
    旧的 client-only `listener.auth` 仍可用于 `reverse`，但用于 `forward` 时需要重新生成双用途 Listener auth。forward transport 解决的是 Server 主动拨入 Listener 的方向问题；如果 Server 也无法访问 Listener，需要额外 relay/中继。

```
┌─────────┐  gRPC/mTLS  ┌──────────┐
│  Server  │◄───────────►│ Listener │
│          │             │          │
│ 状态管理  │             │ ┌──────┐ │   TCP     ┌─────────┐
│ 任务编排  │             │ │ TCP  │◄├──────────►│ Implant │
│ RPC 服务  │             │ ├──────┤ │           └─────────┘
│          │             │ │ HTTP │ │   HTTP
│          │             │ ├──────┤ │           ┌─────────┐
│          │             │ │ REM  │◄├──────────►│ Implant │
│          │             │ ├──────┤ │           └─────────┘
│          │             │ │ Web  │ │
│          │             │ └──────┘ │
└─────────┘             └──────────┘
```

### Pipeline 的角色

Pipeline 是 Listener 与 Implant 交互的具体传输实现：

- 每个 Pipeline 负责一种协议的监听、解析和路由
- Pipeline 相当于传统 C2 中的"Listener"概念，但 IoM 进一步细分了层次

## Pipeline 类型

| 类型 | 协议 | 用途 |
|------|------|------|
| **TCP** | TCP（可选 TLS） | 最基础的传输，直连场景 |
| **HTTP** | HTTP/HTTPS | 伪装为 Web 流量，支持自定义 Header/Body |
| **REM** | 自定义协议 | 基于 [rem](https://github.com/chainreactors/rem) 的灵活传输 |
| **Bind** | 反向连接 | Implant 监听端口，Client 主动连接（不稳定） |
| **Website** | HTTP | 文件托管和伪装 |
| **Custom** | 自定义 | 第三方 Pipeline 接入，详见 [自定义 Pipeline 开发](../development/custom-pipeline-guide.md) |

## 核心机制

### TLS 配置

Pipeline 的 TLS 支持两种配置方式：

=== "config.yaml 配置"

    ```yaml
    tcp:
      
      - name: tcp
        tls:
          enable: true                # 使用自签名证书
          cert_file: path/to/cert     # 或指定证书路径
          key_file: path/to/key
          ca_file: path/to/ca         # 可选
    ```

=== "Client 命令配置"

    ```bash
    cert self_signed                  # 生成自签名证书
    cert import --cert cert.crt --key key.crt  # 导入证书
    pipeline cert bind --pipeline tcp --listener listener-a --cert-name <name>
    cert update --cert-name <name> --cert fullchain.pem --key privkey.pem
    ```

    `pipeline cert bind` 对运行中的 HTTP/TCP Pipeline 执行停止、持久化和重新启动；启动失败时恢复原绑定。停止态只更新数据库。`cert update` 默认通过 `cert apply` 语义重载所有引用，未提供 CA 时保留原 CA。

!!! warning "Implant 对齐"
    Pipeline 开启 TLS 时，Implant 的 profile 中也需要同步开启 `tls.enable: true`。

### Parser 机制

Parser 决定 Pipeline 如何解析 Implant 的通信协议：

| Parser | 说明 |
|--------|------|
| `auto` | 自动检测 Implant 类型 |
| `malefic` | 解析 malefic 主 implant 协议 |
| `pulse` | 解析 pulse 轻量 implant 协议 |

### Encryption 机制

Pipeline 与 Implant 之间的通信加密：

```yaml
encryption:
  - enable: true
    type: aes              # 支持 aes / xor
    key: maliceofinternal  # 密钥需与 Implant profile 一致
```

支持配置多层加密（如同时启用 AES + XOR）。

### HTTP 自定义响应

HTTP Pipeline 支持自定义响应内容，用于流量伪装：

```yaml
http:
  - name: http
    headers:                               # 自定义响应头
      Server: ["nginx/1.22.0"]
      Content-Type: ["text/html; charset=utf-8"]
    error_page: "/var/www/error.html"      # 错误页面
    body_prefix: "<!-- prefix -->"         # Body 前缀
    body_suffix: "<!-- suffix -->"         # Body 后缀
```

### Pipeline 身份与同名规则

Pipeline 的唯一身份是 `listener_id + name`：

- 同一个 Listener 下不能有两个同名 Pipeline。
- 不同 Listener 之间可以使用相同 Pipeline 名称。
- 服务端只收到 `name` 且发现跨 Listener 同名时，会要求调用方补充 `listener_id`，避免误操作。
- 客户端缓存中，如果名称唯一，仍可用 `name` 访问；一旦跨 Listener 同名，会使用 `listener_id:name` 作为缓存 key。
- Profile 也会保存 Pipeline 的 Listener 维度；创建 Profile 时可使用 `listener_id:pipeline_name` 指向跨 Listener 同名 Pipeline。
- 自动默认 Profile 在无重名时沿用 `pipeline_default`，出现跨 Listener 同名时使用 `listener_id_pipeline_default` 避免撞名。
- Website 托管内容同样按 `listener_id + website name + path` 识别；跨 Listener 同名 Website 不会共享内容、文件目录或删除操作；历史无 `listener_id` 的内容只会在同名 Website 唯一时兼容读取。
- REM agent 控制命令和 REM dial 可接收 `listener_id:pipeline_name`，用于跨 Listener 同名 REM 的精确路由；历史无 listener 的 REM agent context 在同名 REM 有歧义时不会被自动归属。

### REM 配置同步

REM Pipeline 的 `console` 是预注册入口，`link` 是启动后生成的当前连接地址：

```yaml
rem:
  - enable: true
    name: rem_default
    console: tcp://0.0.0.0:20000
    link: tcp://10.0.0.1:20000
```

- 首次启动时，如果 `link` 为空，Listener 会根据 `console` 启动 REM，并把生成的 `link` 同步到数据库和 `config.yaml`。
- 如果数据库中已经存在同名 REM，系统会保留数据库中的 REM 身份和 `link`，并把缺失或旧的 `config.yaml` 字段回填为数据库中的值。
- 同一个 Listener 下启用状态的 REM `name` 不能重复；如需不同 REM，请使用不同名称。

## 独立部署

Listener 可以独立部署在与 Server 不同的服务器上：

```bash
./malice-network listener add listener-a
```

该命令会在当前目录生成同名的 `listener-a.auth` 和 `listener-a.yaml`。生成的 YAML 可直接用于 listener-only 模式：

```yaml
listeners:
  enable: true
  name: listener-a
  auth: listener-a.auth
  transport: reverse
```

将以下文件部署到 Listener 主机：

- `malice-network` 可执行文件
- `listener-a.yaml` 配置文件
- `listener-a.auth` 认证凭证

```bash
./malice-network --listener-only -c listener-a.yaml
```

auth 相对路径以 YAML 文件所在目录为基准解析。`listener reset listener-a` 会更新 auth，但不会覆盖已经添加 Pipeline 或 forward 设置的 YAML。

## 实现位置

| 文件 | 职责 |
|------|------|
| `server/listener/listener.go` | Listener 生命周期管理 |
| `server/listener/forward.go` | forward transport Listener 端服务 |
| `server/listener/retire.go` | Listener retire 本地文件清理与关闭调度 |
| `server/listener/tcp.go` | TCP Pipeline 实现 |
| `server/listener/http.go` | HTTP Pipeline 实现 |
| `server/listener/rem.go` | REM Pipeline 实现 |
| `server/listener/custom.go` | Custom Pipeline 接入 |
| `server/rpc/rpc-forward-listener.go` | forward transport Server 端拨号与 stream 适配 |
| `server/rpc/rpc-forward-listener-manage.go` | Client 触发的 forward Listener 连接管理 RPC |
| `server/rpc/rpc-listener-retire.go` | Server 端 Listener retire RPC |
| `server/forwardrpc/forwardrpc.go` | forward transport 手写 gRPC service descriptor |
| `server/internal/core/pipeline.go` | Pipeline 运行时状态 |

## 相关文档

- [Server 配置参考](index.md) - config.yaml 完整配置
- [Listener 操作指南](../operations/listener.md) - 具体操作与使用
- [自定义 Pipeline](../development/custom-pipeline-guide.md) - 第三方 Pipeline 开发
- [代理配置](../operations/proxy.md) - REM 代理集成
