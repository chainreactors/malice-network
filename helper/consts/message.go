package consts

// plugin name

const (
	PluginBlock            = "block"
	PluginRegister         = "register"
	PluginUpload           = "upload"
	PluginDownload         = "download"
	PluginExec             = "exec"
	PluginExecuteAssembly  = "execute_assembly"
	PluginExecuteShellcode = "execute_shellcode"
	PluginExecuteSpawn     = "execute_spawn"
	PluginExecuteSideload  = "execute_sideload"
	PluginExecuteBof       = "execute_bof"
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
