local function bof_path(bof_name, arch)
    return "bof/route/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end
-- route_print
local function run_route_print()
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("route_print", arch)
    print(script_resource(bof_file))
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_route_print = command("route:print", run_route_print, "Display routing table", "T1016")
opsec("route:print", 9.0)