# Guardrail

Guardrail 用于确保 implant 只在预期环境中执行。

## 配置说明

```yaml
guardrail:
  enable: false          # 是否启用
  require_all: true
  ip_addresses: []       # IP地址白名单，支持通配符 ["192.168.*.*", "10.0.*.*"]
  usernames: []          # 用户名白名单，支持通配符 ["*admin*", "root*"]
  server_names: []       # 服务器名白名单，支持通配符 ["*server*", "workstation*"]
  domains: []            # 域名白名单，支持通配符 ["pentest*", "*.local"]
```

### 参数说明

- **enable** : 控制是否启用 Guardrail 功能
- **require_all** :
  - `true` (AND 模式): 所有配置的条件都必须满足，implant 才会执行
  - `false` (OR 模式): 任意一个条件满足，implant 即可执行
- **白名单字段** : 支持使用 `*` 通配符进行模糊匹配

## 使用示例

### 示例 1: AND 模式

如下示例用于限制malefic在特定内网网段内执行

```yaml
guardrail:
  enable: true
  require_all: true  # 所有条件必须同时满足
  ip_addresses:
    - "192.168.10.*"
    - "192.168.20.*"
  usernames: []
  server_names: []
  domains: []
```

### 示例 2: OR 模式

如下示例表示，满足其中一个特征即可

```yaml
guardrail:
  enable: true
  require_all: false  # 任一条件满足即可
  ip_addresses:
    - "10.0.*.*"
  usernames:
    - "*admin*"
    - "test*"
  server_names:
    - "*-test-*"
    - "pentest*"
  domains:
    - "*.test.local"
```

**效果** : implant 会在满足以下任一条件的主机上运行：

- IP 地址在 `10.0.*.*` 网段，或
- 用户名包含 `admin` 或以 `test` 开头，或
- 服务器名包含 `-test-` 或以 `pentest` 开头，或
- 域名为 `*.test.local`
