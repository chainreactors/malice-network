# 原生 SOCKS5 代理

## 概述

`socks5` 在 Client 本机启动 SOCKS5 监听器，并通过当前 Implant 的 `tcp_relay` 模块转发 TCP CONNECT 流量，不依赖 REM。域名由 Implant 侧解析，Client 不会提前解析目标地址。

同一个 Session 可以启动多个监听器，它们共享一个 `tcp_relay` 任务。每个监听器都要求用户名和密码认证。

## 使用

在 Implant 交互上下文中启动监听器：

```text
socks5 start --port 1080 --user admin --pass secret
```

查看和停止监听器：

```text
socks5 list
socks5 list --all
socks5 stop --port 1080
socks5 stop --all
```

## 配置

该功能没有 Server YAML 配置项，运行时通过命令参数配置：

| 参数 | 默认值 | 含义 |
| --- | --- | --- |
| `--bind` | `127.0.0.1` | Client 本地监听地址 |
| `--port` | `1080` | Client 本地监听端口 |
| `--user` | 无 | SOCKS5 用户名，必填 |
| `--pass` | 无 | SOCKS5 密码，必填 |

目标 Session 必须加载 `tcp_relay` 模块。除非确实需要远程访问代理，否则应保留默认回环地址。

## 示例

为 ProxyChains 配置本地代理：

```text
socks5 127.0.0.1 1080 admin secret
```

启动两个共享同一 Session relay 的监听器：

```text
socks5 start --port 1080 --user admin --pass secret
socks5 start --port 1081 --user operator --pass secret
```
