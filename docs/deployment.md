# 部署指南

本文档说明 Server 和 Client 的部署方式。

## Server 部署

### Linux 安装脚本

使用安装脚本快速部署（可选）：

```bash
curl -L "https://raw.githubusercontent.com/chainreactors/malice-network/master/install.sh" | sudo bash
```

**系统要求：**
- 操作系统：Ubuntu、Debian 或 CentOS
- 权限：root 或 sudo
- 网络连接：可访问 `github.com`、`ghcr.io`、`docker.com`

**网络问题解决：**

国内服务器访问 GitHub 可能超时，建议配置代理：

```bash
# 映射本机代理端口到 VPS
ssh -R 1080:127.0.0.1:1080 root@vps.ip

# 设置环境变量
export http_proxy="http://127.0.0.1:1080"
export https_proxy="http://127.0.0.1:1080"

# 非 root 用户保持环境变量
sudo -E bash install.sh
```

**安装交互：**

脚本会提示输入：

1. 安装路径（默认 `/iom`）：
   ```
   Please input the base directory for the installation [default: /iom]:
   ```

2. IP 地址（自动检测）：
   ```
   Please input your IP Address for the server to start [default: <自动检测的IP>]:
   ```

**安装脚本自动完成：**
1. 检查并安装 Docker
2. 下载并安装 Malice-Network 服务端及客户端
3. 下载并安装 Malefic 源码及工具
4. 拉取 Docker 镜像（约 8.21GB）：`ghcr.io/chainreactors/malefic-builder:latest`
5. 配置并启动服务（基于 systemd）

### 下载 Release 部署

Windows 或 macOS 系统部署 Server，从 [Releases](https://github.com/chainreactors/malice-network/releases/latest) 下载对应版本。

- `iom_*` - Client 端
- `malice_network_*` - Server 端

启动 Server：

```bash
./malice_network_linux_amd64
```

指定 IP 启动（Client 可访问的 IP，如公网 IP）：

```bash
./malice_network_linux_amd64 -i 123.123.123.123
```

### Server 启动参数

| 参数 | 说明 |
|------|------|
| `-c, --config` | 配置文件路径（默认 `config.yaml`） |
| `-i, --ip` | 外网 IP 地址，覆盖配置文件中的 ip 字段 |
| `--server-only` | 仅启动 server，不启动 listener |
| `--listener-only` | 仅启动 listener，不启动 server |
| `--daemon` | 以守护进程模式运行 |
| `--debug` | 开启 debug 日志 |
| `--opsec` | 启用 OPSEC 模式 |
| `--quickstart` | 交互式配置向导 |

### 常用配置修改

`config.yaml` 是 Server 端配置文件，从 [仓库](https://github.com/chainreactors/malice-network/blob/master/server/config.yaml) 下载并放到 Server 可执行文件同级目录。

**修改外网 IP：**

```yaml
server:
  ip: 127.0.0.1
```

**调整数据包大小（传输大文件或网络较差时）：**

```yaml
config:
  packet_length: 10485760   # 10M
```

**第三方消息通知配置：**

支持 Telegram、钉钉、飞书、微信等：

```yaml
notify:
  enable: false 
  telegram:
    enable: false
    api_key:        # Telegram API key
    chat_id:        # Telegram 聊天 ID
  dingtalk:
    enable: false
    secret:         # 钉钉 secret
    token:          # 钉钉 token
  lark:
    enable: false
    webhook_url:    # 飞书 webhook URL
  serverchan:
    enable: false
    url:            # ServerChan API key
  pushplus:
    enable: false
    token:          # PushPlus token
    topic:          # 消息主题
    channel:        # 推送渠道：wechat, email, telegram
```

Listener 和编译配置详见 [listeners.md](server/listeners.md) 和 [build.md](server/build.md)。

## 启动 Client

Server 启动后生成两个配置文件：
- `listener.auth` - Listener 凭证
- `admin_[server_ip].auth` - Client 登录凭证

将 `admin_[server_ip].auth` 复制到 Client 所在位置，执行登录：

```bash
./iom_linux_amd64 login admin_[server_ip].auth
```

Client 会自动连接 Server 并将凭证移动到用户配置目录：
- Windows: `C:\Users\user\.config\malice\configs`
- Linux: `/home/[username]/.config/malice/configs`
- macOS: `/Users/[username]/.config/malice/configs`

下次启动 Client 会自动显示所有可用凭证：

```bash
./iom_linux_amd64
```

## 安装 VSCode GUI

**注意：** 浏览器 GUI 和桌面版 GUI 需联系开发人员获取高级版。

### 下载文件

从 [Releases](https://github.com/chainreactors/malice-network/releases/tag/nightly) 下载：
- `iom.vsix` - VSCode 插件
- `iom_*` - Client 文件

### 安装 VSCode 插件

1. 打开 Extensions（Ctrl+Shift+X）
2. 点击 "..." 菜单
3. 选择 "Install from VSIX..."
4. 导入 `iom.vsix`

### 配置插件

在 VSCode 设置中配置 "IoM: Executable Path"，填入 Client 二进制程序路径。

### 打开 IoM 插件

IoM 需要 `.auth` 凭证文件，请先搭建 Server。

如已通过 Client 连接过 Server，会直接显示历史连接的 auth 文件，点击即可进入交互界面。

**重要：** 确保 Server、Client、GUI 版本一致。

## 相关文档

- 快速开始: [getting-started.md](getting-started.md)
- Listener 配置: [server/listeners.md](server/listeners.md)
- 构建配置: [server/build.md](server/build.md)
