# Gonut

[English](README.md) | 中文

在初次使用**G**onut之前，您需要先了解：“**D**onut是什么？”。

## Donut 是什么？

### 1. Donut 简介

>  **Donut** 是一种与位置无关的代码，可以在内存中执行 VBScript、JScript、EXE、DLL 文件和 .NET 程序集。
>
>  Donut 创建的模块可以从 HTTP 服务器暂存，也可以直接嵌入到加载器本身中。
>
> 该模块可选择使用 [Chaskey](https://tinycrypt.wordpress.com/2017/02/20/asmcodes-chaskey-cipher/) 分组密码和 128 位随机生成的密钥进行加密。
>
> 文件在内存中加载并执行后，原始引用将被删除以阻止内存扫描。
>
> 生成器（Generator）和加载器（Loader）支持以下功能：
>
> -  使用 aPLib 或通过 RtlCompressBuffer的 LZNT1、Xpress、Xpress Huffman 压缩输入文件。（注：在Donut v1.0版本中暂时移除了对 Xpress Huffman 的支持。）
> - 使用熵进行 API 哈希和字符串生成。
> - 文件的 128 位对称加密。
> - 覆盖 PE 头。
> - 将 PE 文件存储在 MEM_IMAGE 内存中。
> - 修补反恶意软件扫描接口 (AMSI) 和 Windows 锁定策略 (WLDP)。
> - 修补 Windows 事件跟踪 (ETW)。
> - 修补 EXE 文件的命令行。
> - 修补与退出相关的 API 以避免宿主进程终止。
> - 多种输出格式：C、Ruby、Python、PowerShell、Base64、C#、Hex（十六进制字符串）和 UUID 字符串。
>
> 提供Linux 和 Windows 的动态和静态库，可以将它们集成到您自己的项目中。还有一个 Python 模块，您可以在[构建和使用 Python 扩展](https://github.com/TheWover/donut/blob/master/docs/2019-08-21-Python_Extension.md)中阅读有关该模块的更多信息。

\* 以上内容译自： https://github.com/TheWover/donut#1-introduction

\* 想要更详细的了解什么是`Donut`，以及`Donut`是如何运作的可访问：https://github.com/TheWover/donut

### 2. Donut 的主要组成部分

#### Generator（[生成器](https://github.com/TheWover/donut/blob/master/donut.c)）

Generator根据输入的参数将 VBScript、JScript、EXE、DLL 文件和 .NET 程序集等格式的文件压缩、加密后与加载器的Shellcode进行拼接，最后生成最终要使用的Shellcode。

#### Loader（[加载器](https://github.com/TheWover/donut/tree/master/loader)）

Loader相当于一个Shellcode模板，在内存中执行时会根据Generator在生成时所用到的参数解密、解压原始payload（vbs、js、exe、dll等）到内存并执行payload。

绕过AMSI、WLDP、ETW等操作均由Loader来执行。

### 3. Donut 可用来做什么

如果您正确理解了前面所述的大部分内容，您可以不必理会本段的内容，否则您可以简单的将Donut当成一个PE转Shellcode的工具，并相比其他工具（如[pe_to_shellcode](https://github.com/hasherezade/pe_to_shellcode)等）多了加密，压缩，绕过AMSI、WLDP、ETW，隐蔽调用系统API等功能，并且比那些工具能更好的规避AV、EDR的内存检测与行为检测。

## Gonut是什么？

**G**onut是**D**onut的**Generator**的跨平台实现，纯Go编写，无CGO，支持大部分主流系统（Windows、Linux、macOS等）与架构（i386、amd64、arm、Apple silicon等）。

请再次注意：**G**onut仅仅是**D**onut的**Generator**的跨平台实现，不含**D**onut的**Loader**。

### 为什么会有Gonut

- **D**onut的Generator仅可运行在Windows与Linux系统下。
- **D**onut的Generator在Linux系统下的行为与在Windows系统下的行为不完全一致（[#45](https://github.com/TheWover/donut/issues/45)）。
- **D**onut的Generator在Linux系统下不支持LZNT1、Xpress的压缩功能。
- 目前无法在在Arm架构的Windows、Linux下编译**D**onut。
- **D**onut的Generator在macOS（M系列芯片）下无法使用。
- 让更多人了解到**D**onut这个被严重低估的项目。

为了解决上述问题，就有了**G**onut。

### Gonut的目标

- 在各个系统下（Windows、Linux、macOS等）的行为与**D**onut在Windows系统下的行为一致（压缩功能除外，详见**Gonut与Donut的Generator的区别**）。
- 更好的使用体验。

### 非Gonut的目标

- 由于**G**onut是完全依赖于**D**onut的Loader的，所以不会添加**D**onut所没有（或明确表示不支持）的功能。

### Gonut与其他类似项目的区别

**D**onut官方还推荐了两个第三方实现的Generator：

- [C# generator by n1xbyte](https://github.com/n1xbyte/donutCS)
- [Go generator by awgh](https://github.com/Binject/go-donut)

但是这两个项目均已很久未更新，不支持最新版**D**onut（v1.0）的Loader，且均不支持Decoy、ETW Bypass、压缩、指定输出格式等常用功能。

### Gonut与Donut的Generator的区别

1. 压缩功能的区别：由于**D**onut使用的是非开源的aPLib压缩功能，和微软提供的RtlCompressBuffer功能，这两者均无法在非Windows系统下完美复现，所以目前**G**onut仅能做到尽量模拟这两者的压缩算法与压缩格式。
2. 在**D**onut原本支持的输出格式上又添加了：Golang、Rust等输出格式。

## Gonut如何使用

出于各种原因，Gonut目前**不打算**提供预编译的二进制文件，这意味着如果您想使用Gonut将需要安装最基本的[Golang](https://go.dev/dl/)开发环境**或**[Docker](https://docs.docker.com/get-docker/)运行时环境。

### 通过 Docker 构建

```bash
git clone https://github.com/wabzsy/gonut

cd gonut

docker build -t gonut . -f Dockerfile-cn

# docker run --rm -it -v `pwd`:/opt gonut -h
```

### 通过 go install 安装

```bash
go install -v github.com/wabzsy/gonut/gonut@latest
```

### 通过源码构建

```bash
git clone https://github.com/wabzsy/gonut

cd gonut/gonut

go build -v
```

### 使用方式

与[**D**onut的用法](https://github.com/TheWover/donut)大致相同，下表为命令行版本**G**onut的所支持的参数：

| 选项           | 参数类型 | 描述                                                         |
| -------------- | -------- | ------------------------------------------------------------ |
| -n, --modname  | string   | Module name for HTTP staging.<br/> If entropy is enabled, this is generated randomly. |
| -s, --server   | string   | Server that will host the Donut module. <br/>Credentials may be provided in the following format: <br/>https://username:password@192.168.0.1/ |
| -e, --entropy  | int      | Entropy:<br/>  1=None<br/>  2=Use random names<br/>  3=Random names + symmetric encryption<br/>(default 3) |
| -a, --arch     | int      | Target architecture:<br/>  1=x86<br/>  2=amd64<br/>  3=x86+amd64<br/>(default 3) |
| -o, --output   | string   | Output file to save loader.<br/>(default: loader.[format])   |
| -f, --format   | int      | Output format:<br/>  1=Binary<br/>  2=Base64<br/>  3=C<br/>  4=Ruby<br/>  5=Python<br/>  6=Powershell<br/>  7=C#<br/>  8=Hex<br/>  9=UUID<br/>  10=Golang<br/>  11=Rust<br/>(default 1) |
| -y, --oep      | int      | Create thread for loader and continue execution at \<addr\> supplied. <br/>(eg. 0xdeadbeef) |
| -x, --exit     | int      | Exit behaviour:<br/>  1=Exit thread<br/>  2=Exit process<br/>  3=Do not exit or cleanup and block indefinitely<br/>(default 1) |
| -c, --class    | string   | Optional class name. (required for .NET DLL, format: namespace.class) |
| -d, --domain   | string   | AppDomain name to create for .NET assembly.<br/>If entropy is enabled, this is generated randomly. |
| -i, --input    | string   | Input file to execute in-memory.                             |
| -m, --method   | string   | Optional method or function for DLL. <br/>(a method is required for .NET DLL) |
| -p, --args     | string   | Optional parameters/command line inside quotations for DLL method/function or EXE. |
| -w, --unicode  |          | Command line is passed to unmanaged DLL function in UNICODE format. <br/>(default is ANSI) |
| -r, --runtime  | string   | CLR runtime version. MetaHeader used by default or v4.0.30319 if none available. |
| -t, --thread   |          | Execute the entrypoint of an unmanaged EXE as a thread.      |
| -z, --compress | int      | Pack/Compress file:<br/>  1=None<br/>  2=aPLib               [experimental]<br/>  3=LZNT1  (RTL)   [experimental, Windows only]<br/>  4=Xpress (RTL)   [experimental, Windows only]<br/>  5=LZNT1              [experimental]<br/>  6=Xpress             [experimental, recommended]<br/>(default 1) |
| -b, --bypass   | int      | Bypass AMSI/WLDP/ETW:<br/>  1=None<br/>  2=Abort on fail<br/>  3=Continue on fail<br/>(default 3) |
| -k, --headers  | int      | Preserve PE headers:<br/>  1=Overwrite<br/>  2=Keep all<br/>(default 1) |
| -j, --decoy    | string   | Optional path of decoy module for Module Overloading.        |
| -v, --verbose  |          | verbose output. (debug mode)                                 |
| -h, --help     |          | help for gonut                                               |
| --version      |          | version for gonut                                            |

### Payload 要求

与**D**onut一致，详见[Payload Requirements](https://github.com/TheWover/donut/tree/master#payload-requirements)

## 免责声明

与**D**onut一致，详见[Disclaimer](https://github.com/TheWover/donut/tree/master#8-disclaimer)

我们对任何滥用该软件或技术的行为不承担任何责任。 Gonut 是通过 shellcode 演示 CLR 注入和内存加载，以便为红队人员提供一种模拟对手和防御者的方法，为构建分析和缓解措施提供参考框架。这不可避免地存在恶意软件作者和威胁行为者滥用它的风险。然而，我们认为净收益大于风险。希望这是正确的。如果 EDR 或 AV 产品能够通过签名或行为模式检测 Gonut，我们不会更新 Gonut 以对抗签名或检测方法。为了避免被冒犯，请勿询问。