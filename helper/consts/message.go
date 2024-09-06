package consts

// client module and command
const (
	ModuleUpdate               = "update"
	ModuleExplore              = "explorer"
	ModuleExecution            = "exec"
	ModuleExecuteAssembly      = "execute_assembly"
	ModuleExecuteShellcode     = "execute_shellcode"
	ModuleAliasInlineShellcode = "inline_shellcode"
	ModuleExecutePE            = "execute_pe"
	ModuleAliasInlinePE        = "inline_pe"
	ModuleExecuteDll           = "execute_dll"
	ModuleAliasInlineDll       = "inline_dll"
	ModuleExecuteBof           = "bof"
	ModulePowershell           = "powershell"
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
	ModuleCurl                 = "curl"
	ModuleListModule           = "list_module"
	ModuleLoadModule           = "load_module"
	ModuleRefreshModule        = "refresh_module"
	ModuleListAddon            = "list_addon"
	ModuleLoadAddon            = "load_addon"
	ModuleExecuteAddon         = "execute_addon"
	ModuleClear                = "clear"
)

const (
	CommandSessions         = "sessions"
	CommandNote             = "note"
	CommandGroup            = "group"
	CommandObverse          = "obverse"
	CommandDelSession       = "del"
	CommandUse              = "use"
	CommandBackgroup        = "backgroup"
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
)

var ModuleAliases = map[string]string{
	ModuleAliasInlineShellcode: ModuleExecuteShellcode,
	ModuleAliasInlinePE:        ModuleExecutePE,
	ModuleAliasInlineDll:       ModuleExecuteDll,
}

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
