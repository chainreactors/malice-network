# Server 架构与配置

本目录包含 malice-network Server 的架构设计和配置参考。

具体操作指南请参考 [操作手册](../operations/)。

## 文档列表

- [快速开始](quickstart.md) - Server 下载、启动、首次配置
- [Listener 架构](listeners.md) - Listener 与 Pipeline 的设计机制
- [构建系统架构](build.md) - Build 系统与构建源的设计机制
- [内部机制](internals.md) - RPC 通信、数据持久化、审计/通知/LLM

## Server 架构

Server 是 malice-network 的控制核心，职责包括：

- **状态管理** ：Session / Task / Pipeline 生命周期与并发状态
- **任务编排** ：Task 调度、回调路由、结果解析
- **RPC 服务** ：MaliceRPC（Client 接口）、RootRPC（管理接口）、ListenerRPC（Listener 接口）
- **构建控制** ：Profile 管理、Artifact 构建编排（Docker / GitHub Action / SaaS）
- **审计与通知** ：操作审计记录、第三方消息推送
- **LLM 代理** ：为 Client 侧 AI Agent 提供 LLM Provider 桥接

详细架构说明请参考 [系统架构](../concept.md)。

## 启动模式

| 启动方式 | 说明 |
|----------|------|
| `./malice-network -i <ip>` | Server + Listener 一起启动（最常用） |
| `./malice-network --server-only` | 仅启动 Server，不启动 Listener |
| `./malice-network --listener-only` | 仅启动 Listener，独立部署时使用 |
| `./malice-network --daemon` | 不进入交互向导，按常规服务进程运行并等待退出信号 |
| `./malice-network --quickstart` | 显式进入交互式配置向导，引导完成初始配置 |

| 参数 | 说明 |
|------|------|
| `-c, --config` | 配置文件路径（默认 `config.yaml`） |
| `-i, --ip` | 外网 IP 地址，覆盖配置文件中的 ip 字段 |
| `--debug` | 开启 debug 日志 |
| `--opsec` | 启用 OPSEC 模式 |

当默认启动时，如果配置文件不存在：

- 交互终端会先提示是否进入 quickstart
- 非交互环境会直接生成默认配置并继续启动
- 只有显式传入 `--quickstart` 时，才会强制进入向导

!!! tip "部署指南"
    完整的部署流程见 [部署操作指南](../operations/deployment.md)。

## config.yaml 配置参考

config.yaml 主要分为 `server`（Server 本体配置）和 `listeners`（Listener 与 Pipeline 配置）。完整模板以仓库中的 `server/config.example.yaml` 为准，本节只列常用字段。

### server section

#### 基础配置

```yaml
server:
  enable: true
  ip: 127.0.0.1                    # Server 外网 IP（可被 -i 参数覆盖）
  grpc_host: 0.0.0.0               # gRPC 监听地址
  grpc_port: 5004                   # gRPC 监听端口
  encryption_key: maliceofinternal  # Server 与 Listener 间通信加密密钥
  audit: 1                          # 审计级别
```

#### 传输配置

```yaml
server:
  config:
    packet_length: 1048576          # 模板默认 1MB，可按传输场景调整
    certificate: null               # 自定义证书路径
    certificate_key: null           # 自定义证书私钥路径
```

!!! tip "大文件传输"
    网络环境较差或需要传输大文件时，可增大 `packet_length`，例如设为 `10485760`（10MB）。

#### GitHub Action 构建源

```yaml
server:
  github:
    owner: <github-owner>          # GitHub 用户名或组织名
    repo: malefic                   # 仓库名
    token: <github-token>           # Personal Access Token
    workflow: generate.yml          # Workflow 文件名
```

#### SaaS 构建源

```yaml
server:
  saas:
    enable: true                    # 是否启用 SaaS 编译
    url: https://build.chainreactors.red  # 编译服务地址
    token: null                     # 认证 Token
```

