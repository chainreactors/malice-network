---
title: Session 父子关系
---

# Session 父子关系

## 概述

Session Link 用于在控制面中人工维护 Session 的父子拓扑。每个子 Session 最多有一个父 Session，一个父 Session 可以有多个子 Session，因此整体关系是一组树。

当前实现只记录逻辑关系：

- 关系来源固定为 `manual`。
- 在线和离线 Session 都可以参与绑定，但已经删除的 Session 不可以。
- 设置关系不会修改 Implant 通信路径，也不会改变 Task 的实际下发路由。
- REM 或 Implant 信道不会自动创建、更新或删除关系；自动发现将在后续信道集成中实现。

服务端会拒绝 Session 自关联、父子节点不存在以及会形成环路的关系。删除 Session 时，它作为父节点或子节点的关系都会被清理。

## 用法

列出全部关系：

```bash
session link
session link list
```

按父节点或子节点过滤：

```bash
session link list --parent <session_id>
session link list --child <session_id>
```

设置父子关系：

```bash
session link set --parent <parent_session_id> --child <child_session_id>
```

如果子 Session 已经有父节点，`set` 会直接将它重新挂载到新父节点。`reparent` 是同一命令的别名：

```bash
session link reparent --parent <new_parent_session_id> --child <child_session_id>
```

解除子 Session 的父关系：

```bash
session link unlink --child <child_session_id>
```

解除关系后，该 Session 成为新的根节点；它已有的子节点关系保持不变。

## 配置

Session Link 不需要额外配置。Server 启动时会通过数据库迁移创建 `session_links` 表，并持久化当前关系。`child_session_id` 是唯一键，保证一个子 Session 不会同时拥有多个父节点。

## 示例

建立 `session-b` 通过 `session-a` 上线的逻辑关系：

```bash
session link set --parent session-a --child session-b
```

将 `session-b` 改挂到 `session-c`：

```bash
session link reparent --parent session-c --child session-b
```

查看 `session-c` 的直接子节点并解除 `session-b` 的父关系：

```bash
session link list --parent session-c
session link unlink --child session-b
```
