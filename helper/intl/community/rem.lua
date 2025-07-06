local function rem_path(arch, ext)
    return "rem/rem_community" .. "." .. arch .. "." .. ext
end


function load_rem()
    return load_module(active(), "rem", script_resource(rem_path(session.Os.Arch, "dll")))
end

local rem_load_cmd = command("rem_community:load", load_rem, "load rem with rem.dll", "")

function build_rem_cmdline(pipe, mod, remote_url, local_url)
    local link = rem_link(pipe)
    local args = { "-c", link, "-m", mod }
    if remote_url and remote_url ~= "" then
        table.insert(args, "-r")
        table.insert(args, remote_url)
    end
    if local_url and local_url ~= "" then

        table.insert(args, "-l")
        table.insert(args, local_url)
    end
    return args
end

function run_socks5(arg_0, flag_port, flag_user, flag_pass)
    return rem_dial(active(), arg_0,
        build_rem_cmdline(arg_0, "reverse", "socks5://" .. flag_user .. ":" .. flag_pass .. "@0.0.0.0:" .. flag_port, ""))
end

local rem_socks_cmd = command("rem_community:socks5", run_socks5, "serving socks5 with rem", "T1090")
bind_args_completer(rem_socks_cmd, { rem_completer() })

function run_rem_connect(arg_0)
    rem_dial(active(), arg_0, { "-c", rem_link(arg_0), "-n" })
end

local rem_connect_cmd = command("rem_community:connect", run_rem_connect, "connect to rem", "")
bind_args_completer(rem_connect_cmd, { rem_completer() })

function run_rem_fork(arg_0, arg_1, flag_mod, flag_remote_url, flag_local_url)
    local rpc = require("rpc")
    local task = rpc.RemAgentCtrl(active():Context(), ProtobufMessage.New("clientpb.REMAgent", {
        PipelineId = arg_0,
        Id = arg_1,
        Args = { "-r", flag_remote_url, "-l", flag_local_url, "-m", flag_mod },
    }))
end

local rem_fork = command("rem_community:fork", run_rem_fork, "fork rem", "")
bind_args_completer(rem_fork, { rem_completer(), rem_agent_completer() })


function run_rem(flag_pipe, args)
    local session = active()
    local arch = session.Os.Arch
    local path = rem_path(arch, "exe")
    table.insert(args, "-c")
    table.insert(args, rem_link(flag_pipe))
    return execute_exe(session, script_resource(rem_path), args, true, 600, arch, "", new_sac())
end

local rem_run_cmd = command("rem_community:run", run_rem, "run rem", "")
bind_flags_completer(rem_run_cmd, { pipe = rem_completer() })


function restart_rem_agent(arg_0, arg_1)
    local session = active()
    local path = rem_path(session.Os.Arch, "exe")
    local agent
    for k, v in pairs(pivots()) do
        if v.RemAgentId == arg_1 then
            agent = v
            break
        end
    end
    local args = { "-r", agent.RemoteURL, "-l", agent.LocalURL, "-m", agent.Mod, "-a", agent.RemAgentId }
    table.insert(args, "-c")
    table.insert(args, rem_link(arg_0))

    return execute_exe(session, script_resource(path), args, true, 600, arch, "", new_sac())
end

function get_rem_log(arg_0, arg_1)
    local rpc = require("rpc")
    local log = rpc.RemAgentLog(active():Context(), ProtobufMessage.New("clientpb.REMAgent", {
        Id = arg_1,
        PipelineId = arg_0,
    }))
    print(log.Log)
end

local log_cmd = command("rem_community:log", get_rem_log, "get rem log", "")
bind_args_completer(log_cmd, { rem_completer(), rem_agent_completer() })


function run_rem_stop(arg_0, arg_1)
    local rpc = require("rpc")
    local task = rpc.RemAgentStop(active():Context(), ProtobufMessage.New("clientpb.REMAgent", {
        PipelineId = arg_0,
        Id = arg_1,
    }))
end

local rem_stop_cmd = command("rem_community:stop", run_rem_stop, "stop rem", "")
bind_args_completer(rem_stop_cmd, { rem_completer(), rem_agent_completer() })
