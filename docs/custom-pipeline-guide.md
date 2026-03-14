# CustomPipeline 开发手册

## 1. 概述

### 什么是 CustomPipeline

malice-network 原生支持 TCP/UDP/HTTP/HTTPS 等内置 Pipeline 类型。CustomPipeline（`Pipeline_Custom`）是一种扩展机制，允许**外部进程**通过 `ListenerRPC` gRPC 接口向 C2 服务端注册自定义的 Pipeline，并自行管理会话（session）的生命周期与任务分发。

这使得任何能说 gRPC 的程序——无论是 LLM 代理、MCP 服务器还是其他自定义桥接——都可以将自身管理的会话暴露为 C2 客户端可见的 implant session，无需修改 server 或 implant 代码。

### 架构总览

```
┌──────────────────────────────────────────────────────────┐
│                     C2 Server (malice-network)           │
│  ┌──────────────────┐  ┌──────────────┐                  │
│  │  ListenerRPC      │  │   EventBus   │                  │
│  │  (gRPC Service)   │  │              │                  │
│  └────────┬─────────┘  └──────┬───────┘                  │
└───────────┼────────────────────┼─────────────────────────┘
            │ mTLS               │
            │                    │
┌───────────┼────────────────────┼─────────────────────────┐
│  Custom Pipeline Process (e.g. CLIProxyAPI Bridge)       │
│           │                    │                          │
│  ┌────────▼─────────┐  ┌──────▼───────┐                  │
│  │  SpiteStream      │  │  JobStream   │                  │
│  │  (双向 gRPC 流)    │  │  (控制流)     │                  │
│  └────────┬─────────┘  └──────────────┘                  │
│           │                                              │
│  ┌────────▼─────────┐                                    │
│  │  Session Manager  │  ← 管理本地会话                    │
│  │  (任务注入/结果收集)│                                    │
│  └──────────────────┘                                    │
└──────────────────────────────────────────────────────────┘
```

**数据流：**

1. Bridge 通过 mTLS 连接到 C2 Server 的 ListenerRPC
2. 注册 Listener → 注册 Pipeline（`Pipeline_Custom`）→ 打开 JobStream → StartPipeline → 打开 SpiteStream
3. 本地会话创建时，通过 `Register` RPC 将会话注册为 C2 session
4. C2 客户端下发任务 → Server 通过 SpiteStream 推送给 Bridge → Bridge 分发到本地会话 → 收集结果 → SpiteStream 回传

## 2. 前置条件

### Proto 定义

CustomPipeline 需要以下 proto 定义已存在（malice-network 已内置）：

```protobuf
// clientpb/client.proto

message CustomPipeline {
  string name        = 1;
  string listener_id = 2;
  string host        = 3;
}

message Pipeline {
  // ...
  oneof body {
    TCPPipeline  tcp    = 11;
    // ...
    CustomPipeline custom = 20;  // CustomPipeline 类型
  }
  string type = 50;
}
```

### consts 常量

确保 `consts` 包中包含以下常量：

- `consts.ModuleExecute` — 值为 `"exec"`，用于命令执行模块
- `consts.CtrlPipelineStart` / `CtrlPipelineStop` / `CtrlPipelineSync` — Pipeline 控制信号
- `consts.CtrlStatusSuccess` — 控制响应成功状态码

### 认证文件

需要一个 `listener.auth` mTLS 证书文件，通常由 C2 server 生成：

```yaml
# 配置示例
c2-bridge:
  enable: true
  auth-file: "/path/to/listener.auth"
  listener-name: "my-listener"
  listener-ip: "192.168.1.100"
  pipeline-name: "my-pipeline"
  server-addr: ""  # 可选，覆盖 auth 文件中的地址
```

## 3. 端侧开发步骤

### 3.1 建立 gRPC 连接

使用 `mtls.ReadConfig` 读取 auth 文件，获取 mTLS 凭证后建立 gRPC 连接：

```go
authCfg, err := mtls.ReadConfig(cfg.AuthFile)
if err != nil {
    return nil, err
}

addr := authCfg.Address()
if cfg.ServerAddr != "" {
    addr = cfg.ServerAddr
}

options, err := mtls.GetGrpcOptions(
    []byte(authCfg.CACertificate),
    []byte(authCfg.Certificate),
    []byte(authCfg.PrivateKey),
    authCfg.Type,
)

conn, err := grpc.DialContext(context.Background(), addr, options...)
rpc := listenerrpc.NewListenerRPCClient(conn)
```

