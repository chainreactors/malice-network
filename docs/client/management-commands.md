---
title: Client 管理命令
---

# Client 管理命令

本页记录 Client 侧常用资源管理命令。命令以当前 RPC 和服务端模型为准；没有服务端字段或存储能力的管理动作不会在 Client 侧伪造状态。

## Certificate

```bash
cert inspect <cert_name>
cert verify <cert_name>
cert renew <cert_name> --domain example.com
cert update --cert-name <cert_name> --cert fullchain.pem --key privkey.pem
cert apply --cert-name <cert_name>
cert list-refs <cert_name>
cert prune --expired
```

- `inspect` 下载并展示证书元数据。
- `verify` 校验证书有效期；如果证书条目包含私钥，也会校验证书与私钥是否匹配。
- `renew` 调用 ACME 获取流程。未指定 `--domain` 时，会尝试使用证书条目的 `domain` 或证书名称；默认续期后重载引用。
- `update` 只更新显式提供的字段，未传 `--ca-cert` 时保留原 CA，并默认重载引用；`--clear-ca` 显式清 CA，`--no-reload` 延后重载。
- `apply` 重载所有引用该证书的 Website 和 HTTP/TCP Pipeline，也可用 `--listener`、`--pipeline` 缩小范围。
- `list-refs` 查询 website/pipeline 中引用该证书的条目。
- `prune --expired` 删除已经过期的证书。

## Pipeline, Listener, Job

```bash
listener inspect <listener_id>

pipeline inspect <pipeline>
pipeline restart <pipeline>
pipeline update <pipeline> --enable
pipeline cert bind --pipeline <pipeline> --listener <listener_id> --cert-name <cert_name>
pipeline cert unbind --pipeline <pipeline> --listener <listener_id>
pipeline cert generate --pipeline <pipeline> --listener <listener_id> --save-as <cert_name>
pipeline cert show --pipeline <pipeline> --listener <listener_id>
pipeline health

job inspect <job>
job kill <job>
```

- `pipeline inspect` 从本地缓存或 `ListPipelines` 查询 pipeline。
- `pipeline restart` 顺序调用 `StopPipeline` 和 `StartPipeline`。
- `pipeline update` 需要本地已有 pipeline 缓存，然后调用 `SyncPipeline` 更新 `enable`、`parser` 等字段；证书使用 `pipeline cert bind` 管理。
- `pipeline cert bind` 同时用于首次绑定和换绑。运行中的 HTTP/TCP Pipeline 会自动停止并重启，失败时恢复旧绑定；停止态只保存配置。
- `pipeline cert` 子命令全部使用命名 Flag，不接受位置参数。完整流程见 [Pipeline 证书绑定与更新](pipeline-certificates.md)。
- `pipeline health` 汇总已配置 pipeline、启用数量和当前运行 job 数。
- `job kill` 会停止该 job 对应的 pipeline。

## Artifact

```bash
artifact inspect <artifact_name>
artifact publish <artifact_name> --website <website> --path /payload.bin
artifact prune --failed
artifact prune --older-than 720h
```

- `inspect` 是 `artifact show` 的语义化别名。
- `publish` 下载 artifact 后写入 website content。
- `prune` 可以按失败状态或时间阈值删除 artifact。

当前 artifact 模型没有 tag 字段，`UpdateArtifact` 也只支持 comment 更新，因此 Client 不提供假 `tag` 或 `rename` 状态。

## Website

```bash
website inspect <website>
website route add <file> --website <website> --path /index.html
website route add --artifact <artifact> --website <website> --path /payload.bin
website route list <website>
website route remove <content_id>
website cert <website> --cert-name <cert_name>
website tls <website> --generate
```

- `route` 子命令复用现有 website content RPC。
- `cert` 是 `website tls` 的证书管理入口，可绑定已有证书、导入 inline cert/key、生成临时自签名证书或禁用 TLS。
