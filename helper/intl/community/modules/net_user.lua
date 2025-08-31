local function bof_path(bof_name, arch)
    return "bof/net_user/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end
-- add_net_user
local function run_add_net_user(cmd)
    local username = cmd:Flags():GetString("username")
    local password = cmd:Flags():GetString("password")
    if username == "" then
        error("username is required")
    end
    if password == "" then
        error("password is required")
    end
    local packed_args = bof_pack("ZZ", username, password)
    local session = active()
    local arch = session.Os.Arch
    if not isadmin(session) then
        error("You need to be an admin to run this command")
    end
    local bof_file = bof_path("add_net_user", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_add_net_user = command("net:user:add", run_add_net_user, "Add a new user account <username> <password>", "T1136.001")
cmd_add_net_user:Flags():String("username", "", "the username to add")
cmd_add_net_user:Flags():String("password", "", "the password to set")

opsec("net:user:add", 9.0)

-- enum_net_user
local function run_enum_net_user(cmd)
    local enumtype = cmd:Flags():GetString("type")
    local type_map = {all = 1, locked = 2, disabled = 3, active = 4}
    local _type = type_map[enumtype:lower()]
    if _type == nil then
        error("Parameter must be one of: [all, locked, disabled, active]")
    end

    local packed_args = bof_pack("ii", 0, _type)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_net_user", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_enum_net_user = command("net:user:enum", run_enum_net_user, "Enumerate network users [type]", "T1087.002")
cmd_enum_net_user:Flags():String("type", "all", "enumeration type (all, locked, disabled, active)")
opsec("net:user:enum", 9.0)

-- query_net_user
local function run_query_net_user(cmd)
    local username = cmd:Flags():GetString("username")
    local domain = cmd:Flags():GetString("domain")
    if username == "" then
        error("username is required")
    end
    local packed_args = bof_pack("ZZ", username, domain)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("query_net_user", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_query_net_user = command("net:user:query", run_query_net_user, "Query user information <username> [domain]", "T1087.002")
cmd_query_net_user:Flags():String("username", "", "username to query")
cmd_query_net_user:Flags():String("domain", "", "domain name (optional)")
opsec("net:user:query", 9.0)