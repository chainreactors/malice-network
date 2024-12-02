package consts

import "strings"

var (
	ModuleAliases = map[string]string{
		ModuleAliasInlineShellcode: ModuleExecuteShellcode,
		ModuleAliasInlineExe:       ModuleExecuteExe,
		ModuleAliasInlineDll:       ModuleExecuteDll,
		ModuleAliasShell:           ModuleExecution,
		ModuleAliasPowershell:      ModuleExecution,
	}
	ExecuteModules = []string{
		ModuleExecuteBof, ModuleExecuteDll, ModuleExecuteShellcode,
		ModuleExecuteExe, ModulePowerpick, ModuleExecuteAssembly,
		ModuleAliasInlineExe, ModuleAliasInlineDll, ModuleAliasInlineShellcode,
	}
	InlineModules = []string{
		ModuleAliasInlineExe, ModuleAliasInlineDll, ModuleAliasInlineShellcode,
	}
	SacrificeModules = []string{
		ModuleExecuteExe, ModuleExecuteDll, ModuleExecuteShellcode,
	}
)

// client module and command
const (
	// internal
	ModulePing       = "ping"
	ModuleClear      = "clear"
	ModuleCancelTask = "cancel_task"
	ModuleSleep      = "sleep"
	ModuleSuicide    = "suicide"
	ModuleInit       = "init"
	//execute
	ModuleAliasShell           = "shell"
	ModuleAliasPowershell      = "powershell"
	ModuleExecution            = "exec"
	ModuleExecuteLocal         = "execute_local"
	ModuleExecuteAssembly      = "execute_assembly"
	ModuleInlineAssembly       = "inline_assembly"
	ModuleExecuteShellcode     = "execute_shellcode"
	ModuleAliasInlineShellcode = "inline_shellcode"
	ModuleExecuteExe           = "execute_exe"
	ModuleAliasInlineExe       = "inline_exe"
	ModuleExecuteDll           = "execute_dll"
	ModuleAliasInlineDll       = "inline_dll"
	ModuleExecuteBof           = "bof"
	ModulePowerpick            = "powerpick"
	ModuleUpload               = "upload"
	ModuleDownload             = "download"
	ModulePwd                  = "pwd"
	ModuleLs                   = "ls"
	ModuleCd                   = "cd"
	ModuleMv                   = "mv"
	ModuleMkdir                = "mkdir"
	ModuleRm                   = "rm"
	ModuleCat                  = "cat"
	ModulePs                   = "ps"
	ModuleCp                   = "cp"
	ModuleChmod                = "chmod"
	ModuleChown                = "chown"
	ModuleKill                 = "kill"
	ModuleWhoami               = "whoami"
	ModuleEnv                  = "env"
	ModuleSetEnv               = "env_set"
	ModuleUnsetEnv             = "env_unset"

	ModuleSysInfo = "sysinfo"
	ModuleNetstat = "netstat"
	ModuleBypass  = "bypass"
	ModuleCurl    = "curl"

	// module
	ModuleListModule    = "list_module"
	ModuleLoadModule    = "load_module"
	ModuleRefreshModule = "refresh_module"

	// addon
	ModuleListAddon    = "list_addon"
	ModuleLoadAddon    = "load_addon"
	ModuleExecuteAddon = "execute_addon"

	// registry
	ModuleRegQuery     = "reg_query"
	ModuleRegAdd       = "reg_add"
	ModuleRegDelete    = "reg_delete"
	ModuleRegListKey   = "reg_list_key"
	ModuleRegListValue = "reg_list_value"

	// service
	ModuleServiceList   = "service_list"
	ModuleServiceCreate = "service_create"
	ModuleServiceQuery  = "service_query"
	ModuleServiceStart  = "service_start"
	ModuleServiceStop   = "service_stop"
	ModuleServiceDelete = "service_delete"

	// taskschd
	ModuleTaskSchdList   = "taskschd_list"
	ModuleTaskSchdCreate = "taskschd_create"
	ModuleTaskSchdQuery  = "taskschd_query"
	ModuleTaskSchdStart  = "taskschd_start"
	ModuleTaskSchdStop   = "taskschd_stop"
	ModuleTaskSchdDelete = "taskschd_delete"
	ModuleTaskSchdRun    = "taskschd_run"

	// wmi
	ModuleWmiQuery = "wmi_query"
	ModuleWmiExec  = "wmi_execute"

	// pipe
	ModulePipeUpload = "pipe_upload"
	ModulePipeClose  = "pipe_close"
	ModulePipeRead   = "pipe_read"

	// privilege
	ModuleRunas     = "runas"
	ModulePrivs     = "privs"
	ModuleGetSystem = "getsystem"
)

