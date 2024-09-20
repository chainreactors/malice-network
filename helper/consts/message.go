package consts

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
	ModuleExplore              = "explorer"
	ModuleAliasShell           = "shell"
	ModuleAliasPowershell      = "powershell"
	ModuleExecution            = "exec"
	ModuleExecuteAssembly      = "execute_assembly"
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
	ModuleSetEnv               = "setenv"
	ModuleUnsetEnv             = "unsetenv"
	ModuleInfo                 = "info"
	ModuleNetstat              = "netstat"
	ModuleBypass               = "bypass"
	ModuleCurl                 = "curl"
	ModuleListModule           = "list_module"
	ModuleLoadModule           = "load_module"
	ModuleRefreshModule        = "refresh_module"
	ModuleListAddon            = "list_addon"
	ModuleLoadAddon            = "load_addon"
	ModuleExecuteAddon         = "execute_addon"
	ModuleClear                = "clear"
	ModuleCancelTask           = "cancel_task"
)

const (
	CommandLogin            = "login"
	CommandExit             = "exit"
	CommandSessions         = "sessions"
	CommandTasks            = "tasks"
	CommandFiles            = "files"
	CommandNote             = "note"
	CommandGroup            = "group"
	CommandObverse          = "obverse"
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
	CommandTcp              = "tcp"
	CommandWebsite          = "website"
	CommandListener         = "listener"
	CommandJob              = "job"
	CommandRegister         = "register"
	CommandPipelineStart    = "start"
	CommandPipelineStop     = "stop"
)

// task error
const (
	TaskErrorOperatorError       = 2
	TaskErrorNotExpectBody       = 3
	TaskErrorFieldRequired       = 4
	TaskErrorFieldLengthMismatch = 5
	TaskErrorFieldInvalid        = 6
	TaskError                    = 99
)