### 3.2 注册 Listener

所有 ListenerRPC 调用需要在 gRPC metadata 中携带 `listener_id` 和 `listener_ip`：

```go
func listenerContext() context.Context {
    return metadata.NewOutgoingContext(ctx, metadata.Pairs(
        "listener_id", listenerID,
        "listener_ip", listenerIP,
    ))
}

_, err := rpc.RegisterListener(listenerContext(), &clientpb.RegisterListener{
    Name: cfg.ListenerName,
    Host: cfg.ListenerIP,
})
```

### 3.3 注册 Pipeline

关键点：使用 `Pipeline_Custom` body 类型，`Type` 字段设置为你的自定义标识（如 `"llm"`）：

```go
_, err = rpc.RegisterPipeline(listenerContext(), &clientpb.Pipeline{
    Name:       cfg.PipelineName,
    ListenerId: cfg.ListenerName,
    Enable:     true,
    Type:       "llm",                         // ← 你的自定义类型名
    Body: &clientpb.Pipeline_Custom{           // ← 必须是 Pipeline_Custom
        Custom: &clientpb.CustomPipeline{
            Name:       cfg.PipelineName,
            ListenerId: cfg.ListenerName,
            Host:       cfg.ListenerIP,
        },
    },
})
```

### 3.4 打开 Streams

有两个 gRPC 双向流需要建立：

**JobStream** — Pipeline 生命周期控制流（必须在 `StartPipeline` **之前**打开）：

```go
jobStream, err = rpc.JobStream(listenerContext())
go handleJobStream()
```

**SpiteStream** — 任务分发与结果回传流（在 `StartPipeline` **之后**打开，需要 `pipeline_id` metadata）：

```go
func pipelineContext() context.Context {
    return metadata.NewOutgoingContext(ctx, metadata.Pairs(
        "pipeline_id", pipelineID,
    ))
}

spiteStream, err = rpc.SpiteStream(pipelineContext())
```

### 3.5 启动 Pipeline

```go
_, err = rpc.StartPipeline(listenerContext(), &clientpb.CtrlPipeline{
    Name:       cfg.PipelineName,
    ListenerId: cfg.ListenerName,
})
```

> **顺序至关重要**：`StartPipeline` 会通过 JobStream 推送 `CtrlPipelineStart`，如果 JobStream 尚未打开，调用会超时或死锁。

### 3.6 处理 JobStream

收到控制消息后**必须**回复 `JobStatus`，并且**必须回传 `Job` 字段**：

```go
func handleJobStream() {
    for {
        msg, err := jobStream.Recv()
        if err != nil {
            return
        }

        switch msg.Ctrl {
        case consts.CtrlPipelineStart:
            log.Info("pipeline start acknowledged")
        case consts.CtrlPipelineStop:
            log.Info("pipeline stop requested")
        case consts.CtrlPipelineSync:
            log.Info("pipeline sync requested")
        }

        // ⚠️ 必须回传 msg.Job，否则客户端事件通知会显示空白
        err = jobStream.Send(&clientpb.JobStatus{
            ListenerId: listenerID,
            Ctrl:       msg.Ctrl,
            CtrlId:     msg.Id,
            Status:     int32(consts.CtrlStatusSuccess),
            Job:        msg.Job,  // ← 关键！
        })
    }
}
```

### 3.7 注册会话（Session）

当你的系统产生新会话时，通过 `Register` RPC 将其注册为 C2 session：

```go
registerData := &implantpb.Register{
    Name: agentName,
    Module: []string{
        "exec",       // ← 声明支持的模块，决定客户端可用命令
    },
    Sysinfo: &implantpb.SysInfo{
        Os: &implantpb.Os{
            Name:     osName,
            Version:  osVersion,
            Arch:     arch,
            Hostname: hostname,
            Username: username,
        },
        Process: &implantpb.Process{
            Name: processName,
            Path: processPath,
        },
    },
}

_, err := rpc.Register(listenerContext(), &clientpb.RegisterSession{
    SessionId:    sessionID,
    PipelineId:   pipelineID,
    ListenerId:   listenerID,
    RegisterData: registerData,
    Target:       "llm-agent://" + agentName,
})
```

