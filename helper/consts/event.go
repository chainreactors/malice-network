package consts

// client
const (
	// UpdateStr - "update"
	UpdateStr = "update"
	// VersionStr - "version"
	VersionStr         = "version"
	ExecutionStr       = "exec"
	ExecuteAssemblyStr = "execute-assembly"
	UploadStr          = "upload"
	DownloadStr        = "download"
	BroadcastStr       = "broadcast"
	NotifyStr          = "notify"
	AliasStr           = "alias"
	AliasLoadStr       = "load"
	AliasInstallStr    = "install"
	AliasRemoveStr     = "remove"
)

// event
const (
	EventJoin          = "event"
	EventLeft          = "left"
	EventBroadcast     = "broadcast"
	EventNotify        = "notify"
	EventPipelineStart = "pipeline_start"
	EventPipelineError = "pipeline_error"
	EventPipelineStop  = "pipeline_stop"
	EventTaskCallback  = "task_callback"
	EventTaskDone      = "task_done"
)
