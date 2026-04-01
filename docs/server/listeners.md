# Listener 与 Pipeline

本文档说明 Listener 与 Pipeline 的职责、启动方式、配置项和实现位置。

## 1. Listener 是什么

Listener 是 `malice-network` 的分布式通信层，负责：

- 接收 implant 的真实网络连接
- 维护不同 pipeline 的监听状态
- 对接 parser、加密和协议转换
- 通过 gRPC / mTLS 和 server 同步状态与任务

Listener 负责通信接入与传输处理，Server 负责状态管理与任务编排。

## 2. 当前支持哪些 Pipeline

按 `server/listener/` 目录里的实现，当前仓库里已经落地的类型包括：

- `tcp`
- `http`
- `bind`
- `rem`
- `website`
- `custom`

其中 `custom` 的详细接入方式见 [../custom-pipeline-guide.md](../custom-pipeline-guide.md)。

## 3. 默认配置长什么样

当前 `server/config.yaml` 默认会启用：

- listener 名称: `listener`
- listener IP: `127.0.0.1`
- TCP pipeline: `tcp`, 默认端口 `5001`
- HTTP pipeline: `http`, 默认端口 `8080`
- REM pipeline: `rem_default`

最常改的字段通常是：

- `listeners.ip`
- `listeners.auth`
- `listeners.tcp[*].port`
- `listeners.http[*].port`
- `listeners.auto_build.*`

## 4. 启动模式

### Server + Listener 一起启动

常用部署模式如下：

```bash
./malice_network_linux_amd64 -i <public-ip>
```

### 只启动 server

```bash
./malice_network_linux_amd64 --server-only
```

### 只启动 listener

```bash
./malice_network_linux_amd64 --listener-only -c listener.yaml
```

独立 listener 一般需要：

- 一份可执行文件
- 一份 listener 配置
- 一份对应的 `*.auth` 凭证文件

## 5. Root 命令怎么管理 Listener

按当前 `server/root/listener.go`，内置的 root 命令有：

- `listener add <name>`
- `listener del <name>`
- `listener list`
- `listener reset <name>`

其中：

- `add` 会新增 listener 并在当前目录写出 `<name>.auth`
- `reset` 会重置证书并重新生成 auth 文件
- 这些命令依赖本地 root RPC，所以需要 server 已运行

## 6. 独立部署时要注意什么

### 认证文件

Listener 侧的 mTLS 凭证通常来自：

- 默认的 `listener.auth`
- 或者 `listener add <name>` 生成的 `<name>.auth`

配置里对应字段是：

```yaml
listeners:
  auth: listener.auth
```

### 外网地址

部署时应重点核对以下字段：

- `server.ip`
- `listeners.ip`
- `-i, --ip` 启动参数

## 7. Auto Build

Listener 还可以挂上自动构建策略，当前默认结构在 `server/config.yaml` 里已经留好了：

```yaml
listeners:
  auto_build:
    enable: false
    build_pulse: false
    pipeline:
      - tcp
      - http
    target:
      - x86_64-pc-windows-gnu
```

常见用途：

- 某些 pipeline 启动后自动准备对应 artifact
- 为固定 target 预热常用产物

更完整的构建链路看 [build.md](build.md)。

## 8. 实现位置

相关实现主要位于：

1. `server/listener/listener.go`
2. `server/listener/tcp.go`
3. `server/listener/http.go`
4. `server/listener/rem.go`
5. `server/listener/custom.go`
6. `server/internal/core/pipeline.go`
