# 构建系统架构

本文档说明 malice-network 构建系统的架构设计、构建源机制和 Profile 体系。

操作指南见 [构建操作](../operations/build.md)。

## 架构设计

构建系统是 Server 端的编排层，负责将 Profile + Target + 构建源 组合成最终的 Artifact。

```
┌─────────┐   profile    ┌──────────┐   dispatch   ┌─────────────┐
│  Client  │────────────►│  Server   │────────────►│ Build Source │
│ build cmd│             │ 构建编排   │             │ Docker/      │
└─────────┘             │          │             │ Action/SaaS  │
                        │          │◄────────────│              │
                        │          │  artifact   └─────────────┘
                        │          │
                        │  ┌──────┐│
                        │  │ DB   ││  存储 artifact 记录
                        │  └──────┘│
                        └──────────┘
```

### 三个核心概念

| 概念 | 说明 |
|------|------|
| **Profile** | 构建配置快照，绑定 Pipeline、模块列表、加密、guardrail 等参数 |
| **Build** | 使用 Profile + Target + Source 触发的一次构建任务 |
| **Artifact** | 构建产物，记录在 DB 中，可下载为多种格式 |

## 构建源机制

Server 支持多种构建源，按优先级自动选择或手动指定：

| 构建源 | 机制 | 依赖 | 适用场景 |
|--------|------|------|----------|
| **Docker** | 本地 Docker 容器内编译 | Server 节点安装 Docker + 构建镜像 | 自托管、可控、调试 |
| **GitHub Action** | 远程触发 GitHub Workflow | `server.github` 配置完整 | 无 Docker 环境、分布式 |
| **SaaS** | 调用外部编译服务 API | `server.saas` 配置完整 | 开箱即用、无需本地环境 |
| **Patch** | 对已有模板做补丁式构建 | 已有 artifact 模板 | 高级定制 |

!!! info "自动选择"
    未指定 `--source` 时，Server 按 Docker → GitHub Action → SaaS 的优先级自动寻找可用构建源。

### 构建源配置

=== "Docker"

    需要 Server 节点安装 Docker 并拉取构建镜像：
    ```bash
    docker pull ghcr.io/chainreactors/malefic-builder:latest
    ```

=== "GitHub Action"

    config.yaml 配置：
    ```yaml
    server:
      github:
        owner: <github-owner>
        repo: malefic
        token: <github-token>
        workflow: generate.yml
    ```

=== "SaaS"

    config.yaml 配置：
    ```yaml
    server:
      saas:
        enable: true
        url: https://build.chainreactors.red
        token: null
    ```

## Profile 体系

Profile 是构建配置的快照，包含三部分：

| 部分 | 职责 |
|------|------|
| **basic** | 连接参数：target 地址、协议、TLS、加密、HTTP 伪装 |
| **implants** | 功能配置：模块列表、hot_load、第三方模块、autorun |
| **build** | 编译选项：zigbuild、remap、OLLVM 混淆、PE metadata |

### Profile 与 Pipeline 的绑定

Profile 在创建时绑定一个 Pipeline。编译前，Profile 中的 `basic.target`、`protocol`、`tls` 配置会自动使用 Pipeline 的实际参数，确保编译出的 Implant 能正确连接。

## 构建产物

支持的构建产物类型：

| 产物 | 说明 |
|------|------|
| **Beacon** | 功能完整的主 Implant，beacon 模式运行 |
| **Pulse** | 轻量上线马（~4KB），类似 CS artifact |
| **Prelude** | 多阶段上线的中间 Implant，支持 autorun |
| **Modules** | 功能模块集合，运行时动态加载 |

### 自动构建

Listener 启动时可自动触发构建：

```yaml
listeners:
  auto_build:
    enable: true
    build_pulse: true
    pipeline: [tcp, http]
    target: [x86_64-pc-windows-gnu]
```

自动构建的优先级：Docker > GitHub Action > SaaS。

## 实现位置

| 目录 | 职责 |
|------|------|
| `client/command/build/` | 构建命令入口 |
| `server/build/` | 构建编排与调度 |
| `server/internal/db/` | Artifact 记录持久化 |

## 相关文档

- [Server 配置参考](index.md) - 构建源 config.yaml 配置
- [构建操作指南](../operations/build.md) - Profile 配置与编译操作
- [Listener 架构](listeners.md) - Pipeline 与自动构建
