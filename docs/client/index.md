# 架构与配置

本目录包含 malice-network Client 的机制、架构与设计文档。

具体操作指南请参考 [操作手册](../operations/)。

## 文档列表

- [快速开始](quickstart.md) - 下载、登录、首次操作
- [命令行系统](console.md) - 上下文架构、TUI 多窗口、MCP/LocalRPC 集成
- [插件体系](plugin.md) - MAL / Alias / Extension / Addon / Armory 架构设计
- [Agent 集成](agent.md) - MCP、chat、tapping、skill、schema、tool_call 机制

## Client 架构

Client 是 malice-network 的操作入口，核心职责：

- **命令行系统** ：基于 Cobra 的两级上下文（Client / Implant），支持交互式、TUI、非交互式多种模式
- **插件体系** ：四种扩展机制（MAL / Alias / Extension / Addon）+ Armory 统一分发
- **外部集成** ：通过 MCP 暴露给 AI Agent，通过 LocalRPC 暴露给多语言 SDK

详细架构说明请参考 [系统架构](../concept.md)。

## 命令体系

### 命令分组

Client 的命令按职责分组，不同上下文下可用命令不同：

**Client 上下文（根菜单）** ：

| 命令组 | 命令 | 操作文档 |
|--------|------|----------|
| Generic | `login` / `version` / `status` / `exit` / `!` | [快速开始](quickstart.md) |
| Manage | `session` / `mal` / `alias` / `extension` / `armory` / `config` / `cert` / `audit` / `context` | [后渗透操作](../operations/post-exploitation/)、[插件体系](plugin.md) |
| Listener | `listener` / `pipeline` / `website` | [Listener 操作](../operations/listener.md) |
| Generator | `build` / `profile` / `mutant` | [构建操作](../operations/build.md) |

**Implant 上下文（Session 菜单）** ：

| 命令组 | 命令 | 操作文档 |
|--------|------|----------|
| Implant | `info` / `init` / `tasks` / `list_module` / `load_module` | [会话管理](../operations/post-exploitation/session-management.md)、[模块管理](../operations/post-exploitation/module-management.md) |
| Execute | `run` / `execute` / `shell` / `powershell` / `bof` / `execute_exe` / `inline_exe` / `execute_assembly` | [命令执行](../operations/post-exploitation/command-execution.md) |
| Sys | `whoami` / `env` / `ps` / `kill` / `service` / `reg` / `taskschd` | [系统信息](../operations/post-exploitation/system-info.md)、[服务管理](../operations/post-exploitation/service-management.md) |
| File | `ls` / `cd` / `upload` / `download` / `cat` / `mkdir` / `rm` / `mv` / `cp` | [文件操作](../operations/post-exploitation/file-operations.md) |
| Pivot | `portfwd` / `rportfwd` / `proxy` / `reverse` | [网络代理](../operations/post-exploitation/network-proxy.md)、[代理配置](../operations/proxy.md) |
| Armory / Addon | 动态注册 | [插件体系](plugin.md)、[嵌入式 MAL](../operations/embed-mal.md) |

### 命令注册机制

Client 的命令树通过 `BindFunc` 机制统一注册：

- **内置命令** ：通过 `BindCommonCommands`（Client 上下文）和 `BindBuiltinCommands`（Implant 上下文）在启动时注册
- **插件命令** ：MAL 插件在加载时向命令树动态注册，按 Custom / Community / Professional 分层
- **Alias / Extension** ：在 Implant 命令树初始化时从 manifest 文件解析并注册

### 异步任务机制

`run`、`execute`、`shell`、`powershell` 等命令是异步回传结果的：

- 使用 `-f/--file` 保存输出时，Client 会等待任务完成后将所有输出分片汇总写入文件
- `-f` 适合保存完整结果，不需要额外 `--wait`
- 命令启用静默模式或目标侧无输出时，文件可能为空

### 实现位置

| 目录 | 职责 |
|------|------|
| `client/cmd/cli/` | 启动入口、TUI mux |
| `client/core/` | Console 状态、MCP、LocalRPC、插件桥接 |
| `client/command/` | 命令树注册与实现 |
| `client/plugin/` | MAL 运行时与插件管理 |

## 相关文档

- [操作指南](../operations/) - 部署、构建、后渗透等具体操作
- [Server 架构](../server/) - Server 端架构与配置
- [开发文档](../development/) - MAL 开发、协议、SDK
