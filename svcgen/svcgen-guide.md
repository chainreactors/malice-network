# svcgen 使用与部署指南

svcgen 是一个 IoM mal（Lua）插件，把"构建一次 → 按 C2 地址批量 patch"的 implant 生成流程固化为一条客户端命令。当前用于生成 Windows 服务版 beacon，每个 C2 地址产出一个 exe，换 IP 不用重新编译。

日期：2026-07-27

## 一、命令用法

```
svcgen --src <基础implant.yaml> --dst <输出目录> --addrs ip1:port,ip2:port --wait
```

示例：

```
svcgen --src /home/kali/Desktop/iom/implant-svc.yaml --dst /home/kali/Desktop/iom/out --addrs 192.168.59.141:5001,10.10.10.10:5001 --wait
```

**参数**

| 参数 | 说明 |
|------|------|
| `--src` | 基础 implant.yaml 路径（client 侧绝对路径），决定编译期特性（windows_service、模块等）与固定 `obf_seed` |
| `--dst` | 输出目录（client 侧），不存在会自动创建 |
| `--addrs` | 逗号分隔的 C2 地址列表（tcp+tls 协议，与 targets 配置一致） |
| `--wait` | 全局 flag，阻塞等待完成（建议带上） |

**产物**

- `<dst>/base.exe` — docker 全量编译的基础 beacon
- `<dst>/svc_beacon_<ip_port>.exe` — 每个地址一个变体（服务端模板 patch，秒级）

**执行流程**（插件内部）

1. `rpc.Build`（Source=docker）全量编译 base exe（约 30-60 秒），轮询 `rpc.ListArtifact` 直到完成
2. `rpc.DownloadArtifact` 下载 base.exe
3. 对每个地址：sed 改写 yaml 的 `targets.address` 与 `tls.sni` → `rpc.Build`（Source=patch，模板 patch，约 1 秒）→ 下载

通过 MCP 触发：`execute_command("svcgen --src ... --dst ... --addrs ... --wait")`；需要一个 active session（仅借用其 RPC context，对目标无操作）。

## 二、部署方式

### 1. 前置条件（一次性）

| 依赖 | 说明 | 检查方式 |
|------|------|----------|
| 新版 malefic-mutant | 服务端 `.malice/bin/malefic-mutant` 必须支持 `tool patch -i ... --from-implant`（旧版只有 `--file/--server-address`，不兼容）。注意 **server 重启会还原该二进制**，需要重新替换 | `.malice/bin/malefic-mutant tool patch --help` 看到有 `--from-implant` |
| patch 模板 | 服务端 `<ServerRoot>/templates/` 下放带**固定 obf_seed** 的模板 exe，命名 `malefic-<transport>-<target>[.exe]`，如 `malefic-tcp_tls-x86_64-pc-windows-gnu.exe`。建议同时放 `malefic-http_tls-...` 别名（服务端 DetectTransport 会被 yaml 中 pulse 段的 `http:` 误判） | `ls iommcp/server/.malice/templates/` |
| 基础 yaml | 含固定 `obf_seed`（patch 靠它派生 prefix 定位配置槽；随机 seed 的产物无法 patch）。当前用 `/home/kali/Desktop/iom/implant-svc.yaml`（含 `windows_service: true`，seed `15229217100126305078`） | yaml 里 `basic.obf_seed` 非空 |

### 2. 安装插件

```bash
# 插件源在 svcgen/（mal.yaml + main.lua），也可用 svcgen-20260727.tar.gz 解开
cp -r svcgen /root/.config/malice/mals/svcgen
```

客户端控制台（或 MCP execute_command）：

```
mal load svcgen
# 成功输出: Loaded lua plugin: svcgen, register 1 commands
```

### 3. 更新插件

```
mal remove svcgen
# 覆盖 main.lua 后
mal load svcgen
```

## 三、常见问题

| 现象 | 原因 | 处理 |
|------|------|------|
| `mutant tool failed: unexpected argument '-i'` | 服务端 mutant 是旧版 | 用服务端源码重编 release 版替换 `.malice/bin/malefic-mutant`（备份旧版） |
| `implant.yaml is missing basic.obf_seed` | yaml 没带固定 seed | 在 `basic:` 下加 `obf_seed: <固定整数>`，并用它重新构建一次基础 exe |
| patch 报 `template not found for transport=http_tls` | yaml 中 pulse 段的 `http:` 让传输类型误判 | templates 目录补一个 `malefic-http_tls-<target>.exe`（可与 tcp_tls 同文件） |
| `prefix+seed pattern not found in binary` | 被 patch 的二进制不是用该 seed 构建的 | 模板 exe 必须是用带固定 seed 的 yaml 构建的那个 |
| 生成的 exe 在 Win7 弹缺失 DLL | 旧产物依赖 `RoOriginateErrorW` | 构建端 `.cargo/config.toml` 的 windows-gnu rustflags 加 `"--cfg=windows_slim_errors"`（已在 source_code 与 implant-master 配置） |

## 四、相关文件

- 插件源：`svcgen/`（`mal.yaml`、`main.lua`）
- 归档：`svcgen-20260727.tar.gz`（含插件、gen.lua、auto_gen.sh、implant-svc.yaml）
- 默认配置：`implant-svc.yaml`
- MCP 备用路径：`gen.lua`（execute_lua 直接驱动全流程）、`auto_gen.sh`（shell 封装 iom_mcp.py）
