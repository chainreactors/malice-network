---
title: 多服务器连接
---

# 多目标配置

IoM在0.1.2版本开始支持多服务器配置。当主服务器失效时，会自动切换到备用服务器。

**注意** : 目前只能同时启用一种通信协议，不同协议的配置需要分别使用。

## HTTP 协议

### 基础配置

```yaml
targets:
  - address: "127.0.0.1:8080"
    http:
      method: "POST"
      path: "/"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
        Content-Type: "application/octet-stream"
```

### 多服务器配置示例

```yaml
targets:
  # Primary
  - address: "127.0.0.1:8080"
    http:
      method: "POST"
      path: "/"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
        Content-Type: "application/octet-stream"

  # Secondary
  - address: "192.168.1.100:8080"
    http:
      method: "POST"
      path: "/api"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
        Content-Type: "application/octet-stream"
```


## HTTPS 协议

### 基础配置（IP地址）

```yaml
targets:
  - address: "127.0.0.1:443"
    http:
      method: "POST"
      path: "/"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
        Content-Type: "application/octet-stream"
    tls:
      enable: true
      sni: "127.0.0.1"
      skip_verification: true
```

### 域名配置

```yaml
targets:
  - address: "primary.example.com:443"
    http:
      method: "POST"
      path: "/"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
        Content-Type: "application/octet-stream"
    tls:
      enable: true
      sni: "primary.example.com"
      skip_verification: true
```

### 多服务器配置示例

```yaml
targets:
  # Primary
  - address: "primary.example.com:443"
    http:
      method: "POST"
      path: "/"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
        Content-Type: "application/octet-stream"
    tls:
      enable: true
      skip_verification: true

  # Secondary
  - address: "secondary.example.com:443"
    http:
      method: "POST"
      path: "/"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
        Content-Type: "application/octet-stream"
    tls:
      enable: true
      sni: "secondary.example.com"
      skip_verification: true

```

**配置说明** :

- `tls.enable`: 启用TLS加密
- `tls.sni`: 使用域名时建议配置SNI
- `tls.skip_verification`: 跳过证书验证

## TCP 协议

### 基础配置

```yaml
targets:
  - address: "127.0.0.1:5001"
    tcp: {}
```

### 多服务器配置示例

```yaml
targets:
  # Primary
  - address: "primary.example.com:5001"
    tcp: {}

  # Secondary
  - address: "secondary.example.com:5001"
    tcp: {}

```

**配置说明** :

- 最简单的配置方式，仅需指定地址
- 支持IP地址和域名
- 无加密传输，适用于内网环境

## TCP+TLS 协议

### 基础配置

```yaml
targets:
  - address: "primary.example.com:5001"
    tcp: {}
    tls:
      enable: true
      sni: "primary.example.com"
      skip_verification: true
```

### 多服务器配置示例

```yaml
targets:
  # Primary
  - address: "primary.example.com:5001"
    tcp: {}
    tls:
      enable: true
      sni: "primary.example.com"
      skip_verification: true

  # Secondary
  - address: "secondary.example.com:5001"
    tcp: {}
    tls:
      enable: true
      sni: "secondary.example.com"
      skip_verification: true
```

## REM 协议

### 基础配置

```yaml
targets:
  - address: "127.0.0.1:34996"
    rem:
      link: "tcp://username:password@127.0.0.1:34996?wrapper=demo"
```

### 多服务器配置示例

```yaml
targets:
  # Primary
  - address: "primary.example.com:34996"
    rem:
      link: "tcp://user:pass@primary.example.com:34996?wrapper=demo"

  # Secondary
  - address: "secondary.example.com:34996"
    rem:
      link: "tcp://user:pass@secondary.example.com:34996?wrapper=demo"
```

**配置说明** :

- REM 协议通过 `link` 字段指定完整的 rem 连接地址
- link 格式: `[transport]://[key]:@[host]:[port]?wrapper=[]&tls=[bool]`
- 详细参数请参阅 [rem 文档](/rem/usage/#console)

## 补充说明

### 连接顺序
服务器按照在`targets`数组中的顺序依次尝试连接。

### 自动切换
当当前服务器连接失败时会继续重试连接，超过一定次数后会自动切换到下一个可用服务器。

### 协议限制
同一配置文件中只能使用一种通信协议（HTTP、TCP或REM），不能混用。

## DGA模式配置

DGA（Domain Generation Algorithm）模式允许在默认服务器无法连接时，动态生成新的域名进行连接尝试。

### 配置示例

```yaml
targets:
  - address: "primary.example.com:443"
    domain_suffix: "example.com"  # DGA域名后缀
    http:
      method: "POST"
      path: "/"
      version: "1.1"
      headers:
        User-Agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
        Content-Type: "application/octet-stream"
    tls:
      enable: true
      sni: "primary.example.com"
      skip_verification: true
```
