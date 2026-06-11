# Built-in Plugin Package (community)

IoM includes a built-in `community` MAL plugin package providing 90+ commands that cover common penetration testing operations.
These commands are automatically loaded at client startup and require no manual installation.

Source code is located at `helper/intl/community/`.

## Module Overview

| Module | Commands | Functionality |
|--------|----------|---------------|
| common.lua | 27 | General tools (screenshot, HTTP, file reading, nanodump, mimikatz, etc.) |
| elevate.lua | 20 | Privilege escalation (UAC bypass, Potato family, kernel exploits) |
| enum.lua | 10 | Enumeration (AV, software, domain controllers, .NET, drives, network sessions) |
| persistence.lua | 8 | Persistence (registry, services, scheduled tasks, startup folder, WMI events) |
| rem.lua | 7 | REM remote execution management |
| move.lua | 6 | Lateral movement (psexec, WMI, DCOM, RDP, Pass-the-Ticket) |
| exclusion.lua | 3 | Exclusion list management |
| net_user.lua | 3 | Network user operations |
| token.lua | 2 | Token creation and theft |
| base.lua | 1 | Precompiled module loading |
| clipboard.lua | 1 | Clipboard reading |
| route.lua | 1 | Route operations |
| lib.lua | — | Helper function library (bof_pack, read, has_clr_version) |

## common.lua — General Tools

| Command | Description | OPSEC |
|---------|-------------|-------|
| `screenshot` | Capture a screenshot and save to a specified file | 9.0 |
| `curl` | HTTP request (BOF implementation, no cmd.exe) | 8.0 |
| `readfile` | Read target file contents | 9.0 |
| `nanodump` | Advanced LSASS dump (supports fork, spoof-callstack) | 8.0 |
| `mimikatz` | Mimikatz integration | 7.0 |
| `logonpasswords` | LSASS plaintext credentials | 5.9 |
| `hashdump` | SAM hash extraction | 9.0 |
| `credman` | Credential Manager | 9.0 |
| `autologon` | Auto-logon credentials | 9.0 |
| `askcreds` | Phishing credential prompt | 6.0 |
| `ldapsearch` | LDAP query | 9.0 |
| `domain kerberoast` | Kerberoast attack | 8.0 |
| `domain sessions` | Domain session enumeration | 9.0 |
| `pingscan` | Ping sweep | 8.0 |
| `portscan` | Port scan | 8.0 |
| `nslookup` | DNS lookup (supports multiple record types) | 9.0 |
| `bof-execute_assembly` | BOF inline .NET assembly execution | 8.5 |

## elevate.lua — Privilege Escalation

### Potato Family (requires SeImpersonatePrivilege)

| Command | Description | OPSEC |
|---------|-------------|-------|
| `elevate SweetPotato` | SweetPotato escalation | 8.0 |
| `elevate EfsPotato` | EfsPotato (auto-selects CLR version) | 8.0 |
| `elevate JuicyPotato` | JuicyPotato (older Windows versions) | 7.5 |

### UAC Bypass (requires Administrators group + Medium integrity)

| Command | Description | OPSEC |
|---------|-------------|-------|
| `uac-bypass elevatedcom` | COM interface bypass (BOF) | 8.5 |
| `uac-bypass sspi` | SSPI bypass (BOF) | 8.5 |
| `uac-bypass colordataproxy` | ColorDataProxy bypass (BOF) | 8.5 |
| `uac-bypass registryshell` | Registry shell bypass (BOF) | 8.5 |
| `uac-bypass silentcleanup` | SilentCleanup bypass | 8.0 |
| `uac-bypass editionupgrade` | EditionUpgrade bypass | 8.0 |
| `uac-bypass eventvwr` | Event Viewer bypass (PowerShell) | 6.0 |

### Kernel Exploits

| Command | Target System | OPSEC |
|---------|--------------|-------|
| `elevate cve-2020-0796` | Win10 1903/1909 x64 | 7.0 |
| `elevate ms15-051` | Win7/8.1/2008R2/2012 | 7.0 |
| `elevate ms14-058` | Win7/8.1/2008R2/2012 | 7.0 |
| `elevate ms16-016` | Vista/7/8.1 x86 | 7.0 |
| `elevate ms16-032` | Win7/8.1/10 | 7.0 |
| `elevate HiveNightmare` | Win10 2004-21H1 | 7.0 |

## move.lua — Lateral Movement

| Command | Protocol | OPSEC |
|---------|----------|-------|
| `move psexec` | SMB (upload + service creation) | 7.5 |
| `move wmi-proccreate` | WMI remote process creation | 7.0 |
| `move wmi-eventsub` | WMI event subscription | 7.0 |
| `move dcom` | DCOM remote execution | 7.0 |
| `move rdphijack` | RDP session hijacking | 9.5 |
| `move krb_ptt` | Pass-the-Ticket | 7.0 |

## persistence.lua — Persistence

| Command | Mechanism | OPSEC |
|---------|-----------|-------|
| `persistence Registry_Key` | Registry Run key | 8.0 |
| `persistence Install_Service` | Windows service | 7.0 |
| `persistence Scheduled_Task` | Scheduled task | 7.5 |
| `persistence WMI_Event` | WMI event subscription | 7.5 |
| `persistence startup_folder` | Startup folder | 7.0 |
| `persistence NewLnk` | Create new shortcut | 7.0 |
| `persistence BackdoorLnk` | Hijack existing shortcut | 6.5 |

## enum.lua — Enumeration

| Command | Description |
|---------|-------------|
| `enum av` | Detect security products |
| `enum software` | Installed software |
| `enum dc` | Domain controller info |
| `enum dotnet` | .NET versions |
| `enum drives` | Disk drives |
| `enum localsessions` | Local sessions |
| `enum localcert` | Local certificates |
| `enum files` | File search |
| `enum arp` | ARP table |

## Resource Directory

Resource files bundled with the community plugin:

```
resources/
├── bof/              # BOF files (x64/x86)
│   ├── screenshot/
│   ├── curl/
│   ├── move/
│   ├── enum/
│   └── ...
├── common/           # General tools (mimikatz, nanodump, rdpthief)
├── elevate/          # Escalation tools (EfsPotato, Potato family)
├── injection/        # Injection tools
├── lib/              # Libraries (inline-ea)
├── modules/          # Precompiled DLL modules
├── move/             # Lateral movement BOFs
├── persistence/      # Persistence tools (SharpStay)
└── skills/           # Agent skills
```
