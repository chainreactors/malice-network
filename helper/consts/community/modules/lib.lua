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

