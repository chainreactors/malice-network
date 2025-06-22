-- Example embedded mal plugin
-- This demonstrates the structure and functionality of an embedded mal plugin

local example = {}

-- hello command - basic example
function example.parse_hello(args)
    local name = "World"
    if #args == 1 then
        name = args[1]
    elseif #args > 1 then
        error("Only 0 or 1 arguments are allowed")
    end
    return name
end

function example.run_hello(args)
    local session = active()
    local name = example.parse_hello(args)

    print("Hello, " .. name .. "!")
    print("Session ID: " .. session.ID)
    print("Target OS: " .. session.Os.Name)
    print("Architecture: " .. session.Os.Arch)

    return "Hello command executed successfully"
end

command("example:hello", example.run_hello,
    "Command: example hello [name]", "Example")

-- info command - system information gathering
function example.parse_info(args)
    if #args ~= 0 then
        error("0 arguments are allowed")
    end
    return args
end

function example.run_info(args)
    local session = active()
    example.parse_info(args)

    print("=== System Information ===")
    print("Session ID: " .. session.ID)
    print("Process Name: " .. session.Process.Name)
    print("Process ID: " .. session.Process.Pid)
    print("Process Path: " .. session.Process.Path)
    print("User: " .. session.Process.User)
    print("Operating System: " .. session.Os.Name)
    print("Architecture: " .. session.Os.Arch)
    print("Transport: " .. session.Transport)
    print("Remote Address: " .. session.RemoteAddr)
    print("Last Message: " .. session.LastMessage)

    return "System information collected"
end

command("example:info", example.run_info,
    "Command: example info", "T1082")

-- test command - demonstrates parameter handling
function example.parse_test(args)
    if #args < 1 or #args > 3 then
        error("1 to 3 arguments are required")
    end

    local action = args[1]
    local param1 = ""
    local param2 = ""

    if #args >= 2 then
        param1 = args[2]
    end
    if #args == 3 then
        param2 = args[3]
    end

    return { action = action, param1 = param1, param2 = param2 }
end

function example.run_test(args)
    local session = active()
    local params = example.parse_test(args)

    print("=== Test Command ===")
    print("Action: " .. params.action)
    print("Parameter 1: " .. params.param1)
    print("Parameter 2: " .. params.param2)

    if params.action == "echo" then
        print("Echo: " .. params.param1 .. " " .. params.param2)
    elseif params.action == "reverse" then
        print("Reversed: " .. string.reverse(params.param1))
    elseif params.action == "upper" then
        print("Uppercase: " .. string.upper(params.param1))
    else
        print("Unknown action: " .. params.action)
    end

    return "Test command completed"
end

command("example:test", example.run_test,
    "Command: example test <action> [param1] [param2]", "Example")

-- Event handler example
function example.on_session_connect(event)
    print("Example plugin: Session connected - " .. tostring(event))
end

-- Register event handler (if the event system supports it)
-- register_event("session_connect", example.on_session_connect)

-- BOF example (if BOF functionality is available)
function example.parse_bof_demo(args)
    if #args ~= 0 then
        error("0 arguments are allowed")
    end
    return ""
end

function example.run_bof_demo(args)
    local session = active()
    example.parse_bof_demo(args)

    -- This is a placeholder for BOF functionality
    -- In a real implementation, you would have:
    -- local arch = session.Os.Arch
    -- return bof(session, script_resource("demo." .. arch .. ".o"), args, true)

    print("BOF demo would execute here")
    print("Architecture: " .. session.Os.Arch)

    return "BOF demo completed (placeholder)"
end

command("example:bof", example.run_bof_demo,
    "Command: example bof", "Example")

-- Help command
function example.run_help(args)
    print("=== Example Plugin Help ===")
    print("Available commands:")
    print("  example:hello [name]           - Say hello to someone")
    print("  example:info                   - Show system information")
    print("  example:test <action> [params] - Test parameter handling")
    print("  example:bof                    - BOF demonstration")
    print("  example:help                   - Show this help")
    print("")
    print("Actions for test command:")
    print("  echo <text1> <text2>  - Echo the parameters")
    print("  reverse <text>        - Reverse the text")
    print("  upper <text>          - Convert to uppercase")

    return "Help displayed"
end

command("example:help", example.run_help,
    "Command: example help", "Example")

print("Example embedded mal plugin loaded successfully (Community Version)")
print("Available commands: example:hello, example:info, example:test, example:bof, example:help")
