# 插件体系

IoM 提供四种扩展机制，覆盖从脚本自动化到二进制能力注入的完整扩展链路，通过 Armory 统一分发。

## 架构总览

```
┌─────────────────────────────────────────────────────────┐
│                      Armory 市场                         │
│              搜索 / 安装 / 更新 / 分发                    │
└──────────┬──────────┬──────────┬──────────┬──────────────┘
           │          │          │          │
     ┌─────▼──┐ ┌─────▼──┐ ┌────▼───┐ ┌───▼────┐
     │  MAL   │ │ Alias  │ │Extension│ │ Addon  │
     │  Lua   │ │ 命令包装│ │ BOF/DLL│ │ Module │
     └───┬────┘ └───┬────┘ └───┬────┘ └───┬────┘
         │          │          │          │
    Client 侧   Implant 侧  Implant 侧  Implant 侧
    命令注册     命令注册     命令注册     模块加载
```

### 四种扩展机制

| 类型 | 运行时 | 作用域 | 加载方式 | 典型用途 |
|------|--------|--------|----------|----------|
| **MAL** | Lua 5.1（gopher-lua） | Client 侧 | 脚本动态加载 | 自动化编排、批量操作、自定义命令 |
| **Alias** | 外部二进制 | Implant 侧 | 命令包装 | 将 PE/DLL/.NET 工具注册为命令 |
| **Extension** | BOF/DLL | Implant 侧 | 反射加载 | CobaltStrike BOF 生态兼容 |
| **Addon** | Rust Module | Implant 侧 | 动态链接 | Implant 功能模块热加载 |

## MAL 插件系统

MAL (Malice Application Language) 是 Client 侧的脚本插件系统，基于 Lua 实现。

### 设计理念

- **Client 侧执行** ：MAL 脚本在 Client 进程中运行，通过 RPC 调用 Server/Implant 能力
- **命令注册** ：插件可以向命令树注册新命令，与内置命令无差别使用
- **两类来源** ：支持随 Client 编译的嵌入式插件，以及安装到用户配置目录的外部插件

### 发现机制

MAL 插件从两类位置发现：

| 路径 | 来源 | 说明 |
|------|------|------|
| `~/.config/malice/mals/<name>/` | external | 外部安装目录；只有当前 profile 的 `mals` 列表包含该名称时，启动阶段才会自动加载 |
| Client 内嵌文件（`intl.UnifiedFS`） | embedded | 随 Client 分发；默认构建包含 Community 和 Custom，`professional` 构建标签额外包含 Professional |

当前实现不会自动扫描工作目录下的 `./mals/<name>/`。`mal load <name>` 也是从
`~/.config/malice/mals/<name>/mal.yaml` 读取插件，并将名称写入当前 profile。

嵌入式插件按照 Custom → Professional → Community 的顺序注册；如果多个嵌入式插件的
名称相同，先注册的插件保留，后续同名插件跳过。外部插件由独立的 external 插件表管理。

### 命令注册层级

MAL 插件注册的命令按来源分层显示：

| 层级 | 说明 |
|------|------|
| **Custom** | 用户自定义插件注册的命令 |
| **Community** | 社区插件注册的命令 |
| **Professional** | 专业版插件注册的命令 |

外部插件注册的命令按插件名称单独分组，不属于以上三个嵌入式层级。

### Session 平台与命令可见性

Client 命令树本身支持根据命令的 `os`、`arch`、`implant` 和 `depend` annotation，
在切换 Session 时重新计算命令是否隐藏。Alias、Extension 和部分内置命令会写入这些
annotation，因此可以根据当前 Session 的系统、架构和模块动态过滤。

这些 compatibility annotation 同时也是执行约束：即使用户手工输入了已隐藏的命令，
Client 也会在进入命令 handler 前按当前或 `--use` 指定的 Session 重新校验并拒绝执行。
单独设置 Cobra 的 `Hidden` 不代表兼容性或权限限制，仍只影响帮助、补全和搜索展示。

