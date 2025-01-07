# Gonut

English | [中文](README_zh.md)

Before using **G**onut for the first time, you need to understand: "What is **D**onut?"

## What is Donut?

### 1. [Introduction to Donut](https://github.com/TheWover/donut#1-introduction)

>  **Donut** is a position-independent code that enables in-memory execution of VBScript, JScript, EXE, DLL files and dotNET assemblies. A module created by Donut can either be staged from a HTTP server or embedded directly in the loader itself. The module is optionally encrypted using the [Chaskey](https://tinycrypt.wordpress.com/2017/02/20/asmcodes-chaskey-cipher/) block cipher and a 128-bit randomly generated key. After the file is loaded and executed in memory, the original reference is erased to deter memory scanners. The generator and loader support the following features:
>
>  - Compression of input files with aPLib and LZNT1, Xpress, Xpress Huffman via RtlCompressBuffer.
>  - Using entropy for API hashes and generation of strings.
>  - 128-bit symmetric encryption of files.
>  - Overwriting native PE headers.
>  - Storing native PEs in MEM_IMAGE memory.
>  - Patching Antimalware Scan Interface (AMSI) and Windows Lockdown Policy (WLDP).
>  - Patching Event Tracing for Windows (ETW).
>  - Patching command line for EXE files.
>  - Patching exit-related API to avoid termination of host process.
>  - Multiple output formats: C, Ruby, Python, PowerShell, Base64, C#, Hexadecimal, and UUID string.
>
>  There are dynamic and static libraries for both Linux and Windows that can be integrated into your own projects. There's also a python module which you can read more about in [Building and using the Python extension.](https://github.com/TheWover/donut/blob/master/docs/2019-08-21-Python_Extension.md)

Note: Support for Xpress Huffman was temporarily removed in Donut v1.0.

To understand in more detail what `Donut` is and how `Donut` works, you can visit: https://github.com/TheWover/donut

### 2. Main components of Donut

#### [Generator](https://github.com/TheWover/donut/blob/master/donut.c)

The Generator compresses and encrypts files of formats such as VBScript, JScript, EXE, DLL files, and .NET assemblies based on input parameters, concatenates them with the shellcode of the Loader, and finally generates the final shellcode to be used.

#### [Loader](https://github.com/TheWover/donut/tree/master/loader)

The Loader is essentially a shellcode template. When executed in memory, it will decrypt and decompress the original payload (vbs, js, exe, dll, etc.) into memory and execute the payload according to the parameters provided by the Generator.

Bypassing operations such as AMSI, WLDP, ETW are executed by the Loader.

### 3. What can Donut be used for?

If you have correctly understood most of the content mentioned earlier, you don't need to pay attention to this section. Otherwise, you can simply regard Donut as a PE-to-Shellcode tool, which has additional features such as encryption, compression, bypassing AMSI, WLDP, ETW, stealthy invocation of system APIs compared to other tools (such as [pe_to_shellcode](https://github.com/hasherezade/pe_to_shellcode), etc.), and can better evade AV, EDR's memory detection and behavior detection.

## What is Gonut?

**G**onut is a cross-platform implementation of **D**onut's **Generator**, written in pure Go without CGO, and supports most mainstream systems (Windows, Linux, macOS, etc.) and architectures (i386, amd64, arm, Apple silicon, etc.).

Note again: **G**onut is just a cross-platform implementation of **D**onut's **Generator** and does not include **D**onut's **Loader**.

### Why Gonut exists

- **D**onut's Generator can only run on Windows and Linux systems.
- The behavior of **D**onut's Generator under Linux is not completely consistent with that under Windows. ([#45](https://github.com/TheWover/donut/issues/45))
- **D**onut's Generator does not support Xpress, LZNT1 compression under Linux.
- Currently, it is impossible to compile **D**onut under Arm architecture on Windows and Linux.
- **D**onut's Generator cannot be used under macOS (M-series chips).
- Let more people know about **D**onut, a seriously underrated project.

To solve the above problems, **G**onut was born.

### Goals of Gonut

- Have behavior consistent with **D**onut on Windows across all systems (Windows, Linux, macOS, etc.) except for the compression feature (for more details, see the **Differences between Gonut and Donut's Generator**).
- Provide a better user experience.

### Not Goals of Gonut

- Since **G**onut is completely dependent on **D**onut's Loader, it will not add features that Donut does not have (or explicitly states it does not support).

### Differences between Gonut and Other Similar Projects

**D**onut recommends two third-party implementations of Generators:

- [C# generator by n1xbyte](https://github.com/n1xbyte/donutCS)
- [Go generator by awgh](https://github.com/Binject/go-donut)

However, these two projects have not been updated for a long time, do not support the latest version (v1.0) of Donut Loader, and do not support common functions such as Decoy, ETW Bypass, compression, specified output formats, etc.

### Differences between Gonut and Donut's Generator

1. Difference in compression function: Since **D**onut uses the non-open source aPLib compression function and Microsoft's RtlCompressBuffer function, both of which cannot be perfectly reproduced on non-Windows systems, currently Gonut can only try to simulate these two compression algorithms and compression formats.
2. Added output formats: Golang, Rust, etc.

## How to Use Gonut

For various reasons, Gonut currently does **not intend** to provide precompiled binary files, which means that if you want to use Gonut, you will need to install the most basic [Golang](https://go.dev/dl/) development environment or [Docker](https://docs.docker.com/get-docker/) runtime environment.

### Building via Docker

```bash
git clone https://github.com/wabzsy/gonut

cd gonut

docker build -t gonut .

# docker run --rm -it -v `pwd`:/opt gonut -h
```

### Installation via go install

```bash
go install -v github.com/wabzsy/gonut/gonut@latest
```

### Building from source

```bash
git clone https://github.com/wabzsy/gonut

cd gonut/gonut

go build -v
```

### Usage

Much the same as [**D**onut's usage](https://github.com/TheWover/donut#4-usage). The following table lists switches supported by the command line version of the **G**onut:

| Switch         | Argument type | Description                                                  |
| -------------- | ------------- | ------------------------------------------------------------ |
| -n, --modname  | string        | Module name for HTTP staging.<br/> If entropy is enabled, this is generated randomly. |
| -s, --server   | string        | Server that will host the Donut module. <br/>Credentials may be provided in the following format: <br/>https://username:password@192.168.0.1/ |
| -e, --entropy  | int           | Entropy:<br/>  1=None<br/>  2=Use random names<br/>  3=Random names + symmetric encryption<br/>(default 3) |
| -a, --arch     | int           | Target architecture:<br/>  1=x86<br/>  2=amd64<br/>  3=x86+amd64<br/>(default 3) |
| -o, --output   | string        | Output file to save loader.<br/>(default: loader.[format])   |
| -f, --format   | int           | Output format:<br/>  1=Binary<br/>  2=Base64<br/>  3=C<br/>  4=Ruby<br/>  5=Python<br/>  6=Powershell<br/>  7=C#<br/>  8=Hex<br/>  9=UUID<br/>  10=Golang<br/>  11=Rust<br/>(default 1) |
| -y, --oep      | int           | Create thread for loader and continue execution at \<addr\> supplied. <br/>(eg. 0xdeadbeef) |
| -x, --exit     | int           | Exit behaviour:<br/>  1=Exit thread<br/>  2=Exit process<br/>  3=Do not exit or cleanup and block indefinitely<br/>(default 1) |
| -c, --class    | string        | Optional class name. (required for .NET DLL, format: namespace.class) |
| -d, --domain   | string        | AppDomain name to create for .NET assembly.<br/>If entropy is enabled, this is generated randomly. |
| -i, --input    | string        | Input file to execute in-memory.                             |
| -m, --method   | string        | Optional method or function for DLL. <br/>(a method is required for .NET DLL) |
| -p, --args     | string        | Optional parameters/command line inside quotations for DLL method/function or EXE. |
| -w, --unicode  |               | Command line is passed to unmanaged DLL function in UNICODE format. <br/>(default is ANSI) |
| -r, --runtime  | string        | CLR runtime version. MetaHeader used by default or v4.0.30319 if none available. |
| -t, --thread   |               | Execute the entrypoint of an unmanaged EXE as a thread.      |
| -z, --compress | int           | Pack/Compress file:<br/>  1=None<br/>  2=aPLib               [experimental]<br/>  3=LZNT1  (RTL)   [experimental, Windows only]<br/>  4=Xpress (RTL)   [experimental, Windows only]<br/>  5=LZNT1              [experimental]<br/>  6=Xpress             [experimental, recommended]<br/>(default 1) |
| -b, --bypass   | int           | Bypass AMSI/WLDP/ETW:<br/>  1=None<br/>  2=Abort on fail<br/>  3=Continue on fail<br/>(default 3) |
| -k, --headers  | int           | Preserve PE headers:<br/>  1=Overwrite<br/>  2=Keep all<br/>(default 1) |
| -j, --decoy    | string        | Optional path of decoy module for Module Overloading.        |
| -v, --verbose  |               | verbose output. (debug mode)                                 |
| -h, --help     |               | help for gonut                                               |
| --version      |               | version for gonut                                            |

### Payload Requirements

The same as **D**onut, see [Payload Requirements](https://github.com/TheWover/donut/tree/master#payload-requirements) for details.

## Disclaimer

The same as **D**onut, see [Disclaimer](https://github.com/TheWover/donut/tree/master#8-disclaimer) for details.

We are not responsible for any misuse of this software or technique. Gonut is provided as a demonstration of CLR Injection and in-memory loading through shellcode in order to provide red teamers a way to emulate adversaries and defenders a frame of reference for building analytics and mitigations. This inevitably runs the risk of malware authors and threat actors misusing it. However, we believe that the net benefit outweighs the risk. Hopefully that is correct. In the event EDR or AV products are capable of detecting Gonut via signatures or behavioral patterns, we will not update Gonut to counter signatures or detection methods. To avoid being offended, please do not ask.