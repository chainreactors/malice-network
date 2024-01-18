package consts

// module and command
const (
	ModuleUpdate          = "update"
	ModuleExecution       = "exec"
	ModuleExecuteAssembly = "execute-assembly"
	ModuleUpload          = "upload"
	ModuleDownload        = "download"
	CommandBroadcast      = "broadcast"
	CommandVersion        = "version"
	CommandNotify         = "notify"
	CommandAlias          = "alias"
	CommandAliasLoad      = "load"
	CommandAliasInstall   = "install"
	CommandAliasRemove    = "remove"
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
