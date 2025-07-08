local function bof_path(bof_name, arch)
    return "bof/enum/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end
-- enum_dotnet
local function run_dotnet_enum(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_dotnet", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_dotnet_enum = command("enum:dotnet_process", run_dotnet_enum, "Find processes that most likely have .NET loaded.", "T1033")
opsec("enum:dotnet_process", 9.0)

-- enum_drives
local function run_enum_drives(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_drives", arch)
    return bof(session, script_resource(bof_file), {}, true)
end
local cmd_enum_drives = command("enum:drives", run_enum_drives, "Enumerate system drives", "T1135")
opsec("enum:drives", 9.0)

-- enum_files
local function run_enum_files(cmd)
    local directory = cmd:Flags():GetString("directory")
    local pattern = cmd:Flags():GetString("pattern")
    local keyword = cmd:Flags():GetString("keyword")
    if directory == "" then
        error("directory is required")
    end
    if pattern == "" then
        error("search pattern is required")
    end
    local packed_args = bof_pack("zzz", directory, pattern, keyword)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_files", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_enum_files = command("enum:files", run_enum_files, "Enumerate files <directory> <pattern> [keyword]", "T1083")
cmd_enum_files:Flags():String("directory", "", "directory path to search")
cmd_enum_files:Flags():String("pattern", "", "search pattern (e.g., *.txt)")
cmd_enum_files:Flags():String("keyword", "", "optional keyword filter")
opsec("enum:files", 9.0)

-- enum_localcert
local function run_enum_localcert(cmd)
    local store = cmd:Flags():GetString("store")
    if store == "" then
        error("certificate store name is required")
    end
    local packed_args = bof_pack("Z", store)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_localcert", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_enum_localcert = command("enum:localcert", run_enum_localcert, "Enumerate local certificates <store>", "T1553.003")
cmd_enum_localcert:Flags():String("store", "", "certificate store name")
opsec("enum:localcert", 9.0)

-- enum_localsessions
local function run_enum_localsessions(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_localsessions", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_localsessions = command("enum:localsessions", run_enum_localsessions, "Enumerate local user sessions", "T1033")
opsec("enum:localsessions", 9.0)

-- enum_dns
local function run_enum_dns(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_dns", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_dns = command("enum:dns", run_enum_dns, "Enum DNS configuration", "T1016")
opsec("enum:dns", 9.0)

-- enum_dc
local function run_enum_dc(cmd)
    local session = active()
    local arch = session.Os.Arch

    if arch ~= "x64" then
        error("x86 is not supported")
    end

    local bof_file = bof_path("enum_dc", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_dc = command("enum:dc", run_enum_dc, "Enumerate domain information using Active Directory Domain Services", "T1018")
opsec("enum:dc", 9.0)

-- enum_arp
local function run_arp()
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_arp", arch)
    return bof(session, script_resource(bof_file), {}, true)
end
local cmd_enum_arp = command("enum:arp", run_arp, "Enum ARP table", "T1016")
opsec("enum:arp", 9.0)