#### Module 声明

`Module` 列表决定了客户端对该 session 可用的命令：

| Module | 对应客户端命令 |
|--------|---------------|
| `"exec"` | `execute`、shell 命令执行 |
| `"ls"` | `ls` 文件列表 |
| `"cd"` | `cd` 目录切换 |
| `"pwd"` | `pwd` 当前目录 |
| `"cat"` | `cat` 文件读取 |
| `"upload"` | 文件上传 |
| `"download"` | 文件下载 |

只声明你实际能处理的模块。未声明的模块对应的客户端命令会被隐藏或报错。

### 3.8 接收与分发 C2 任务

从 SpiteStream 接收任务请求，分发到本地会话，收集结果后回传：

```go
func handleSpiteRecv() {
    for {
        req, err := spiteStream.Recv()
        if err != nil {
            return
        }

        sessionID := req.GetSession().GetSessionId()
        spite := req.GetSpite()
        if spite == nil || sessionID == "" {
            continue
        }

        var taskID uint32
        if t := req.GetTask(); t != nil {
            taskID = t.GetTaskId()
        }

        switch spite.Name {
        case consts.ModuleExecute:
            if exec := spite.GetExecRequest(); exec != nil {
                cmd := extractCommand(exec.Path, exec.Args)
                go executeAndForward(sessionID, taskID, cmd)
            }
        default:
            if r := spite.GetRequest(); r != nil {
                cmd := spite.Name
                if len(r.Args) > 0 {
                    cmd += " " + strings.Join(r.Args, " ")
                }
                go executeAndForward(sessionID, taskID, cmd)
            }
        }
    }
}
```

### 3.9 回传任务结果

将执行结果封装为 `ExecResponse` 通过 SpiteStream 发回：

```go
func forwardResult(sessionID string, taskID uint32, stdout []byte, exitCode int32) {
    spite := &implantpb.Spite{
        Name: consts.ModuleExecute,
        Body: &implantpb.Spite_ExecResponse{
            ExecResponse: &implantpb.ExecResponse{
                Stdout:     stdout,
                StatusCode: exitCode,
                End:        true,  // ← 标记结果完成
            },
        },
    }

    spiteStream.Send(&clientpb.SpiteResponse{
        ListenerId: listenerID,
        SessionId:  sessionID,
        TaskId:     taskID,
        Spite:      spite,
    })
}
```

### 3.10 心跳保活

注册的 session 需要定期 checkin，否则 server 会标记为离线：

```go
func checkinLoop() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            for _, sessionID := range registeredSessions {
                rpc.Checkin(listenerContext(), &implantpb.Ping{
                    Nonce: int32(time.Now().Unix() & 0x7FFFFFFF),
                })
            }
        case <-ctx.Done():
            return
        }
    }
}
```

### 3.11 转发 Observe 事件（可选）

如果你的系统有实时事件流（如 LLM 对话流），可以通过 SpiteStream 转发自定义事件：

```go
func forwardObserveEvent(event *ObserveEvent) {
    spite := &implantpb.Spite{
        Name: "llm.observe",
        Body: &implantpb.Spite_Common{
            Common: &implantpb.CommonBody{
                Name:        event.Type,
                StringArray: []string{event.Format, event.SessionID},
                BytesArray:  [][]byte{[]byte(event.RawJSON)},
            },
        },
    }

    spiteStream.Send(&clientpb.SpiteResponse{
        ListenerId: listenerID,
        SessionId:  event.SessionID,
        Spite:      spite,
    })
}
```

## 4. 常见坑与解决方案

### 4.1 JobStatus 必须回传 `Job` 字段

**现象**：Pipeline 启动成功，但客户端收到的事件通知中 pipeline 信息为空白。

**原因**：`JobStream.Send` 回复 `JobStatus` 时没有设置 `Job` 字段。Server 端的事件系统依赖这个字段来填充事件详情。

**解决**：

```go
// ✗ 错误
jobStream.Send(&clientpb.JobStatus{
    ListenerId: listenerID,
    Ctrl:       msg.Ctrl,
    Status:     int32(consts.CtrlStatusSuccess),
})

// ✓ 正确 — 回传原始 msg.Job
jobStream.Send(&clientpb.JobStatus{
    ListenerId: listenerID,
    Ctrl:       msg.Ctrl,
    CtrlId:     msg.Id,
    Status:     int32(consts.CtrlStatusSuccess),
    Job:        msg.Job,  // ← 必须
})
```

