# 构建与 Profile

本文档说明 `malice-network` 中的 Profile、Build 和构建源配置。

## 1. 基本概念

### Pipeline

Pipeline 决定 implant 通过什么网络方式和 listener 通信，比如 `tcp`、`http`、`rem`。

### Profile

Profile 是一份构建配置快照，负责把 pipeline、模块、反沙箱、guardrail、构建参数等固定下来。

### Build

`build` 命令使用某个 profile，结合 target 和构建源，真正生成 beacon / bind / prelude / module / artifact。

## 2. 典型工作流

### 第一步：创建 profile

```bash
./iom_linux_amd64 profile new --name tcp-demo --pipeline tcp
```

也可以从现成配置导入：

```bash
./iom_linux_amd64 profile load ./config.yaml --name tcp-demo --pipeline tcp
```

### 第二步：查看 profile

```bash
./iom_linux_amd64 profile list
./iom_linux_amd64 profile show tcp-demo
```

### 第三步：开始构建

```bash
./iom_linux_amd64 build beacon --profile tcp-demo --target x86_64-pc-windows-gnu --source docker
```

## 3. 当前 build 支持哪些产物

按 `client/command/build/commands.go`，常见入口包括：

- `build beacon`
- `build bind`
- `build prelude`
- `build modules`

常用辅助命令包括：

- `profile list`
- `profile new`
- `profile load`
- `profile show`
- `profile delete`

## 4. 构建源怎么选

当前命令层暴露的常见构建源有：

| 构建源 | 适合场景 | 依赖 |
| --- | --- | --- |
| `docker` | 自托管、可控、调试方便 | server 节点本地有 Docker |
| `action` | 借 GitHub Action 做远程构建 | `server.github` 配置完整 |
| `saas` | 直接接外部构建服务 | `server.saas` 配置完整 |
| `patch` | 对模板做高级补丁式构建 | 更适合已有模板与高级场景 |

## 5. Server 端需要准备什么

### Docker 构建

如果要用 `--source docker`，server 节点需要能访问 Docker 环境，并准备好构建所需镜像或模板。

### GitHub Action 构建

对应 `server/config.yaml` 里的这一段：

```yaml
server:
  github:
    owner: <github-owner>
    repo: malefic
    token: <github-token>
    workflow: generate.yml
```

### SaaS 构建

对应这一段：

```yaml
server:
  saas:
    enable: true
    url: https://build.example.com
    token: <token>
```

如果 `enable` 是 `false`，或者 `url/token` 没配全，`saas` 构建会直接失败。

## 6. 命令示例

### 构建 beacon

```bash
./iom_linux_amd64 build beacon --profile tcp-demo --target x86_64-pc-windows-gnu --source docker
```

### 构建 bind

```bash
./iom_linux_amd64 build bind --profile tcp-demo --target x86_64-pc-windows-gnu --source docker
```

### 构建 prelude

```bash
./iom_linux_amd64 build prelude --profile tcp-demo --target x86_64-pc-windows-gnu --source docker
```

### 构建 modules

```bash
./iom_linux_amd64 build modules --profile tcp-demo --target x86_64-pc-windows-gnu --source docker
```

## 7. 和 Implant 仓库的关系

`malice-network` 负责的是控制面与构建编排，不是 implant 全部源码本体。

默认 implant 家族来自：

- <https://github.com/chainreactors/malefic>

相关实现主要分布在：

- 本仓库的 `client/command/build/`
- 本仓库的 `server/build/`
- `malefic` 仓库本身

Implant 背景介绍看 [../implant/overview.md](../implant/overview.md)。

## 8. 自动构建

Listener 侧自动构建可通过 `listeners.auto_build` 启用：

```yaml
listeners:
  auto_build:
    enable: true
    build_pulse: true
    pipeline:
      - tcp
    target:
      - x86_64-pc-windows-gnu
```

适用场景包括固定 Listener、固定 Target 和固定上线流程。
