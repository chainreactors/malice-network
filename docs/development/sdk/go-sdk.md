# Go

IoM Go SDK 是 Malice Network 的官方 Golang SDK，提供完整的 gRPC 客户端功能。

> 完整文档和示例请参考：[Go SDK](https://github.com/chainreactors/IoM-go)

## 快速开始

### 安装

```bash
go get github.com/chainreactors/IoM-go
```

### 最小示例

```go
package main

import (
    "context"
    "fmt"
    "github.com/chainreactors/IoM-go/client"
    "github.com/chainreactors/IoM-go/mtls"
    "github.com/chainreactors/IoM-go/proto/client/clientpb"
)

func main() {
    // 加载配置并连接
    config, _ := mtls.ReadConfig("config.yaml")
    conn, _ := mtls.Connect(config)
    defer conn.Close()

    // 初始化服务器状态
    server, _ := client.NewServerStatus(conn, config)

    // 获取活动会话
    sessions := server.AlivedSessions()
    session := server.AddSession(sessions[0])

    // 获取任务
    tasks, _ := server.Rpc.GetTasks(context.Background(), &clientpb.TaskRequest{
        SessionId: session.SessionId,
    })

    for _, task := range tasks.Tasks {
        fmt.Printf("任务 %d: %s\n", task.TaskId, task.Type)
    }
}
```

## 核心概念

SDK 的设计完全对应 IoM 的架构概念，详细架构说明请参考 [核心概念](../../concept.md)。

### ServerStatus - 服务器状态

对应 [Client 架构](../../client/)，通过 `ServerStatus` 管理与 Server 的 gRPC 连接。

**初始化方式：**

```go
// 从配置文件加载（推荐）
config, _ := mtls.ReadConfig("config.yaml")
conn, _ := mtls.Connect(config)
server, _ := client.NewServerStatus(conn, config)

// 手动配置
config := &mtls.ClientConfig{
    Host:     "127.0.0.1",
    Port:     5004,
    Operator: "admin",
    CACert:   "...",
    Cert:     "...",
    Key:      "...",
}
```

### Session - 会话

对应 [Session](../../client/console.md)，代表一个已连接的 Implant。

```go
// 获取所有活动会话
sessions := server.AlivedSessions()

// 添加会话以便使用
session := server.AddSession(sessions[0])

// 获取特定会话
session, _ := server.GetOrUpdateSession(sessionId)
```

### Task - 任务

对应 [Task](../../client/console.md)，IoM 使用基于任务的异步执行模型。

```go
// 获取任务列表
tasks, _ := server.Rpc.GetTasks(context.Background(), &clientpb.TaskRequest{
    SessionId: session.SessionId,
})

// 注册任务完成回调
server.DoneCallbacks.Store(
    fmt.Sprintf("%s-%d", task.SessionId, task.TaskId),
    func(resp *clientpb.TaskContext) {
        fmt.Printf("任务完成: %s\n", string(resp.Spite.Body))
    },
)
```

### Context - 会话上下文

SDK 提供会话上下文管理，可添加自定义元数据。

```go
// 获取会话上下文
ctx := session.Context()

// 添加自定义值
sessionWithValue, _ := session.WithValue("key1", "value1")

// 克隆会话
sdkSession := session.Clone(consts.CalleeSDK)
```

## 常用操作

### 会话管理

```go
// 更新所有会话（包括已断开的）
server.UpdateSessions(true)

// 检查会话能力
if session.HasDepend("execute") {
    // 会话具有 execute 模块
}

// 设置活动会话
server.ActiveTarget.Set(session)
```

### 监听器管理

```go
// 获取监听器
listeners, _ := server.Rpc.GetListeners(context.Background(), &clientpb.Empty{})

// 列出管道
pipelines, _ := server.Rpc.ListPipelines(context.Background(), &clientpb.Listener{})
```

### 事件处理

```go
import "github.com/chainreactors/IoM-go/consts"

// 注册事件钩子
server.On(client.EventCondition{
    Type: consts.EventSession,
    Op:   consts.CtrlSessionRegister,
}, func(event *clientpb.Event) (bool, error) {
    fmt.Printf("新会话: %s\n", event.Session.SessionId)
    return true, nil
})

// 启动事件流
eventStream, _ := server.Rpc.Events(context.Background(), &clientpb.Empty{})
go func() {
    for {
        event, _ := eventStream.Recv()
        server.HandlerEvent(event)
    }
}()
```

## 高级功能

### 观察者模式

监控多个会话：

```go
// 添加观察者
observerId := server.AddObserver(session)

// 获取观察者日志
log := server.ObserverLog(session.SessionId)

// 完成后移除观察者
defer server.RemoveObserver(observerId)
```

### 活动目标管理

```go
// 设置活动会话
server.ActiveTarget.Set(session)

// 获取活动会话
activeSession := server.ActiveTarget.Get()

// 将会话置于后台
server.ActiveTarget.Background()
```

## 错误处理

```go
import "log"

conn, err := mtls.Connect(config)
if err != nil {
    log.Fatalf("连接失败: %v", err)
}
defer conn.Close()

session, err := server.GetOrUpdateSession(sessionId)
if err != nil {
    log.Printf("获取会话失败: %v", err)
}
```

## 最佳实践

1. **正确关闭连接** - 使用 defer 确保连接关闭
   ```go
   conn, _ := mtls.Connect(config)
   defer conn.Close()
   ```

2. **复用 ServerStatus 实例** - 避免频繁创建连接
   ```go
   server, _ := client.NewServerStatus(conn, config)
   // 复用 server 进行多次操作
   ```

3. **使用事件流** - 实时监听服务器事件
   ```go
   eventStream, _ := server.Rpc.Events(context.Background(), &clientpb.Empty{})
   go server.HandlerEvent(event)
   ```

4. **检查会话能力** - 执行前确认模块可用
   ```go
   if session.HasDepend("execute") {
       // 执行操作
   }
   ```

## 相关资源

- [Go SDK 源码](https://github.com/chainreactors/IoM-go)
- [示例代码](https://github.com/chainreactors/IoM-go/tree/master/example)
- [核心概念](../../concept.md)
- [Proto 协议定义](https://github.com/chainreactors/proto)
