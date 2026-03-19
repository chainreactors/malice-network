# MAL Lua API Complete Reference

## Command Registration

### command(name, fn, short, ttp)

Register a new client command.

```lua
local cmd = command("my_cmd", handler_function, "Short description", "T1059")
```

- `name` — Command name. Supports colon-delimited hierarchy (e.g., `"move:psexec"` creates a `psexec` subcommand under `move`)
- `fn` — Lua handler function
- `short` — Short description
- `ttp` — MITRE ATT&CK technique ID (empty string if none)
- Returns a `cobra.Command` object for chaining flag additions

### Flag System

Add flags via the returned cobra.Command:

```lua
local cmd = command("scan", run_scan, "Port scan", "T1046")
cmd:Flags():String("target", "", "Target IP")
cmd:Flags():Int("port", 445, "Target port")
cmd:Flags():Bool("verbose", false, "Verbose output")
cmd:Flags():StringSlice("ports", {}, "Multiple ports")
```

Read flags in the handler:

```lua
local function run_scan(args, cmd)
    local target = cmd:Flags():GetString("target")
    local port = cmd:Flags():GetInt("port")
    local verbose = cmd:Flags():GetBool("verbose")
end
```

### Parameter Conventions

Handler function parameter names have special meanings:

| Prefix/Name | Meaning | Example |
|-------------|---------|---------|
| `arg_xxx` | Positional argument | `arg_target` -> first argument |
| `flag_xxx` | Flag value | `flag_port` -> value of `--port` |
| `cmdline` | Full command line string | Auto-injected |
| `args` | Argument array | Auto-injected |
| `cmd` | cobra.Command object | Auto-injected |

### help(name, text) / example(name, text) / opsec(name, score)

```lua
opsec("screenshot", 9.0)
help("screenshot", [[ ... ]])
example("screenshot", "screenshot --filename desktop.png")
```

### UI Schema Functions

```lua
ui_widget("target", "text")         -- widget: text, textarea, checkbox, updown, tags
ui_placeholder("target", "192.168.1.0/24")
ui_required("target", true)
ui_group("target", "Network")
ui_range("port", 1, 65535)
ui_order("target", 1)
ui_set("target", {widget = "text", required = true, placeholder = "IP"})
```

## Beacon Execution Functions

These are the most commonly used implant execution functions. The `b` prefix indicates beacon package functions:

### Command Execution

| Function | Description | OPSEC |
|----------|-------------|-------|
| `bof(session, path, args, output)` | Execute BOF (inline, no new process) | 9.5 |
| `bshell(session, cmd)` | Execute via cmd.exe | 3.0 |
| `bpowershell(session, cmdline)` | PowerShell execution | 3.0 |
| `bpowerpick(session, script, ps)` | PS execution without powershell.exe | 6.0 |

### Program Execution

| Function | Description | OPSEC |
|----------|-------------|-------|
| `bexecute(session, cmd)` | Execute a local binary | 7.0 |
| `bexecute_exe(session, path, args, sac)` | Execute PE in a sacrifice process | 7.0 |
| `bexecute_assembly(session, path, args)` | Execute .NET assembly | 7.5 |
| `binline_exe(session, path, args)` | Inline PE execution | 8.5 |
| `binline_dll(session, path, entrypoint, args)` | Inline DLL execution | 8.5 |
| `binline_shellcode(session, path)` | Inline shellcode execution | 8.5 |
| `binline_execute(session, path, args)` | BOF inline execution | 9.5 |

### Injection

| Function | Description |
|----------|-------------|
| `bdllinject(session, ppid, path)` | DLL injection into target process |
| `bshinject(session, ppid, arch, path)` | Shellcode injection into target process |
| `bdllspawn(session, ppid, path)` | DLL spawn in new process |

### Usage Examples

```lua
local session = active()
local sac = new_sacrifice(0, true, true, true, "")

-- BOF execution (safest)
local task = bof(session, script_resource("tool.x64.o"), bof_pack("z", "arg"), true)

-- .NET assembly execution
local task = bexecute_assembly(session, script_resource("tool.exe"), {"arg1", "arg2"})

-- PE execution in sacrifice process
local task = bexecute_exe(session, script_resource("tool.exe"), "arg1 arg2", sac)
```

## Resource Access

| Function | Purpose |
|----------|---------|
| `script_resource(filename)` | Get file path under the plugin's `resources/` directory |
| `global_resource(filename)` | Get global shared resource path (`~/.malice/resources/`) |
| `find_resource(session, base, ext)` | Auto-locate plugin resource by architecture |
| `find_global_resource(session, base, ext)` | Auto-locate global resource by architecture |
| `read_resource(filename)` | Read plugin resource file contents |
| `read_global_resource(filename)` | Read global resource file contents |

```lua
-- Auto-locate BOF by architecture
local session = active()
local path = find_resource(session, "bof/tool/tool", "o")
-- x64 -> resources/bof/tool/tool.x64.o
```

## Session and Task

### Session Object

```lua
local session = active()
session.SessionId                -- Session ID
session.Os.Arch                  -- "x64" / "x86"
session.Os.Hostname              -- Hostname
session.Os.Username              -- Username
session.Os.Name                  -- OS name
session.Os.ClrVersion            -- .NET CLR version list
```