运行期间通过 `mal load`、`mal install`、`mal update` 或 `mal refresh` 注册的外部命令，
会走与启动期相同的命令注册流程，并同步当前 Session 可见性、命令搜索索引和事件 hook。
reload、refresh 和 remove 会先注销旧命令及旧 hook，避免同名命令或回调重复残留。

多个 Client 进程各自维护运行时插件表，不会实时广播本地 profile 变化。一个终端执行
`mal load`、`mal install` 或 `mal remove` 后，其他已运行终端可执行 `mal refresh`：该命令
会先重新读取磁盘上的 `malice.yaml`，再重建本进程的外部插件、命令、事件 hook 和搜索索引；
重启 Client 也会得到相同的最新 profile 快照。

当前 MAL `command()` API 只为命令写入 `ttp` 和 `mal` annotation，没有提供正式的
目标 OS/Arch 声明接口。因此，MAL 命令不会仅因为当前 Session 是 Windows、Linux 或
其他系统而自动隐藏。插件可以在执行函数中读取 `active().Os.Name` 和
`active().Os.Arch` 并拒绝不支持的平台，但这种运行时检查不会影响命令行中的展示。

!!! tip "延伸阅读"
    
    - 插件开发 → [MAL 插件开发](../development/mal/)
    - 内置插件使用 → [嵌入式 MAL 操作指南](../operations/embed-mal.md)

## Alias 机制

Alias 将外部工具（PE、DLL、.NET 程序等）包装为 Client 命令。

### 设计理念

- **命令透明** ：安装后的 alias 在交互式命令行中与内置命令无差别
- **Implant 侧执行** ：实际执行发生在 Implant 上（通过 execute_exe / execute_assembly 等）
- **生态兼容** ：兼容 Sliver 的 alias manifest 格式

### 加载流程

```
alias manifest (JSON)
    ↓ LoadAlias
解析命令定义
    ↓ RegisterAlias
注册到 Implant 命令树
    ↓
交互式命令行可用
```

## Extension 机制

Extension 是二进制扩展插件，主要用于加载 BOF（Beacon Object File）和 DLL。

### 设计理念

- **BOF 兼容** ：直接使用 CobaltStrike 社区的 BOF 工具
- **反射加载** ：在 Implant 进程内执行，不产生新进程
- **多命令注册** ：一个 extension manifest 可以注册多个命令

### 加载流程

```
extension manifest (JSON)
    ↓ LoadExtensionManifest
解析多个 ExtCommand
    ↓ ExtensionRegisterCommand (per command)
注册到 Implant 命令树
    ↓
交互式命令行可用（bof 命令执行）
```

## Addon 机制

Addon 是 Implant 侧的 Rust 功能模块，支持运行时动态加载。

### 设计理念

- **按需加载** ：`nano` 模式 implant 只包含最小功能，通过 addon 按需扩展
- **与 Build 联动** ：模块通过 `build modules` 编译，编译后可动态加载
- **热插拔** ：不需要重新编译或重启 implant

!!! tip "操作指南"
    Addon 的编译和加载操作见 [构建操作指南](../operations/build.md) 和 [模块管理操作](../operations/post-exploitation/module-management.md)。

## Armory 分发平台

Armory 是 IoM 的插件分发平台，统一管理 MAL、Alias、Extension 的搜索、安装和更新。

### 设计理念

- **统一入口** ：一个命令管理所有类型的插件
- **GitHub 源** ：支持 GitHub 仓库作为 armory 源
- **版本管理** ：支持插件的更新和版本追踪

### 默认源

| 仓库 | 说明 |
|------|------|
| [mal-community](https://github.com/chainreactors/mal-community) | 社区 MAL 插件 |
| [mals](https://github.com/chainreactors/mals) | MAL 框架 |

可通过 `config` 配置自定义 armory 源，使用私有 GitHub 仓库分发内部插件。

## 相关文档

- [命令行系统](console.md) — 命令注册与上下文机制
- [MAL 插件开发](../development/mal/) — 完整开发文档
- [嵌入式 MAL](../operations/embed-mal.md) — 内置插件操作
- [构建操作](../operations/build.md) — Addon 模块编译
