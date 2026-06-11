# 参考手册

本目录包含自动生成的参考文档和 SDK 使用指南，用于查阅完整的命令、API 和接口定义。

!!! info "自动生成"
    `commands/` 和 `lua-api/` 下的文档由代码自动生成，请勿手动编辑。

    生成命令：
    ```bash
    go run ./client/cmd/genhelp/    # 生成命令参考
    go run ./client/cmd/genlua/     # 生成 Lua API 参考
    ```

## 命令参考

自动生成的完整命令手册，包含所有命令的用法、参数和示例。

- [Client 命令参考](commands/client.md) - Client 上下文全部命令
- [Implant 命令参考](commands/implant.md) - Implant 上下文全部命令
- [Community 插件命令](commands/community.md) - 内置社区插件命令

## Lua API 参考

自动生成的 MAL 插件 Lua API 定义。

- [Builtin API](lua-api/builtin.md) - 内置函数 API
- [RPC API](lua-api/rpc.md) - gRPC 方法 API
- [Beacon API](lua-api/beacon.md) - CobaltStrike 兼容 API

## SDK 使用指南

SDK 文档位于开发文档中：[SDK 文档](../development/sdk/)。

## 相关文档

- [操作指南](../operations/) - 使用引导与操作手册
- [MAL 插件开发](../development/mals/) - 插件开发文档
- [核心概念](../concept.md) - 架构与协议边界
