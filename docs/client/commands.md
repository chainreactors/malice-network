# Client 命令总览

本文档概述 client 的启动方式、主要命令分组和对应实现位置。

## 1. Client 的角色

Client 是 operator 的主操作面，负责：

- 登录 server
- 管理 session、listener、pipeline、website
- 管理 profile 和 artifact 构建
- 加载 MAL、alias、extension、armory
- 暴露本地 RPC / MCP / TUI 能力

## 2. 启动方式

### 导入凭证登录

```bash
./iom_linux_amd64 login ./admin_<server-ip>.auth
```

### 正常进入交互式 client

```bash
./iom_linux_amd64
```

### 常用全局参数

- `--tui`: 进入多窗口 TUI 模式
- `--rpc <addr>`: 打开本地 gRPC 服务
- `--mcp <addr>`: 打开 MCP 服务
- `--daemon`: 后台保活，不直接进入交互控制台

## 3. 命令分组

`client/command/` 下的主要命令组如下：

| 命令组 | 作用 |
| --- | --- |
| `login` / `version` / `status` | 登录、版本和状态检查 |
| `sessions` | 查看、切换、标记、观察 implant session |
| `listener` / `pipeline` / `website` | 管理 listener、pipeline 与 website |
| `profile` / `build` | 构建 profile 与 artifact |
| `modules` / `addon` / `mutant` | 模块、addon、mutant 相关操作 |
| `service` / `taskschd` / `reg` / `sys` / `filesystem` | 目标侧能力调用 |
| `exec` | assembly / dll / exe / shellcode / powershell 等执行能力 |
| `mal` / `alias` / `extension` / `armory` | 插件与扩展生态 |
| `agent` / `ai` | AI / bridge / agent 相关能力 |
| `cert` / `audit` / `context` | 证书、审计、上下文数据管理 |

## 4. 典型操作流程

### 先看 server 里有什么

```bash
status
listener
pipeline
session
```

### 选中某个 session

```bash
session
use <session-id>
help
```

进入 session 上下文后，命令集合会根据 implant 已加载模块动态变化。

## 5. 构建相关命令

这条线最常用：

```bash
profile list
profile new --name tcp-demo --pipeline tcp
build beacon --profile tcp-demo --target x86_64-pc-windows-gnu --source docker
```

更完整的构建链路看 [../server/build.md](../server/build.md)。

## 6. MAL 插件相关命令

最常用的命令是：

```bash
mal list
mal load <name>
mal install <zip-or-tar.gz>
mal refresh
mal update --all
```

插件开发入门看 [../development/mal/](../development/mal/)。

## 7. 实现位置

主要实现位置如下：

- `client/command/generic/`: 登录、版本、广播等基础命令
- `client/command/sessions/`: session 入口
- `client/command/build/`: build / profile
- `client/command/mal/`: MAL 安装、加载、更新
- `client/plugin/`: MAL 运行时和插件管理
- `client/core/`: client 状态、事件、RPC、插件桥接
