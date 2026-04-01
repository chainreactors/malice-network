# Malice Network 路线图

本文档概述项目方向和主要里程碑。

## 当前方向

### 1. 更丰富的传输与互联互通

目标不是只做点对点 C2，而是继续扩展多种 pipeline 形态、分布式 listener 和更灵活的桥接方式。

### 2. 更好的 GUI 与操作体验

项目已经具备 CLI / TUI / GUI 多种入口，后续重点仍然会放在可用性、交互一致性和新手上手成本上。

### 3. 更完整的插件生态

MAL、armory、alias、extension 和 bridge 型集成会继续增强，尽量把自动化、插件化和第三方生态整合到同一条工作流里。

### 4. 更轻的构建链路

继续优化 `docker`、`action`、`saas` 等构建方式，让 operator 不必在本地准备一整套复杂交叉编译环境。

## 历史里程碑

### v0.0.1

- 打通基础的 client / server / listener / implant 交互
- 建立 session、task、pipeline、event 等核心模型
- 落地第一批 implant 基础命令和执行模块

### v0.0.2

- client 端重构，增强交互体验
- 加入 CI / CD 与更可控的构建流程
- MAL 插件能力开始成形

### v0.0.3

- listener parser 和第三方兼容能力继续增强
- build、profile、explorer、module 等链路明显完善
- implant 模块覆盖面扩大到 service / registry / taskschd / token 等

### v0.0.4

- 持续修复兼容性问题
- 加强 loader、TLS、kit、shellcode 相关能力
- 进一步补强工程化和可交付性

### v0.1.x

- 更强调开箱即用
- 更强调 GUI、代理、插件和 OPSEC 组合能力
- 更强调“控制面 + 构建面 + 插件面”一体化

## 说明

- 这是方向文档，不是对具体发布日期的承诺
- 历史描述保留了 IoM 阶段的术语背景
- 当前仓库的使用与实现说明以 `docs/` 中的架构、Listener、Build 和命令文档为准
