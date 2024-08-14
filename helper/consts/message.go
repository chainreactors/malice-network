package consts

// client module and command
const (
	ModuleUpdate           = "update"
	ModuleExecution        = "exec"
	ModuleExecuteAssembly  = "execute_assembly"
	ModuleInlineAssembly   = "inline_assembly"
	ModuleExecuteShellcode = "execute_shellcode"
	ModuleInlineShellcode  = "inline_shellcode"
	ModuleExecutePE        = "execute_pe"
	ModuleInlinePE         = "inline_pe"
	ModuleExecuteDll       = "execute_dll"
	ModuleInlineDll        = "inline_dll"
	ModuleExecuteBof       = "execute_bof"
	ModulePowershell       = "powershell"
	ModuleUpload           = "upload"
	ModuleDownload         = "download"
	ModulePwd              = "pwd"
	ModuleLs               = "ls"
	ModuleCd               = "cd"
	ModuleMv               = "mv"
	ModuleMkdir            = "mkdir"
	ModuleRm               = "rm"
	ModuleCat              = "cat"
	ModulePs               = "ps"
	ModuleCp               = "cp"
	ModuleChmod            = "chmod"
	ModuleChown            = "chown"
	ModuleKill             = "kill"
	ModuleWhoami           = "whoami"
	ModuleEnv              = "env"
	ModuleSetEnv           = "setenv"
	ModuleUnsetEnv         = "unsetenv"
	ModuleInfo             = "info"
	ModuleNetstat          = "netstat"
	ModuleCurl             = "curl"
	ModuleListModule       = "list_module"
	ModuleLoadModule       = "load_module"
	ModuleListExtension    = "list_extension"
	ModuleLoadExtension    = "load_extension"
	ModuleExecuteExtension = "execute_extension"
)

const (
	CommandSync             = "sync"
	CommandBroadcast        = "broadcast"
	CommandVersion          = "version"
	CommandNotify           = "notify"
	CommandAlias            = "alias"
	CommandAliasLoad        = "load"
	CommandAliasInstall     = "install"
	CommandAliasRemove      = "remove"
	CommandArmory           = "armory"
	CommandArmoryUpdate     = "update"
	CommandArmorySearch     = "search"
	CommandArmoryLoad       = "load"
	CommandExtension        = "extension"
	CommandExtensionList    = "list"
	CommandExtensionLoad    = "load"
	CommandExtensionInstall = "install"
	CommandExtensionRemove  = "remove"
)

// ctrl type
const (
	CtrlPipelineStart = 0 + iota
	CtrlPipelineStop
	CtrlWebsiteStart = 0 + iota
	CtrlWebsiteStop
)

// ctrl status
const (
	CtrlStatusSuccess = 0 + iota
	CtrlStatusFailed
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
