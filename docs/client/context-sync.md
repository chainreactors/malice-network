# 上下文同步

## 概述

`context sync` 将服务端保存的上下文同步到 Client。下载文件、截图、键盘记录、上传记录和媒体文件通过 `SyncStream` 流式接收，避免 Client 和中间代理一次性缓存完整文件。

Client 会校验分片的总长度、偏移量和结束标记，并先写入临时文件。只有完整接收后，文件才会出现在 Client 临时目录中。连接旧版服务端时，如果服务端未实现 `SyncStream`，Client 会自动回退到原有的 `Sync` RPC。

## 用法

```text
context sync <context_id>
```

同步完成后，命令会输出上下文详情。文件类上下文还会输出本地保存路径，文件名格式为：

```text
<context_id>_<remote_filename>
```

## 配置

该命令没有单独的 Client 配置项。文件保存在 Client 配置根目录下的 `temp` 目录中。服务端当前以固定大小的分片发送内容，Client 不依赖具体分片大小，只按 RPC 中的偏移量和总长度进行校验。

## 示例

```text
context sync 2f48e559-2ea2-4de8-b909-0d90a7f702f3
```

成功同步文件类上下文时，输出包含：

```text
File saved to: /home/user/.config/malice/temp/2f48e559-2ea2-4de8-b909-0d90a7f702f3_capture.bin
```

如果流传输中断或分片不连续，命令会返回错误并删除未完成的临时文件。
