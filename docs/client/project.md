---
title: Project 管理
---

# Project 管理

## 概述

Project 用于记录独立的任务项目。当前阶段提供 Project 的创建、查询、更新和删除能力，但尚未将 Session、Pipeline 等资源按 Project 隔离。

Server 首次初始化数据库时会创建名为 `default` 的 Project。该 Project 不能删除。

## 使用

| 命令 | 说明 |
|------|------|
| `project` | 列出所有未删除的 Project |
| `project create <name>` | 创建 Project |
| `project get <name\|id>` | 按名称或 ID 查看详情 |
| `project update <name\|id>` | 更新名称、描述或备注 |
| `project delete <name\|id>` | 软删除 Project |
| `project delete <name\|id> --hard` | 永久删除 Project |

创建和更新命令支持以下参数：

- `--description` / `-d`：Project 描述。
- `--note` / `-n`：Project 备注。
- `project update --name`：设置新的 Project 名称。

## 配置

Project 不需要额外配置。Server 启动时通过现有数据库连接自动创建或更新 `projects` 表，并保证 `default` Project 存在。

## 示例

```bash
project create red-team-q3 --description "Q3 assessment" --note "Internal"
project
project get red-team-q3
project update red-team-q3 --name red-team-q3-final --description "Completed"
project delete red-team-q3-final
```

永久删除不可恢复：

```bash
project delete red-team-q3-final --hard
```
