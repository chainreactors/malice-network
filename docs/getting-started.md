# Malice Network 快速开始

本文档说明发布产物、初始化启动、凭证登录和首次构建流程。内容基于当前仓库代码和发布配置。

## 1. 发布产物

Release 中的主要二进制如下：

- `malice_network_<os>_<arch>`: server / listener 端
- `iom_<os>_<arch>`: client 端

对应关系来自当前 `.goreleaser.yml`：

- server 构建产物命名为 `malice_network_*`
- client 构建产物命名为 `iom_*`

## 2. 启动 Server

### 交互式初始化

首次启动可使用交互式向导：

```bash
./malice_network_linux_amd64 --quickstart
```

向导会生成初始化所需的核心配置：

- server 外网 IP
- gRPC 监听地址和端口
- 默认 listener 名称与 IP
- 需要启用的 pipeline 类型
- 可选的构建源配置
- 可选的通知配置

### 直接启动

已准备 `config.yaml` 时，可直接启动：

```bash
./malice_network_linux_amd64 -i <public-ip>
```

常用参数：

- `-c, --config`: 指定配置文件路径，默认是 `config.yaml`
- `-i, --ip`: 覆盖配置里的外网 IP
- `--server-only`: 只启动 server
- `--listener-only`: 只启动 listener
- `--quickstart`: 仅在配置文件不存在时运行初始化向导
- `--debug`: 打开 debug 日志

### 首次启动生成的文件

按当前代码，首次初始化后通常会在工作目录生成：

- `config.yaml`: 默认配置文件
- `admin_<server-ip>.auth`: client 登录凭证
- `listener.auth`: 默认 listener 的 mTLS 凭证

## 3. 登录 Client

将 `admin_<server-ip>.auth` 放到 client 可访问的位置后执行：

```bash
./iom_linux_amd64 login ./admin_<server-ip>.auth
```

导入成功后，client 会把凭证复制到本地配置目录：

- Linux / macOS: `~/.config/malice/configs`
- Windows: `%USERPROFILE%/.config/malice/configs`

之后可以直接启动 client：

```bash
./iom_linux_amd64
```

## 4. 默认 Listener / Pipeline 是什么

当前默认 `server/config.yaml` 会启用这些基础能力：

- TCP pipeline: `tcp`, 默认端口 `5001`
- HTTP pipeline: `http`, 默认端口 `8080`
- REM pipeline: `rem_default`
- gRPC 控制面端口: `5004`

更详细的 listener 和 pipeline 说明看：

- [server/listeners.md](server/listeners.md)
- [custom-pipeline-guide.md](custom-pipeline-guide.md)

## 5. 首次构建 Artifact

Malice Network 的构建入口位于 client，基本流程如下：

1. 准备 pipeline
2. 创建 profile
3. 执行 `build`

一个最小示例：

```bash
./iom_linux_amd64 profile new --name tcp-demo --pipeline tcp
./iom_linux_amd64 build beacon --profile tcp-demo --target x86_64-pc-windows-gnu --source docker
```

注意：

- `--pipeline` 需要填现有 pipeline 名称，不是随便写的标签
- `--source` 需要与 server 端已启用的构建源匹配
- 常见构建源是 `docker`、`action`、`saas`

更完整的构建说明看 [server/build.md](server/build.md)。

## 6. 相关文档

- Client 命令总览: [client/commands.md](client/commands.md)
- MAL 插件开发: [development/mal/](development/mal/)
- Implant 概览: [implant/overview.md](implant/overview.md)
- 架构图: [architecture.md](architecture.md)