### Architecture Helper

```lua
local arch = barch(session)      -- Get architecture, defaults to "x64" if empty
```

### Task Result Handling

```lua
local task = bof(session, path, args, true)
wait(task)                       -- Wait for completion
taskprint(task)                  -- Print formatted output
assemblyprint(task)              -- Print .NET output
local result = get(task, 0)      -- Get the Nth result
```

### Callback Functions

Control how task output is written:

```lua
callback_file("/tmp/output.txt")      -- Write results to file
callback_append("/tmp/log.txt")       -- Append results to file
callback_discard()                     -- Discard results
```

### Module Loading

```lua
-- Load a precompiled DLL module into the implant
load_module(session, "module_name", script_resource("modules/mod.x64.dll"))
```

### File Operations

```lua
-- Upload a file to the target
uploadraw(session, file_content, remote_path, "0644", false)

-- Get the current implant's own binary
local payload = self_artifact(session)
```

## Sacrifice Process Configuration

```lua
-- Create sacrifice process configuration
local sac = new_sacrifice(
    1234,       -- ppid (0 = no spoofing)
    true,       -- block_dll (block non-Microsoft DLLs)
    true,       -- disable ETW
    true,       -- bypass AMSI
    ""          -- argue (argument spoofing string)
)

-- Shorthand version
local sac = new_sac()  -- Simplified version defined in community lib.lua

-- Bypass configuration
local bypass = new_bypass(true, true, false)  -- amsi, etw, wldp
local bypass_all = new_bypass_all()           -- Bypass all

-- Binary execution configuration
local bin = new_binary(path, args, output, timeout, arch, process, sac)
```

## BOF Argument Packing

```lua
-- bof_pack is a convenience wrapper provided by community lib.lua
local packed = bof_pack("Zizb", wide_string, integer, ansi_string, binary_data)

-- Low-level function
local packed = pack_bof_args(format, args_table)
```

| Format Char | Type | Description |
|-------------|------|-------------|
| `z` | string | ANSI string |
| `Z` | string | Wide string (UTF-16) |
| `i` | int32 | 32-bit integer |
| `s` | int16 | 16-bit short integer |
| `b` | binary | Binary data (length-prefixed) |

## Encoding and Utilities

```lua
base64_encode(input)              -- Base64 encode
base64_decode(input)              -- Base64 decode
arg_hex(input)                    -- Hex encode, returns "hex::..." prefixed string
pack_binary(data)                 -- Binary packing
random_string(length)             -- Generate random string
file_exists(path)                 -- Check if file exists
is_full_path(path)                -- Check if path is absolute
timestamp()                       -- Current timestamp
timestamp_format(format)          -- Formatted timestamp
shellsplit("cmd arg1 'arg 2'")    -- Parse command line -> {"cmd", "arg1", "arg 2"}
parse_octal("0755")               -- Parse octal
parse_hex("0xff")                 -- Parse hexadecimal
format_path(path)                 -- Format file path
```

## Event Callbacks

Define global functions prefixed with `on_` in Lua to automatically register them as event callbacks.

### Session Events

```lua
function on_beacon_initial(event)      end  -- New session registered
function on_beacon_checkin(event)      end  -- Session check-in
function on_beacon_error(event)        end  -- Session error
function on_beacon_indicator(event)    end  -- Session log indicator
function on_beacon_output(event)       end  -- Task output
function on_beacon_output_alt(event)   end  -- Alternate output
function on_beacon_output_jobs(event)  end  -- Job completed
function on_beacon_output_ls(event)    end  -- Directory listing completed
function on_beacon_output_ps(event)    end  -- Process listing completed
function on_beacon_tasked(event)       end  -- Task dispatched
```

### General Events

```lua
function on_event_action(event)         end  -- Broadcast action
function on_event_beacon_initial(event) end  -- Session initialization
function on_event_join(event)           end  -- Client joined
function on_event_notify(event)         end  -- Notification
function on_event_public(event)         end  -- Public broadcast
function on_event_quit(event)           end  -- Client disconnected
```

### Timers

```lua
function on_heartbeat_1s()   end    function on_heartbeat_5s()   end
function on_heartbeat_10s()  end    function on_heartbeat_15s()  end
function on_heartbeat_30s()  end    function on_heartbeat_1m()   end
function on_heartbeat_5m()   end    function on_heartbeat_10m()  end
function on_heartbeat_15m()  end    function on_heartbeat_20m()  end
function on_heartbeat_30m()  end    function on_heartbeat_60m()  end
```

## Protobuf Types

All message types from three packages are directly available in Lua:

```lua
-- implantpb messages
local req = ExecRequest.New({Path = "/bin/ls", Args = {"-la"}})

-- clientpb messages
local body = CommonBody.New({Name = "operation", StringArray = {"a", "b"}})

-- modulepb messages (module-related)
local mod = ModuleBody.New({Name = "my_module"})

-- Wrap and send via spite()
local task = execute_module(session, spite(body), "common")
```

### Accessing Protobuf Repeated Fields

```lua
-- Use numeric indices to iterate over repeated fields
local clr_versions = session.Os.ClrVersion
local i = 1
while true do
    local version = clr_versions[i]
    if version == nil then break end
    print(version)
    i = i + 1
end
```
