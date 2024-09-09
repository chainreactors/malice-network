package consts

const (
	CtrlTaskCallback = "task_callback"
	CtrlTaskFinish   = "task_finish"
	CtrlTaskCancel   = "task_cancel"
	CtrlTaskError    = "task_error"
	CtrlWebUpload    = "web_upload"
)

// ctrl type
const (
	CtrlPipelineStart = 0 + iota
	CtrlPipelineStop
	CtrlWebsiteStart = 0 + iota
	CtrlWebsiteStop
	RegisterWebsite
)

// ctrl status
const (
	CtrlStatusSuccess = 0 + iota
	CtrlStatusFailed
)

// event
const (
	EventJoin      = "join"
	EventLeft      = "left"
	EventBroadcast = "broadcast"
	EventNotify    = "notify"
	EventPipeline  = "pipeline"
	EventSession   = "session"
	EventListener  = "listener"
	EventTask      = "task"
	EventWebsite   = "website"
)
