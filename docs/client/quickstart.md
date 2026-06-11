# 快速开始

本文档帮助你在最短时间内完成 Client 的下载、登录和首次操作。

## 下载

从 [GitHub Releases](https://github.com/chainreactors/malice-network/releases/latest) 下载对应平台的 Client 二进制文件。

文件名格式为 `iom_[os]_[arch]`，例如：

- `iom_linux_amd64` — Linux x86_64
- `iom_darwin_arm64` — macOS Apple Silicon
- `iom_windows_amd64.exe` — Windows x86_64

!!! tip "网络问题"
    国内服务器下载 GitHub release 可能超时，建议配置代理：
    ```bash
    export http_proxy="http://127.0.0.1:1080"
    export https_proxy="http://127.0.0.1:1080"
    ```

## 登录 Server

Server 启动后会在当前目录生成认证凭证文件 `admin_[server_ip].auth`。

!!! warning "安全提示"
    `.auth` 文件是 IoM 的认证凭证，请妥善保管，不要泄露。

### 首次登录（导入凭证）

```bash
./iom login admin_[server_ip].auth
```

执行后，凭证会被复制到用户配置目录：

| 平台 | 路径 |
|------|------|
| Linux | `~/.config/malice/configs/` |
| macOS | `~/Library/Application Support/malice/configs/` |
| Windows | `C:\Users\<user>\.config\malice\configs\` |

### 后续登录

直接运行 Client，会自动列出所有已保存的凭证供选择：

```bash
./iom
```

## 基本操作流程

### 查看 Server 状态

登录后进入交互式命令行，先了解当前环境：

```bash
status          # 查看 server 状态
listener        # 查看 listener 列表
pipeline        # 查看 pipeline 列表
session         # 查看会话列表
```

### 下载 Implant

!!! info "开箱即用"
    v0.1.1 起，Server 默认通过云编译服务自动构建 Implant，无需额外配置即可下载。

在交互式命令行中查看并下载已编译的 artifact：

```bash
artifact list
```

在 artifact 表格中选中对应条目即可下载到本地。也可以指定格式下载：

```bash
artifact download <artifact-name> --format raw
```

如需自定义编译 Implant，请参考 [构建操作指南](../operations/build.md)。

### 操作 Implant

目标上线后，使用 `session` 查看会话，使用 `use` 进入会话上下文：

```bash
session                 # 查看所有会话
use <session-id>        # 进入会话上下文
help                    # 查看当前可用命令
```

!!! tip "上下文切换"
    进入 session 上下文后，命令集会根据 implant 已加载的模块动态变化。
    使用 `background` 返回 Client 上下文。

### 执行后渗透命令

在 session 上下文中，可以执行各种后渗透操作：

```bash
whoami                  # 查看当前用户
ls                      # 列出文件
upload local.txt /tmp/  # 上传文件
download /etc/passwd    # 下载文件
```

更多后渗透操作请参考 [后渗透操作指南](../operations/post-exploitation/)。

## 启动模式

Client 支持多种启动模式，满足不同使用场景：

```bash
./iom                           # 标准交互式模式
./iom --tui                     # TUI 多窗口模式
./iom --auth admin.auth --daemon                  # 服务进程模式
./iom --auth admin.auth --daemon --mcp 127.0.0.1:5005      # 启用 MCP 服务
./iom --auth admin.auth --daemon --rpc 127.0.0.1:15004     # 启用 LocalRPC 服务
```

详细说明请参考 [命令行系统](console.md)。

## 下一步

- [插件体系](plugin.md) — MAL / Armory 扩展能力
- [AI Agent 集成](agent.md) — MCP 与 AI Agent 联动
- [操作指南](../operations/) — 完整操作手册
