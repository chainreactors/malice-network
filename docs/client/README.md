# Client 架构与配置

本目录包含 malice-network Client 的架构说明和配置文档。

## 文档列表

- [快速开始](quickstart.md) - Client 登录与基础使用
- [命令行系统](console.md) - CLI/TUI、MCP、LocalRPC 启动机制
- [Agent 集成](agent.md) - MCP 与 Agent 命令体系
- [插件系统](plugin.md) - Client 侧插件机制

## Client 架构

Client 是 malice-network 的操作入口，提供 CLI/TUI 交互界面，负责：

- 命令派发与会话管理
- 插件加载与本地集成
- MCP/LocalRPC 接口暴露

详细架构说明请参考 [核心概念](../concept.md)。

## 使用指南

关于 Client 的具体使用方法，请参考：

- [快速开始](quickstart.md) - Client 快速上手指南
- [操作指南](../operations/) - 完整操作手册
- [后渗透操作](../operations/post-exploitation/) - 后渗透操作指南

## 相关文档

- [核心概念](../concept.md) - 架构与协议边界
- [Server 架构](../server/) - Server 端架构与配置
- [开发文档](../development/) - 开发与扩展指南