func SubCommandName(module string) string {
	i := strings.Index(module, "_")
	if i == -1 {
		return module
	} else {
		return module[i+1:]
	}
}

const (
	CommandLogin            = "login"
	CommandExit             = "exit"
	CommandWait             = "wait"
	CommandRecover          = "recover"
	CommandPolling          = "polling"
	CommandSessions         = "sessions"
	CommandNewBindSession   = "newbind"
	CommandTasks            = "tasks"
	CommandFiles            = "files"
	CommandExplore          = "explorer"
	CommandSession          = "session"
	CommandSessionNote      = "note"
	CommandSessionGroup     = "group"
	CommandObverse          = "obverse"
	CommandHistory          = "history"
	CommandDelSession       = "del"
	CommandUse              = "use"
	CommandBackground       = "background"
	CommandSync             = "sync"
	CommandBroadcast        = "broadcast"
	CommandVersion          = "version"
	CommandNotify           = "notify"
	CommandAlias            = "alias"
	CommandAliasLoad        = "load"
	CommandAliasList        = "list"
	CommandAliasInstall     = "install"
	CommandAliasRemove      = "remove"
	CommandArmory           = "armory"
	CommandArmoryUpdate     = "update"
	CommandArmorySearch     = "search"
	CommandArmoryLoad       = "load"
	CommandArmoryInstall    = "install"
	CommandExtension        = "extension"
	CommandExtensionList    = "list"
	CommandExtensionLoad    = "load"
	CommandExtensionInstall = "install"
	CommandExtensionRemove  = "remove"
	CommandMal              = "mal"
	CommandMalLoad          = "load"
	CommandMalList          = "list"
	CommandMalInstall       = "install"
	CommandMalRemove        = "remove"
	CommandMalRefresh       = "refresh"
	CommandPipelineTcp      = "tcp"
	CommandPipelineBind     = "bind"
	CommandWebsite          = "website"
	CommandListener         = "listener"
	CommandJob              = "job"
	CommandPipeline         = "pipeline"
	CommandPipelineNew      = "new"
	CommandPipelineList     = "list"
	CommandPipelineStart    = "start"
	CommandPipelineStop     = "stop"
	CommandPipelineDelete   = "delete"
	CommandBuild            = "build"
	CommandBuildPrelude     = "prelude"
	CommandBuildBeacon      = "beacon"
	CommandBuildBind        = "bind"
	CommandBuildShellCode   = "shellcode"
	CommandBuildModules     = "modules"
	CommandBuildPulse       = "pulse"
	CommandBuildLog         = "log"
	CommandArtifact         = "artifact"
	CommandArtifactList     = "list"
	CommandArtifactDownload = "download"
	CommandArtifactUpload   = "upload"
	CommandProfile          = "profile"
	CommandProfileList      = "list"
	CommandProfileLoad      = "load"
	CommandSRDI             = "srdi"
	CommandReg              = "reg"
	CommandRegExplorer      = "reg_explorer"
	CommandService          = "service"
	CommandTaskSchd         = "taskschd"
	CommandPipe             = "pipe"
	CommandAction           = "action"
	CommandActionRun        = "run"
	CommandActionEnable     = "enable"
	CommandActionDisable    = "disable"
	CommandActionList       = "list"
)

const (
	OPSecLOW   = "low"
	OPSecMID   = "mid"
	OPSecHIGH  = "high"
	OPSecOPsec = "opsec"
)
