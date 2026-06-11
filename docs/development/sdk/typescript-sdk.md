# TypeScript

IoM TypeScript SDK 是一个共享的 gRPC 客户端库，用于 VSCode 扩展和 Web 应用开发。

> 完整文档和示例请参考：[TypeScript SDK](https://github.com/chainreactors/IoM-typescript)

## 快速开始

### 安装

```bash
npm install iom-typescript
# 或在 monorepo 工作区中
# "iom-typescript": "workspace:*"
```

### 最小示例

```typescript
import { GrpcClient, AuthConfig } from 'iom-typescript';

const client = new GrpcClient({
    onLog: (message: string) => console.log(message),
    timeout: 10,
    maxMessageSize: 50 * 1024 * 1024
});

const config: AuthConfig = {
    operator: 'admin',
    host: 'localhost',
    port: 31337,
    type: 'mtls',
    ca: 'base64_encoded_ca_cert',
    cert: 'base64_encoded_client_cert',
    key: 'base64_encoded_client_key'
};

await client.connect(config, 'config_file_name');

if (client.isConnected()) {
    const rpc = client.getRpc();
    rpc.malice.getSessions(Empty, (error, response) => {
        console.log('Sessions:', response);
    });
}

client.disconnect();
```

## 核心概念

SDK 的设计完全对应 IoM 的架构概念，详细架构说明请参考 [核心概念](../../concept.md)。

### GrpcClient - 客户端

对应 [Client 架构](../../client/)，通过 `GrpcClient` 类与 Server 进行 gRPC 通讯。

**初始化方式：**

```typescript
// 创建客户端实例
const client = new GrpcClient({
    onLog: (message: string) => console.log(message),
    timeout: 10,
    maxMessageSize: 50 * 1024 * 1024
});

// 连接配置
const config: AuthConfig = {
    operator: 'admin',
    host: '127.0.0.1',
    port: 5004,
    type: 'mtls',
    ca: '...',
    cert: '...',
    key: '...'
};

await client.connect(config, 'config_name');
```

### RPC 调用

SDK 提供两个主要的 RPC 客户端：

```typescript
const rpc = client.getRpc();

// Malice 客户端 - 会话和任务管理
rpc.malice.getSessions(Empty, callback);
rpc.malice.getTasks(request, callback);

// Listener 客户端 - 监听器管理
rpc.listener.getListeners(Empty, callback);
rpc.listener.listPipelines(request, callback);
```

### Session - 会话

对应 [Session](../../client/console.md)，代表一个已连接的 Implant。

```typescript
import { Empty } from 'iom-typescript';

rpc.malice.getSessions(Empty, (error, response) => {
    if (!error && response) {
        response.sessions.forEach(session => {
            console.log(`会话 ID: ${session.sessionId}`);
        });
    }
});
```

### Task - 任务

对应 [Task](../../client/console.md)，IoM 使用基于任务的异步执行模型。

```typescript
import { TaskRequest } from 'iom-typescript';

const request: TaskRequest = {
    sessionId: 'session_id_here'
};

rpc.malice.getTasks(request, (error, response) => {
    if (!error && response) {
        response.tasks.forEach(task => {
            console.log(`任务 ${task.taskId}: ${task.type}`);
        });
    }
});
```

## 常用操作

### 连接管理

```typescript
// 检查连接状态
if (client.isConnected()) {
    // 执行操作
}

// 断开连接
client.disconnect();
```

### 会话操作

```typescript
// 获取所有会话
rpc.malice.getSessions(Empty, (error, response) => {
    if (error) {
        console.error('获取会话失败:', error);
    } else {
        console.log('会话列表:', response.sessions);
    }
});
```

### 监听器操作

```typescript
// 获取监听器
rpc.listener.getListeners(Empty, (error, response) => {
    if (!error) {
        console.log('监听器:', response.listeners);
    }
});

// 列出管道
rpc.listener.listPipelines(listenerRequest, (error, response) => {
    if (!error) {
        console.log('管道:', response.pipelines);
    }
});
```

## 在不同环境中使用

### VSCode 扩展

```typescript
import * as vscode from 'vscode';
import { GrpcClient, AuthConfig } from 'iom-typescript';

export class ExtensionGrpcClient {
    private client: GrpcClient;

    constructor(outputChannel: vscode.OutputChannel) {
        this.client = new GrpcClient({
            onLog: (message: string) => outputChannel.appendLine(message)
        });
    }

    async connect(config: AuthConfig) {
        return this.client.connect(config, 'vscode-extension');
    }

    getRpc() {
        return this.client.getRpc();
    }

    disconnect() {
        this.client.disconnect();
    }
}
```

### Next.js / React

```typescript
'use client';
import { useState, useEffect } from 'react';
import { GrpcClient, AuthConfig } from 'iom-typescript';

export default function GrpcConnection() {
    const [client, setClient] = useState<GrpcClient | null>(null);

    useEffect(() => {
        const grpcClient = new GrpcClient({
            onLog: (message) => console.log(message)
        });
        setClient(grpcClient);

        return () => grpcClient.disconnect();
    }, []);

    const handleConnect = async (config: AuthConfig) => {
        if (client) {
            await client.connect(config, 'web-client');
        }
    };

    return (
        // UI 组件
    );
}
```

## 类型定义

SDK 导出所有 protobuf 生成的类型：

```typescript
import {
    Session, Sessions,
    Task, Tasks,
    Listener, Listeners,
    Pipeline, Pipelines,
    Empty,
    MaliceRPCClient,
    ListenerRPCClient
} from 'iom-typescript';
```

## 错误处理

```typescript
try {
    await client.connect(config, 'client_name');
    console.log('连接成功');
} catch (error) {
    console.error('连接失败:', error);
}

// RPC 调用错误处理
rpc.malice.getSessions(Empty, (error, response) => {
    if (error) {
        console.error('RPC 调用失败:', error);
        return;
    }
    // 处理响应
});
```

## 最佳实践

1. **使用日志回调** - 便于调试和监控
   ```typescript
   const client = new GrpcClient({
       onLog: (message) => console.log(`[gRPC] ${message}`)
   });
   ```

2. **正确清理资源** - 组件卸载时断开连接
   ```typescript
   useEffect(() => {
       return () => client.disconnect();
   }, []);
   ```

3. **配置超时和消息大小** - 根据需求调整
   ```typescript
   const client = new GrpcClient({
       timeout: 30,
       maxMessageSize: 100 * 1024 * 1024
   });
   ```

4. **检查连接状态** - 执行前确认已连接
   ```typescript
   if (client.isConnected()) {
       // 执行 RPC 调用
   }
   ```

## 相关资源

- [TypeScript SDK 源码](https://github.com/chainreactors/IoM-typescript)
- [核心概念](../../concept.md)
- [Proto 协议定义](https://github.com/chainreactors/proto)
