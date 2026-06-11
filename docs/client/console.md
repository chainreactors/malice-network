# 命令行系统

Client 提供多种运行模式，覆盖交互操作、自动化脚本和外部集成等场景。

## 运行模式总览

| 模式 | 启动方式 | 用途 |
|------|---------|------|
| 交互式 Shell | `./iom` | 日常操作，人工交互 |
| TUI 多窗口 | `./iom --tui` | 多 session 并行操作 |
| 非交互执行 | `./iom --auth admin.auth session` | 脚本集成，单次命令 |
| 服务进程 | `./iom --auth admin.auth --daemon` | 不进入交互控制台，保持 MCP/LocalRPC 等服务运行 |
| MCP 服务 | `./iom --auth admin.auth --daemon --mcp <addr>` | AI Agent 接入 |
| LocalRPC 服务 | `./iom --auth admin.auth --daemon --rpc <addr>` | SDK 编程接入 |

## 交互式 Shell

### 基本启动

```bash
./iom                   # 选择已保存凭证登录
./iom login admin.auth  # 首次导入凭证
```

登录后进入交互式命令行，支持 Tab 补全、命令历史、分组帮助。

### 上下文系统

Client 采用两级上下文架构，不同上下文下可用命令不同：

=== "Client 上下文（根菜单）"

    登录后默认进入，管理 Server 资源：

    | 命令组 | 命令 | 说明 |
    |--------|------|------|
    | Generic | `login` / `version` / `status` / `exit` | 基础操作 |
    | Manage | `session` / `mal` / `alias` / `extension` / `armory` / `config` / `cert` / `audit` | 资源管理 |
    | Listener | `listener` / `pipeline` / `website` | 通信管理 |
    | Generator | `build` / `profile` / `mutant` | 构建管理 |

=== "Implant 上下文（Session 菜单）"

    使用 `use <session-id>` 进入，执行后渗透操作：

    | 命令组 | 命令 | 说明 |
    |--------|------|------|
    | Implant | `info` / `init` / `tasks` / `list_module` / `load_module` | 会话与模块 |
    | Execute | `run` / `execute` / `shell` / `powershell` / `bof` / `execute_exe` / `inline_exe` / `execute_assembly` | 命令执行 |
    | Sys | `whoami` / `env` / `ps` / `kill` / `service` / `reg` / `taskschd` | 系统操作 |
    | File | `ls` / `cd` / `upload` / `download` / `cat` / `mkdir` / `rm` / `mv` / `cp` | 文件操作 |
    | Pivot | `portfwd` / `rportfwd` / `proxy` / `reverse` | 网络代理 |

!!! info "动态命令"
    Implant 上下文的可用命令取决于 implant 已加载的模块。`nano` 模式的 implant 只有最小命令集，可通过 `load_module` 按需扩展。

### 上下文切换

```bash
use <session-id>    # Client → Implant 上下文
background          # Implant → Client 上下文
switch              # 在多个 session 间快速切换
```

### 补全与帮助

- **Tab 补全** ：命令名、子命令、Flag、Session ID、Pipeline 名称等均支持智能补全
- **帮助系统** ：`help` 按分组列出所有命令，`help <command>` 查看详细用法
- **Shell 转义** ：使用 `!` 前缀执行本地命令，如 `! ls -la`

## TUI 多窗口模式

使用 `--tui` 启动类似 tmux 的多 pane 管理界面：

```bash
./iom --tui
```

### 布局与机制

```
┌──────────────┬─────────────────────────────────────┐
│  Sidebar     │                                     │
│              │          Active Pane                 │
│  console-0   │                                     │
│  08d6c05a    │   [08d6c05a] > whoami               │
│  a3f2b1e7    │   NT AUTHORITY\SYSTEM               │
│              │                                     │
│              │                                     │
└──────────────┴─────────────────────────────────────┘
```

- **Index pane** （首个 pane）：接收全局事件输出，承载 MCP 和 LocalRPC 服务
- **Session pane** ：`use <session>` 时自动创建独立 pane，以 `--quiet` 模式运行，屏蔽全局事件
- **切换** ：通过侧边栏在不同 pane 间切换

## 非交互式执行

直接追加子命令，执行完毕后退出：

```bash
./iom --auth admin.auth session             # 列出会话
./iom --auth admin.auth listener            # 列出 listener
./iom --auth admin.auth build beacon \
    --profile tcp_default \
    --target x86_64-pc-windows-gnu          # 编译 beacon
```

!!! tip "凭证指定"
    非交互模式下通过 `--auth` 显式指定凭证文件，避免进入凭证选择交互。

## Daemon 模式

使用 `--daemon` 保持服务运行，不进入交互式控制台。它不会 fork 成系统后台进程，适合配合 systemd、任务调度器或脚本进程管理使用：

```bash
./iom --auth admin.auth --daemon --mcp 127.0.0.1:5005 --rpc 127.0.0.1:15004
```

典型场景：

- 为 AI Agent 提供持久 MCP 服务
- 为 SDK 提供持久 LocalRPC 服务
- 搭配脚本实现无人值守自动化

## MCP 服务

通过 `--mcp` 暴露 [Model Context Protocol](https://modelcontextprotocol.io/) 接口：

```bash
./iom --auth admin.auth --daemon --mcp 127.0.0.1:5005
```

MCP 服务会注册通用工具和 Client 命令接口：

- **Tools** ：`execute_command`、`execute_lua`、`search_commands`、`get_history`，以及从 Cobra 命令树生成的命令 Tool
- **Resources** ：从命令树生成的只读 Resource，用于查询 session、listener、artifact 等常用状态

支持 Claude Code / Claude Desktop 等任何兼容 MCP 协议的客户端接入。详见 [AI Agent 集成](agent.md)。

## LocalRPC 服务

通过 `--rpc` 暴露本地 gRPC 接口，供多语言 SDK 编程接入：

```bash
./iom --auth admin.auth --daemon --rpc 127.0.0.1:15004
```

LocalRPC 当前提供命令执行、Lua 执行、历史查询、命令 schema/group 查询、命令搜索和流式命令输出等接口。

| SDK | 仓库 |
|-----|------|
| Go | [IoM-go](https://github.com/chainreactors/IoM-go) |
| Python | [IoM-python](https://github.com/chainreactors/IoM-python) |
| TypeScript | [IoM-typescript](https://github.com/chainreactors/IoM-typescript) |

## 启动参数速查

| 参数 | 说明 |
|------|------|
| `--auth <path>` | 指定认证凭证文件 |
| `--tui` | TUI 多窗口模式 |
| `--daemon` | 服务进程模式，不进入交互式控制台 |
| `--mcp <addr>` | 启用 MCP 服务 |
| `--rpc <addr>` | 启用 LocalRPC 服务 |
| `--console` | 带子命令执行后仍进入交互式控制台，不能与 `--daemon` 同时使用 |
| `--yes` | 跳过确认提示 |

## 相关文档

- [快速开始](quickstart.md) — 首次登录与操作
- [插件体系](plugin.md) — MAL / Alias / Extension / Addon 架构
- [AI Agent 集成](agent.md) — MCP 与 Agent 详解
- [操作指南](../operations/) — 具体操作手册
