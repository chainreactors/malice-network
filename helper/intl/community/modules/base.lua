
function load_preset(arg_0)
    local session = active()
    local arch = barch(session)

    return load_module(session, "arg_0", script_resource("modules/"  .. arg_0 .. "." .. arch .. ".dll"))
end

local load_preset_cmd = command("load_preset", load_preset, "load full|fs|execute|sys|rem precompiled modules", "")
bind_args_completer(load_preset_cmd, { values_completer({"full", "fs", "execute", "sys", "rem"}) })