### 4.2 Module 列表决定客户端可用命令

**现象**：注册 session 后，客户端 `use <session>` 后发现大部分命令不可用。

**原因**：`Register` 时 `Module` 列表为空或不完整。

**解决**：在 `implantpb.Register.Module` 中声明所有你能处理的模块名。最少应包含 `"exec"` 以支持基础命令执行。

### 4.3 `Pipeline.Type` 应使用自定义字符串

**现象**：Pipeline 类型显示为 `"custom"` 而非预期的 `"llm"` 等自定义名称。

**原因**：注册时 `Pipeline.Type` 被设置为 `"custom"` 或空字符串。

**解决**：`Pipeline.Type` 应使用你的自定义标识字符串（如 `"llm"`、`"mcp"` 等），而非 `"custom"`。`Pipeline_Custom` 是 protobuf oneof body 类型，`Type` 是独立的字符串字段。

```go
&clientpb.Pipeline{
    Type: "llm",                          // ← 你的自定义类型名
    Body: &clientpb.Pipeline_Custom{...}, // ← protobuf body 类型
}
```

### 4.4 LLM 代理的 Tool Output 解析

**现象**：命令执行结果中混入了 LLM 代理的元数据（如 "Exit code: 0", "Wall time: 1 seconds"），导致 C2 客户端显示冗余信息。

**原因**：LLM 代理（如 Claude Code、Codex CLI）返回的 tool 执行结果通常包含元数据头部，而非纯 stdout。

**解决**：实现 output 解析函数，剥离元数据并提取实际输出：

```go
func parseToolOutput(raw string) *implantpb.ExecResponse {
    resp := &implantpb.ExecResponse{}
    lines := strings.Split(raw, "\n")

    // 检测是否包含元数据
    hasMetadata := false
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if exitCodeRe.MatchString(trimmed) ||
            strings.HasPrefix(strings.ToLower(trimmed), "wall time:") ||
            trimmed == "Output:" {
            hasMetadata = true
            break
        }
    }

    if !hasMetadata {
        resp.Stdout = []byte(raw)
        return resp
    }

    // 解析元数据行，提取 exit code 和实际输出
    var outputLines []string
    inOutput := false
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        if inOutput {
            outputLines = append(outputLines, line)
            continue
        }
        if m := exitCodeRe.FindStringSubmatch(trimmed); m != nil {
            code, _ := strconv.Atoi(m[1])
            resp.StatusCode = int32(code)
            continue
        }
        if trimmed == "Output:" {
            inOutput = true
            continue
        }
        // ...
    }
    resp.Stdout = []byte(strings.Join(outputLines, "\n"))
    return resp
}
```

### 4.5 并发任务路由（FIFO inflight 队列模式）

**现象**：多个任务同时下发到同一 session 时，结果可能错配——task A 的结果被错误地关联到 task B。

**原因**：简单的"注入命令 → 等待下一个结果"模式在并发场景下无法保证任务-结果的对应关系。

**解决**：使用 FIFO inflight 队列模式：

1. 每个命令分配唯一 `commandID` 并关联 `taskID`
2. 命令入队时携带 taskID
3. 结果收集时通过 taskID 匹配
4. 每个 task 启动独立 goroutine 订阅结果通道，按 taskID 过滤

```go
// 每个任务启动独立的等待协程
func (b *Bridge) injectCommand(sessionID string, taskID uint32, cmd string) {
    cmdID := generateCommandID()
    pendingCmd := &PendingCommand{
        ID:     cmdID,
        TaskID: taskID,
        // ...
    }
    enqueueCommand(sessionID, pendingCmd)
    go b.waitAndForwardResult(sessionID, taskID)
}

// 等待协程通过 subscribe 机制过滤自己的 taskID
func (b *Bridge) waitAndForwardResult(sessionID string, taskID uint32) {
    subID := fmt.Sprintf("bridge-task-%d", taskID)
    ch := subscribe(sessionID, subID)
    defer unsubscribe(sessionID, subID)

    for result := range ch {
        if result.TaskID != taskID {
            continue  // 不是我的结果，跳过
        }
        // 转发结果...
        return
    }
}
```

