# MAL 插件快速开始

## 简介

MAL 基于 [lua5.1](https://www.lua.org/manual/5.1/) 和 [gopher-lua](https://github.com/yuin/gopher-lua) 实现的插件系统，提供：

- 类似 AggressiveScript 的简化 API
- 完整的原始 gRPC API
- 内置 Lua 库集合

Lua 是简单的脚本语言，推荐使用 [VSCode](https://code.visualstudio.com/) + [Lua 插件](https://marketplace.visualstudio.com/items?itemName=sumneko.lua) 开发。

## 基础使用

### Hello World

编写 `hello.lua`：

```lua
print("hello world") -- 在 client 标准输出打印

broadcast("hello world") -- 在所有 client 打印

notify("hello world") -- 在所有 client 打印，并发送第三方通知（飞书、Telegram、微信）
```

编写 `mal.yaml` 配置：

```yaml
name: hello
type: lua
author: your-name
version: v0.0.1
entry: hello.lua
```

将文件放入插件目录：

- Windows: `%USERPROFILE%/.config/malice/mals/hello/`
- Linux / macOS: `~/.config/malice/mals/hello/`

加载插件：

```bash
mal load hello
```

### 分发插件

[社区仓库](https://github.com/chainreactors/mal-community) 中的插件通过 zip 包分发。

将 `hello.lua` 和 `mal.yaml` 打包为 zip 后安装：

```bash
mal install hello.zip
```

## 注册命令

通过 `command` 函数将 Lua 函数注册为 client 命令。

### 基本语法

```lua
command(name, function, help, ttp)
```

参数说明：
- `name`: 命令名称，支持多级命令（用 `:` 分隔）
- `function`: 执行的 Lua 函数
- `help`: 命令帮助信息
- `ttp`: MITRE ATT&CK TTP 编号

详细文档：[builtin.md](builtin.md#command)

### 基础用法

#### 简单命令注册

```lua
local function hello()
    print("hello world")
end

command("hello", hello, "print hello world", "T1000")
```

执行 `hello` 命令时调用 `hello` 函数并输出 "hello world"。

#### 多级命令注册

```lua
local function logonpasswords()
    -- 实现
end

local function tickets()
    -- 实现
end

command("mimikatz:logonpasswords", logonpasswords, "logonpasswords", "T1000")
command("mimikatz:tickets", tickets, "tickets", "T1000")
```

`:` 表示子命令分隔符，可添加任意层级：

```lua
command("mimikatz:common:hello", hello, "print hello world", "T1000")
```

### 参数处理

命令行输入分为：
- `args`: 位置参数，如 `hello arg1 arg2`
- `flags`: 标志参数，如 `hello -s` 或 `hello --long`
- `cmdline`: 完整命令行字符串
- `cmd`: [Cobra](https://cobra.dev/) 命令对象，用于手动解析

#### args 参数

```lua
local function print_args(args)
    print(args[1])
    print(args[2])
end

command("hello", print_args, "print args", "T1000")
```

执行 `hello arg1 arg2` 时输出 `arg1` 和 `arg2`。注意 Lua 数组索引从 1 开始。

**`arg_<number>` 形式：**

```lua
local function print_args(arg_1, arg_2, arg_3)
    print(arg_1)
    print(arg_2)
    print(arg_3)
end

command("hello", print_args, "print args", "T1000")
```

执行 `hello arg1 arg2 arg3` 时分别输出三个参数。适用于参数数量固定且较少的场景。

#### flags 参数

```lua
local function print_flags(cmd)
    local name = cmd:Flags():GetString("name")
    print(name)
end

local cmd = command("hello", print_flags, "print flags", "T1000")
cmd:Flags():String("name", "", "the name to print")
```

执行 `hello --name flag1` 时输出 `flag1`。适用于参数较多或需要精确控制的场景。

**`flag_<name>` 自动注册：**

```lua
local function print_flags(flag_name)
    print(flag_name)
end

command("hello", print_flags, "print flags", "T1000")
```

使用 `flag_<name>` 格式的参数会自动注册为标志参数。执行 `hello --name flag1` 时输出 `flag1`。

#### cmdline 参数

```lua
local function print_cmdline(cmdline)
    print(cmdline)
end

command("hello", print_cmdline, "print cmdline", "T1000")
```

`cmdline` 参数将所有命令行参数以空格分隔拼接为字符串。执行 `hello arg1 arg2 arg3` 时输出 `arg1 arg2 arg3`。

### 辅助函数

增强命令功能：

- `help("hello", "a description for this command")` - 添加详细帮助
- `example("hello", "a example for this command")` - 添加命令示例
- `opsec("hello", 9.8)` - 添加 OPSEC 评分

添加自动补全：

```lua
local rem_socks_cmd = command("rem_community:socks5", run_socks5, "serving socks5 with rem", "T1090")
bind_args_completer(rem_socks_cmd, { rem_completer() })
```

## 标准库与内置库

### MAL Package

MAL 将 API 分为三个 package：

- [builtin](builtin.md) - 直接可用的核心 API
- [rpc](rpc.md) - gRPC 相关 API 的 Lua 实现
- [beacon](beacon.md) - CobaltStrike 兼容层 API，实现 AggressiveScript 中 `b` 开头函数

### Lua 标准库

支持 Lua 5.1 标准库：

- package
- table
- io
- os
- string
- math
- debug
- channel
- coroutine

异步/并发文档：[gopher-lua](https://github.com/yuin/gopher-lua)

### Lua 扩展库

已导入的常用扩展库：

- [argparse](https://github.com/vadv/gopher-lua-libs/tree/master/argparse/) - CLI 参数解析
- [base64](https://github.com/vadv/gopher-lua-libs/tree/master/base64/) - Base64 编解码
- [cmd](https://github.com/vadv/gopher-lua-libs/tree/master/cmd/) - 命令执行
- [db](https://github.com/vadv/gopher-lua-libs/tree/master/db/) - 数据库访问
- [filepath](https://github.com/vadv/gopher-lua-libs/tree/master/filepath/) - 路径操作
- [goos](https://github.com/vadv/gopher-lua-libs/tree/master/goos/) - 操作系统接口
- [humanize](https://github.com/vadv/gopher-lua-libs/tree/master/humanize/) - 人性化格式
- [inspect](https://github.com/vadv/gopher-lua-libs/tree/master/inspect/) - 对象打印
- [ioutil](https://github.com/vadv/gopher-lua-libs/tree/master/ioutil/) - IO 工具
- [json](https://github.com/vadv/gopher-lua-libs/tree/master/json/) - JSON 处理
- [log](https://github.com/vadv/gopher-lua-libs/tree/master/log/) - 日志
- [plugin](https://github.com/vadv/gopher-lua-libs/tree/master/plugin/) - 插件加载
- [regexp](https://github.com/vadv/gopher-lua-libs/tree/master/regexp/) - 正则表达式
- [shellescape](https://github.com/vadv/gopher-lua-libs/tree/master/shellescape/) - Shell 转义
- [stats](https://github.com/vadv/gopher-lua-libs/tree/master/stats/) - 统计
- [storage](https://github.com/vadv/gopher-lua-libs/tree/master/storage/) - 持久化存储
- [strings](https://github.com/vadv/gopher-lua-libs/tree/master/strings/) - 字符串操作（UTF-8 支持）
- [tcp](https://github.com/vadv/gopher-lua-libs/tree/master/tcp/) - TCP 客户端
- [template](https://github.com/vadv/gopher-lua-libs/tree/master/template/) - 模板引擎
- [time](https://github.com/vadv/gopher-lua-libs/tree/master/time/) - 时间处理
- [yaml](https://github.com/vadv/gopher-lua-libs/tree/master/yaml/) - YAML 处理
- [http](https://github.com/cjoudrey/gluahttp) - HTTP 客户端
- [crypto](https://github.com/tengattack/gluacrypto) - 加密（MD5、SHA1、SHA256、HMAC、AES）

通过 `require` 引入依赖：

```lua
local crypto = require("crypto")
```

## 插件架构

### 插件组成

| 组件 | 说明 | 必需 |
|------|------|------|
| **mal.yaml** | 插件元数据配置 | ✅ |
| **entry 脚本** | Lua 入口文件 | ✅ |
| **资源文件** | 二进制/配置文件 | ❌ |
| **依赖库** | 其他 MAL 库 | ❌ |

### 插件目录结构

mal-community 仓库中的插件遵循统一的目录结构：

```
community-elevate/          # 提权工具包
├── mal.yaml               # 插件配置文件
├── main.lua              # 入口脚本
├── resources/            # 资源文件目录
│   ├── windows/         # Windows 平台资源
│   │   ├── x64/        # 64 位程序
│   │   └── x86/        # 32 位程序
│   └── linux/          # Linux 平台资源
└── lib/                 # 依赖库文件
    └── utils.lua       # 工具函数

community-domain/          # 域渗透工具包
├── mal.yaml              # 插件配置文件
├── main.lua             # 入口脚本
├── modules/             # 功能模块
│   ├── kerberos.lua    # Kerberos 相关
│   ├── ldap.lua        # LDAP 查询
│   └── dcsync.lua      # DCSync 功能
├── resources/           # 资源文件
│   └── tools/          # 工具二进制
└── config/             # 配置文件
    └── targets.yaml    # 目标配置
```

典型的 `mal.yaml` 配置：

```yaml
name: community-elevate
type: lua
author: chainreactors
version: v1.0.0
entry: main.lua
description: Windows/Linux 提权工具集合
depend_module:
  - lib
  - common
resources:
  - resources/
tags:
  - elevate
  - privilege
  - windows
  - linux
```

### API 分层

MAL 提供三层 API 体系：

| API 层 | 用途 | 特点 | 适用场景 |
|-------|------|------|----------|
| **Builtin** | 核心功能 | 简单直观 | 常规操作 |
| **Beacon** | CS 兼容 | AggressorScript 风格 | CS 迁移 |
| **RPC** | 完整功能 | 原始 gRPC | 高级操作 |

## 高级用法

### Beacon Package

Beacon package 按照 CobaltStrike 的 AggressiveScript API 签名封装，提供类似的编写体验。

```lua
local beacon = require("beacon")

beacon.bexecute(active(), "whoami")
```

支持的所有 AggressiveScript 风格 API 文档：[beacon.md](beacon.md)

### 创建 Protobuf Message

在 builtin 和 beacon 包中，绝大多数 API 都是高度封装的。但调用 rpc 包中的接口时，需要手动创建对应的 protobuf message。

所有 protobuf message 都已注册到 Lua 中：

```lua
local bin = ExecuteBinary.New()
bin.Name = "execute_assembly"
bin.Args = {"whoami"}
bin.Bin = read_resource("example.exe")
```

或使用构造函数：

```lua
local bin = ExecuteBinary.New({
    Name = "execute_assembly",
    Bin = read_resource("example.exe"),
    Type = "example_type",
    Args = {"whoami"}
})
```

### 动态创建 Protobuf Message

使用 `ProtobufMessage` 通过反射获取 package name：

```lua
local msg = ProtobufMessage.New("modulepb.ExecuteBinary", {
    Name = "execute_assembly",
    Bin = read_resource("example.exe"),
    Type = "example_type",
    Args = {"whoami"}
})
```

### 调用 RPC 命令

```lua
function load_rem()
    local rpc = require("rpc")

    local task = rpc.LoadRem(active():Context(), ProtobufMessage.New("modulepb.Request", {
        Name = "load_rem",
        Bin = read_resource("chainreactors/rem.dll"),
    }))
    wait(task)
end
```

### execute_module 动态执行

`execute_module` 提供灵活的方式在运行时动态构造请求。

#### 基本用法

使用 `spite(body)` 函数从 protobuf message 构建 Spite，通过 `execute_module` 执行：

```lua
local function run_whoami()
    local session = active()
    local req = Request.New({
        Name = "whoami"
    })
    local s = spite(req)
    return execute_module(session, s, "response")
end
```

#### 支持的 body 类型

`spite()` 函数支持多种 protobuf message 类型：

```lua
-- Request 类型
local s1 = spite(Request.New({Name = "whoami"}))

-- ExecuteBinary 类型
local s2 = spite(ExecuteBinary.New({
    Name = "execute_assembly",
    Bin = read_resource("example.exe"),
    Type = "execute_assembly",
    Args = {"arg1", "arg2"}
}))

-- 直接传入 Spite（返回自身）
local s3 = spite(existing_spite)
```

#### 使用 Callback 处理响应

`execute_module` 支持传入第 4 个参数作为 callback 函数，用于自定义响应处理逻辑。Callback 接收 `TaskContext` 参数，可访问任务、会话和响应内容。

```lua
local function run_whoami2(cmd)
    local session = active()
    local s = spite(Request.New({
        Name = "whoami2"
    }))
    -- 传入 callback 处理响应
    return execute_module(session, s, "response", function(ctx)
        -- 获取并打印输出
        local resp = ctx.Spite:GetResponse()
        print(resp.Output)
        return ctx
    end)
end

command("whoami2", run_whoami2, "Print current user", "T1033")
```

### 注册为库

MAL 允许插件作为类库，成为其他库的依赖。

在 `mal.yaml` 中设置 `lib: true`：

```yaml
name: community-lib
type: lua
author: your-name
version: v0.0.1
entry: main.lua
lib: true
```

库实现示例：

```lua
-- community-lib/main.lua

local lib = {}

function lib.demo()
    print("this is a lib demo")
end

return lib
```

在其他插件中使用：

```lua
-- community-other/main.lua

local clib = require("community-lib")
clib.demo()
```
