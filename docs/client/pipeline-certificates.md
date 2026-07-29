---
title: Pipeline 证书绑定与更新
---

# Pipeline 证书绑定与更新

本文说明 HTTP/TCP Pipeline 的单向 TLS 证书绑定、换绑和续期流程。这里管理的是 Pipeline 对外提供 HTTP(S) 或 TCP TLS 信道时使用的服务端证书，不会修改 Client、Server 或 Listener 之间的 mTLS 身份证书。

## 行为概览

- `pipeline cert bind` 将已存储的证书绑定到 HTTP/TCP Pipeline，也用于换绑。
- 运行中的 Pipeline 会自动执行停止、保存新绑定、重新启动；即使证书名称未改变，也会重启以重新读取证书内容。
- 新配置启动失败时，服务端会恢复原绑定并尝试重新启动旧配置。
- 已停止或 Listener 暂时离线的 Pipeline 只更新持久化配置，不会被自动启动。
- `pipeline cert unbind` 会关闭该 Pipeline 的 TLS；运行中 Pipeline 同样会自动重启。
- Website 继续使用 `website tls` 或 `website cert` 管理 TLS。

Pipeline 的身份是 `listener_id + pipeline name`。不同 Listener 存在同名 Pipeline 时，必须使用 `--listener` 消除歧义。

## 使用方法

绑定或换绑已有证书：

```bash
pipeline cert bind \
  --pipeline http-main \
  --listener edge-a \
  --cert-name prod-cert
```

查看和解除绑定：

```bash
pipeline cert show --pipeline http-main --listener edge-a
pipeline cert unbind --pipeline http-main --listener edge-a
```

生成临时自签名证书，证书材料仅保存在 Pipeline 配置中：

```bash
pipeline cert generate --pipeline http-main --listener edge-a
```

生成证书并保存到证书库，便于其他 Pipeline 或 Website 复用：

```bash
pipeline cert generate \
  --pipeline http-main \
  --listener edge-a \
  --save-as http-main-cert \
  --comment "edge-a HTTP certificate"
```

这些新命令只接受命名 Flag，不接受位置参数。旧的 `pipeline start <name> --cert-name <cert>` 仍受支持；运行中的 Pipeline 会转入同一换绑流程。`pipeline update --cert-name` 不再用于证书绑定，请改用 `pipeline cert bind`。

## 更新正式证书

导入证书后，可保持证书名称不变，仅替换证书与私钥内容：

```bash
cert update \
  --cert-name prod-cert \
  --cert /etc/letsencrypt/live/example.com/fullchain.pem \
  --key /etc/letsencrypt/live/example.com/privkey.pem
```

默认情况下，更新成功后会逐个重载所有引用 `prod-cert` 的 Website 和 HTTP/TCP Pipeline。每个目标都会返回独立结果；一个目标失败不会阻止后续目标继续处理。

需要在维护窗口手动应用时：

```bash
cert update \
  --cert-name prod-cert \
  --cert fullchain.pem \
  --key privkey.pem \
  --no-reload

cert apply --cert-name prod-cert
```

也可以只重载一个目标：

```bash
cert apply \
  --cert-name prod-cert \
  --listener edge-a \
  --pipeline http-main
```

未传 `--ca-cert` 时，原 CA 内容保持不变。只有显式传入 `--clear-ca` 才会清空 CA；`--ca-cert` 与 `--clear-ca` 不能同时使用。证书和私钥必须成对提供，并会在写入数据库前校验是否匹配。

ACME 续期默认也会重载引用；使用 `cert renew <cert_name> --no-reload` 可只续期、不立即应用。

## 配置文件

现有 Listener YAML 配置保持兼容：

```yaml
http:
  - name: http-main
    host: 0.0.0.0
    port: 443
    tls:
      enable: true
      cert_file: /etc/ssl/example/fullchain.pem
      key_file: /etc/ssl/example/privkey.pem
      ca_file: /etc/ssl/example/ca.pem
```

CLI 证书命令更新的是 Server 数据库中的 Pipeline 与证书记录，不会改写 Listener YAML。需要由 YAML 作为长期配置源时，应同步维护 YAML 中的证书路径。

## 故障处理

`cert apply` 可重复执行，用于重试更新后未能重载的目标。运行中 Pipeline 换绑失败时，错误会同时报告新配置启动错误和可能发生的恢复错误；检查 `pipeline cert show` 与 `pipeline health` 可确认最终持久化和运行状态。
