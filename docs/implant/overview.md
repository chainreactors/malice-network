# Implant 概览

本文档说明 Implant 家族及其与 `malice-network` 的关系。

## 1. 仓库边界

`malice-network` 不是 implant 源码主仓库。

当前默认 implant 家族是：

- <https://github.com/chainreactors/malefic>

本仓库负责的是：

- server / client / listener 控制面
- profile 与 artifact 构建编排
- artifact 下载与管理
- implant 会话接入后的运行时控制

## 2. 主要组件

默认 implant 家族的主要组件如下：

| 组件 | 作用 |
| --- | --- |
| `malefic` | 主 implant，本体能力最完整 |
| `malefic-mutant` | 负责配置、变体生成、patch、格式转换 |
| `malefic-pulse` | 更轻量的上线模板 |
| `malefic-prelude` | 多阶段上线里的中间阶段 |
| `malefic-srdi` | 与 SRDI 相关的能力组件 |
| `malefic-modules` | implant 功能模块实现集合 |

## 3. 集成链路

典型集成链路如下：

1. 在 `malice-network` 的 client 中准备 pipeline 和 profile
2. 通过 `build` 命令选择 `docker`、`action` 或 `saas`
3. Server 侧调度构建流程
4. 生成的 artifact 被记录并下载
5. Implant 上线后，由 server、listener 和 client 接管会话与任务

排查问题时可按以下分层判断：

- 上线失败，是 listener / pipeline / parser 问题
- 任务执行失败，是 RPC / session / module / implant 问题
- 产物生成失败，是 profile / build source / mutant / target 问题

## 4. 相关文档

相关文档：

- 控制面架构: [../architecture.md](../architecture.md)
- 快速开始: [../getting-started.md](../getting-started.md)
- 构建链路: [../server/build.md](../server/build.md)
- Listener / Pipeline: [../server/listeners.md](../server/listeners.md)

## 5. 相关仓库

详细实现位于相关仓库：

- `malefic`
- 相关 target / kit / module 仓库

Implant 的模块实现、feature 组合、目标平台支持和 mutant 细节不全部位于当前仓库。
