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
	ModulePwd              = "pwd"
	ModuleLs               = "ls"

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
