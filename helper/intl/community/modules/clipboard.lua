local function bof_path(bof_name, arch)
    return "bof/enum/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end

-- dump_clipboard
local function run_dump_clipboard(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("dump_clipboard", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_dump_clipboard = command("clipboard:dump", run_dump_clipboard, "Dump clipboard content", "T1115")
opsec("clipboard:dump", 9.0)