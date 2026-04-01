# MAL 插件开发文档

MAL 是 `malice-network` client 侧的插件系统，基于 Lua 5.1 实现。

## 文档导航

- [快速开始](quickstart.md) - 插件开发入门指南
- [Builtin API](builtin.md) - 核心内置函数完整参考
- [Beacon API](beacon.md) - CobaltStrike 兼容层 API
- [RPC API](rpc.md) - gRPC 原始调用接口
- [Embed API](embed.md) - 嵌入式资源与高级用法

## 社区资源

- [mal-community](https://github.com/chainreactors/mal-community) - 官方插件仓库
- [mals](https://github.com/chainreactors/mals) - 插件索引仓库

## 快速链接

### 基础概念
- [Hello World](quickstart.md#hello-world)
- [注册命令](quickstart.md#注册命令)
- [参数处理](quickstart.md#参数处理)

### API 参考
- [Artifact 相关](builtin.md#artifact)
- [Session 管理](builtin.md#session)
- [文件操作](builtin.md#file)
- [进程操作](builtin.md#process)

### 高级用法
- [Protobuf Message](quickstart.md#创建-protobuf-message)
- [RPC 调用](quickstart.md#调用-rpc-命令)
- [注册为库](quickstart.md#注册为库)
