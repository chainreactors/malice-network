-- Example embedded mal plugin
-- This demonstrates the structure and functionality of an embedded mal plugin
local time = require("time")
function bof_pack(format, ...)
    local args = {...}
    return pack_bof_args(format, args)
end
function read(filename)
    local file = io.open(filename, "r")
    if not file then
        print("file not found")
        return nil
    end
    local content = file:read("*all")
    file:close()
    return content
end
function new_sac()
    local sac = new_sacrifice(0, false, false, false, "")
    return sac
end


local function bof_path(bof_name, arch)
    return "bof/" .. bof_name .. "." .. arch .. ".o"
end
local function command_register(command_name, command_function, help_string, ttp)
    command(command_name, command_function, help_string, ttp)
end

-- screenshot
local function run_screenshot(args)
    local filename
    if #args == 1 then
        filename = args[1]
    else
        filename = "screenshot.jpg"
    end
    local packed_args = bof_pack("z", filename)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("screenshot", arch)
    local result = bof(session, script_resource(bof_file), packed_args, true)
    return result
end
command_register("screenshot_bof", run_screenshot,
                 "Command: situational screenshot <filename>", "T1113")