### 4.6 JobStream 必须在 StartPipeline 之前打开

**现象**：`StartPipeline` 调用挂起或超时。

**原因**：`StartPipeline` 会通过 JobStream 推送 `CtrlPipelineStart` 消息并等待响应。如果 JobStream 尚未建立，消息无处投递。

**解决**：严格遵循启动顺序：

```
RegisterListener → RegisterPipeline → JobStream (open + goroutine) → StartPipeline → SpiteStream
```

## 5. 完整示例

CLIProxyAPI 项目的 `internal/bridge/` 包实现了一个完整的 LLM-to-C2 桥接，可作为参考：

| 文件 | 职责 |
|------|------|
| [`bridge.go`](../internal/bridge/bridge.go) | Bridge 结构体定义、gRPC 连接建立、完整启动生命周期（`Start` 方法） |
| [`commands.go`](../internal/bridge/commands.go) | SpiteStream 接收循环、命令分发（`exec` / module request / tool call）、tool output 解析 |
| [`register.go`](../internal/bridge/register.go) | 会话注册（`onNewSession`）、User-Agent 解析、Module 声明 |
| [`forward.go`](../internal/bridge/forward.go) | 结果转发（`waitAndForwardResult`）、observe 事件转发 |
| [`jobs.go`](../internal/bridge/jobs.go) | JobStream 处理循环、控制消息应答 |
| [`watcher.go`](../internal/bridge/watcher.go) | 会话事件观察（`observeSession`）、Checkin 心跳循环 |

### 启动序列总结

```go
// 1. 建立 mTLS gRPC 连接
conn := grpc.DialContext(ctx, addr, tlsOptions...)
rpc := listenerrpc.NewListenerRPCClient(conn)

// 2. 注册 Listener
rpc.RegisterListener(listenerCtx, &clientpb.RegisterListener{...})

// 3. 注册 Pipeline (Pipeline_Custom body, 自定义 Type)
rpc.RegisterPipeline(listenerCtx, &clientpb.Pipeline{
    Type: "llm",
    Body: &clientpb.Pipeline_Custom{Custom: &clientpb.CustomPipeline{...}},
})

// 4. 打开 JobStream 并启动处理协程
jobStream = rpc.JobStream(listenerCtx)
go handleJobStream()

// 5. 启动 Pipeline（触发 CtrlPipelineStart → JobStream 应答）
rpc.StartPipeline(listenerCtx, &clientpb.CtrlPipeline{...})

// 6. 打开 SpiteStream（需要 pipeline_id metadata）
spiteStream = rpc.SpiteStream(pipelineCtx)

// 7. 启动 SpiteStream 接收循环
go handleSpiteRecv()

// 8. 启动 Checkin 心跳
go checkinLoop()

// 9. 注册已有会话 + 监听新会话
for _, sess := range existingSessions {
    go registerSession(sess)
}
onNewSession = func(sess) { go registerSession(sess) }
```

## 6. 扩展：添加新 Pipeline 类型

CustomPipeline 机制的设计目标是**零 server 代码修改**。只要你的外部进程遵循上述协议，即可注册任意类型的 pipeline。

### 示例：MCP Pipeline

假设你想将 MCP (Model Context Protocol) 服务器桥接到 C2：

```go
// 只需改变 Type 和业务逻辑，协议流程完全相同
_, err = rpc.RegisterPipeline(listenerContext(), &clientpb.Pipeline{
    Name:       "mcp-bridge",
    ListenerId: "mcp-listener",
    Enable:     true,
    Type:       "mcp",  // ← 自定义类型标识
    Body: &clientpb.Pipeline_Custom{
        Custom: &clientpb.CustomPipeline{
            Name:       "mcp-bridge",
            ListenerId: "mcp-listener",
            Host:       "127.0.0.1",
        },
    },
})
```

### 添加新类型的 Checklist

端侧（你的进程）：

- [ ] 实现 gRPC mTLS 连接
- [ ] 实现完整的启动序列（见第 3 节）
- [ ] 实现命令接收与分发
- [ ] 实现结果回传
- [ ] 实现心跳保活
- [ ] 处理 tool output 格式差异
