# MAL Plugin Usage Guide

MAL (Malice Scripting Language) is IoM's Lua plugin system. By installing MAL plugins, you can extend the client's command capabilities without recompiling.

## What Are MAL Plugins

MAL plugins are Lua script packages. Each plugin can:
- Register new client commands (e.g., `screenshot`, `curl`, `nanodump`)
- Integrate BOF (Beacon Object File) tools
- Add event listeners (session online notifications, etc.)
- Extend automation capabilities

IoM includes a built-in `community` plugin (200+ commands) by default, covering screenshots, enumeration, privilege escalation, lateral movement, persistence, and other common operations.

## Plugin Management Commands

### List Installed Plugins

```
mal list
```

Displays all loaded plugins (built-in + externally installed), including name, version, author, and source.

### Install a Plugin

```
mal install /path/to/plugin.tar.gz
```

Installs from a local tar.gz package into `~/.malice/mals/`.

### Install from the Community Repository

```
mal refresh                          # Update the plugin index
mal install <plugin-name>            # Install from the repository
```

Community repository: https://github.com/chainreactors/mal-community

### Load a Plugin (for development)

```
mal load /path/to/plugin-dir         # Load directly from a directory
```

### Uninstall a Plugin

```
mal remove <plugin-name>
```

### Update Plugins

```
mal update <plugin-name>             # Update a specific plugin
mal update --all                     # Update all plugins
```

## Built-in Community Plugin Command Categories

The following commands are provided by the built-in `community` plugin (some require the corresponding implant module):

### Reconnaissance and Enumeration

| Command | Description |
|---------|-------------|
| `screenshot` | Capture a screenshot |
| `clipboard` | Read clipboard contents |
| `enum av` | Detect security products |
| `enum software` | Installed software |
| `enum dc` | Domain controller info |
| `enum dotnet` | .NET versions |
| `enum drives` | Disk drives |
| `curl` | HTTP request (BOF implementation, no cmd.exe) |

### Credential Harvesting

| Command | Description |
|---------|-------------|
| `hashdump` | SAM hash extraction |
| `logonpasswords` | LSASS credentials |
| `nanodump` | Advanced LSASS dump |
| `mimikatz` | Mimikatz integration |
| `credman` | Credential Manager |
| `domain kerberoast` | Kerberoast |

### Privilege Escalation

| Command | Description |
|---------|-------------|
| `elevate SweetPotato` | Potato escalation |
| `elevate EfsPotato` | EFS Potato |
| `uac-bypass elevatedcom` | UAC bypass (BOF) |
| `uac-bypass sspi` | UAC bypass (SSPI) |
| `uac-bypass silentcleanup` | UAC bypass (SilentCleanup) |

### Lateral Movement

| Command | Description |
|---------|-------------|
| `move psexec` | PsExec lateral movement |
| `move wmi-proccreate` | WMI remote execution |
| `move dcom` | DCOM remote execution |
| `move krb_ptt` | Pass-the-Ticket |

### Persistence

| Command | Description |
|---------|-------------|
| `persistence Registry_Key` | Registry Run key |
| `persistence Install_Service` | Service installation |
| `persistence Scheduled_Task` | Scheduled task |
| `persistence startup_folder` | Startup folder |

### Token Operations

| Command | Description |
|---------|-------------|
| `token make` | Create a token |
| `token steal` | Steal a process token |
| `rev2self` | Revert to original token |

Use `search_commands("keyword")` to find more commands, and `<command> --help` to view detailed usage.

## Relationship Between Plugins and Modules

MAL plugins run on the **client side**, but many commands require the implant to support the corresponding **module**.

If a command reports that the module is unavailable:
1. Run `modules list` to view the modules supported by the current session
2. Run `modules load <module>` to attempt dynamic loading
3. If loading is not possible, rebuild the implant with the required module included
