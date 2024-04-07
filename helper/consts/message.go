package consts

// module and command
const (
	ModuleUpdate           = "update"
	ModuleExecution        = "exec"
	ModuleExecuteAssembly  = "execute-assembly"
	ModuleExecuteShellcode = "execute-shellcode"
	ModuleExecuteBof       = "execute-bof"
	ModuleUpload           = "upload"
	ModuleDownload         = "download"
	ModuleCurl             = "curl"
	ModulePwd              = "pwd"      // TODO impl client
	ModuleLs               = "ls"       // TODO impl client
	ModuleCd               = "cd"       // TODO impl client
	ModuleMv               = "mv"       // TODO impl client
	ModuleMkdir            = "mkdir"    // TODO impl client
	ModuleRm               = "rm"       // TODO impl client
	ModuleCat              = "cat"      // TODO impl client
	ModulePs               = "ps"       // TODO impl client
	ModuleKill             = "kill"     // TODO impl client
	ModuleWhoami           = "whoami"   // TODO impl client
	ModuleEnv              = "env"      // TODO impl client
	ModuleSetEnv           = "setenv"   // TODO impl client
	ModuleUnsetEnv         = "unsetenv" // TODO impl client
	ModuleNetstat          = "netstat"

	CommandSync         = "sync"
	CommandBroadcast    = "broadcast"
	CommandVersion      = "version"
	CommandNotify       = "notify"
	CommandAlias        = "alias"
	CommandAliasLoad    = "load"
	CommandAliasInstall = "install"
	CommandAliasRemove  = "remove"
)

// ctrl type
const (
	CtrlPipelineStart = 0 + iota
	CtrlPipelineStop
)

// ctrl status
const (
	CtrlStatusSuccess = 0 + iota
	CtrlStatusFailed
)