!!! warning "安全提示"
    使用默认 SaaS 编译视为同意用户协议。可通过设置 `enable: false` 关闭，转用 Docker 或 GitHub Action 私有化编译。

#### LLM Provider 配置

```yaml
server:
  llm:
    default_provider: openai        # 默认 LLM Provider
    providers:
      openai:
        api_key: ""                 # API Key
        endpoint: https://api.openai.com/v1  # API Endpoint
    proxy_url: ""                   # 代理地址
    timeout: 120                    # 超时时间（秒）
```

#### 消息通知配置

```yaml
server:
  notify:
    enable: false                   # 是否启用通知
    telegram:
      enable: false
      api_key: ""                   # Telegram Bot API Key
      chat_id: ""                   # Telegram Chat ID
    dingtalk:
      enable: false
      secret: ""                    # 钉钉 Secret
      token: ""                     # 钉钉 Token
    lark:
      enable: false
      webhook_url: ""               # 飞书 Webhook URL
    serverchan:
      enable: false
      url: ""                       # ServerChan API Key
    pushplus:
      enable: false
      token: ""                     # PushPlus Token
      topic: ""                     # 消息主题
      channel: ""                   # 推送渠道（wechat/email/telegram）
```

### listeners section

#### 基础配置

```yaml
listeners:
  name: listener                    # Listener 名称
  ip: 127.0.0.1                    # Listener 外网 IP
  auth: listener.auth              # 认证凭证文件路径
  enable: true                     # 是否启用
```

#### 自动构建

```yaml
listeners:
  auto_build:
    enable: true                   # 是否启用自动构建
    build_pulse: true              # 是否自动编译 pulse
    pipeline:                      # 为哪些 pipeline 自动构建
      - tcp
      - http
    target:                        # 自动构建的目标架构
      - x86_64-pc-windows-gnu
```

#### Pipeline 配置

每种 Pipeline 类型的配置结构见 [Listener 架构](listeners.md)，操作指南见 [Listener 操作](../operations/listener.md)。

**TCP Pipeline** :

```yaml
listeners:
  tcp:
    - name: tcp
      port: 5001
      host: 0.0.0.0
      parser: auto                 # 协议解析器（auto/malefic/pulse）
      enable: true
      tls:
        enable: true
      encryption:
        - enable: true
          type: aes                # 加密类型（aes/xor）
          key: maliceofinternal
```

**HTTP Pipeline** :

```yaml
listeners:
  http:
    - name: http
      port: 8080
      host: 0.0.0.0
      parser: auto
      enable: true
      tls:
        enable: true
      encryption:
        - enable: true
          type: aes
          key: maliceofinternal
      error_page: ""               # 自定义错误页面路径
```

**REM Pipeline** :

```yaml
listeners:
  rem:
    - name: rem_default
      enable: true
      console: null                # REM 控制台监听地址
```

**Bind Pipeline** :

```yaml
listeners:
  bind:
    - name: bind_default
      enable: false
```

**Website Pipeline** :

```yaml
listeners:
  website:
    - name: default-website
      port: 80
      root: /
      enable: true
```

## 实现位置

| 目录 | 职责 |
|------|------|
| `server/cmd/server/` | 启动入口 |
| `server/internal/configs/` | 配置加载与管理 |
| `server/internal/core/` | Session / Task / Pipeline 运行时状态 |
| `server/rpc/` | RPC Handler（MaliceRPC / RootRPC / ListenerRPC） |
| `server/listener/` | Listener 与 Pipeline 实现 |
| `server/build/` | 构建编排 |
| `server/internal/db/` | 数据库与持久化 |
| `server/internal/audit/` | 审计记录 |
| `server/internal/notify/` | 消息通知 |
| `server/internal/llm/` | LLM Provider 代理 |

## 相关文档

- [操作指南](../operations/) - 部署、构建、后渗透等具体操作
- [Client 架构](../client/) - Client 端机制与设计
- [系统架构](../concept.md) - 整体架构说明
