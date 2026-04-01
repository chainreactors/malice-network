## basic

### bcat

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### bcd

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### bcp

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### bcurl

**Arguments**

- `session` [Session] -  special session
- `url` [string] -  target url

**Example**

```
bcurl(active(),"http://example.com")
```

### bdllinject

**Arguments**

- `$1` [Session] 
- `$2` [number] 
- `$3` [string] 

### bdllspawn

**Arguments**

- `$1` [Session] 
- `$2` [number] 
- `$3` [string] 

### bdownload

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [boolean] 

### benum_drivers

**Arguments**

- `session` [Session] -  special session

**Example**

```
benum_drivers(active())
```

### benv

**Arguments**

- `$1` [Session] 

### bexecute

**Arguments**

- `session` [Session] -  special session
- `cmd` [string] -  command to execute

**Example**

```
bexecute(active(),"whoami")
```

### bexecute_assembly

**Arguments**

- `sessions` [Session] - 
- `path` [string] - 
- `args` [string] - 

**Example**

```
bexecute_assembly(active(),"sharp.exe",{})
```

### bexecute_exe

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 
- `$4` [SacrificeProcess] 

### binline_dll

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 
- `$4` [string] 

### binline_exe

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### binline_execute

**Arguments**

- `session` [Session] -  special session
- `bofPath` [string] -  path to BOF
- `args` [string] -  arguments

**Example**

```
binline_execute(active(),"/path/dir.x64.o","/path/to/list")
```

### binline_shellcode

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### bkill

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### blist_devices

**Arguments**

- `$1` [Session] 

### bls

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  path to list files

**Example**

```
bls(active(),"/tmp")
```

### bmkdir

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### bmv

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### bnetstat

**Arguments**

- `$1` [Session] 

### bpipe_read

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### bpowerpick

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  powershell script
- `ps` [string] -  ps args

**Example**

```
bpowerpick(active(),"powerview.ps1",{""}))
```

### bpowershell

**Arguments**

- `session` [Session] - 
- `cmd` [string] - 

**Example**

```
bpowershell(active(),"dir")
```

### bps

**Arguments**

- `$1` [Session] 

### bpwd

**Arguments**

- `$1` [Session] 

### breg_queryv

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 
- `$4` [string] 

### breq_query

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### brm

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### bsetenv

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### bshell

**Arguments**

- `sessions` [Session] - 
- `cmd` [string] - 

**Example**

```
bshell(active(),"whoami",true)
```

### bshinject

**Arguments**

- `$1` [Session] 
- `$2` [number] 
- `$3` [string] 
- `$4` [string] 

### bsleep

**Arguments**

- `sess` [Session] - special session
- `interval` [string] - time interval, in seconds
- `jitter` [number] - jitter, percentage of interval

**Example**

```
bsleep(active(), 10, 0.5)
```

### bunsetenv

**Arguments**

- `$1` [Session] 
- `$2` [string] 

### bupload

**Arguments**

- `session` [Session] -  special session
- `path` [string] -  source path

**Example**

```
bupload(active(),"/source/path")
```

### buploadraw

**Arguments**

- `$1` [Session] 
- `$2` [string] 
- `$3` [string] 

### bwhoami

**Arguments**

- `$1` [Session] 

