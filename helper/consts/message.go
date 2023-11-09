package consts

// plugin name

const (
	PluginBlock    = "block"
	PluginRegister = "register"
	PluginUpload   = "upload"
	PluginDownload = "download"
	PluginExec     = "exec"
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
