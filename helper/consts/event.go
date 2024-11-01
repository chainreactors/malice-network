package consts

const (
	CalleeCMD      = "cmd"
	CalleeMal      = "mal"
	CalleeSDK      = "sdk"
	CalleeExplorer = "explorer"
)

const (
	CtrlTaskCallback    = "task_callback"
	CtrlTaskFinish      = "task_finish"
	CtrlTaskCancel      = "task_cancel"
	CtrlTaskError       = "task_error"
	CtrlWebUpload       = "web_upload"
	CtrlListenerStart   = "listener_start"
	CtrlListenerStop    = "listener_stop"
	CtrlPipelineStart   = "pipeline_start"
	CtrlPipelineStop    = "pipeline_stop"
	CtrlWebsiteStart    = "website_start"
	CtrlWebsiteStop     = "website_stop"
	CtrlWebsiteRegister = "website_register"
	CtrlJobStart        = "job_start"
	CtrlJobStop         = "job_stop"
	CtrlSessionRegister = "session_register"
	CtrlSessionLog      = "session_log"
	CtrlSessionTask     = "session_task"
	CtrlSessionError    = "session_finish"
	CtrlSessionStop     = "session_stop"
)

// ctrl status
const (
	CtrlStatusSuccess = 0 + iota
	CtrlStatusFailed
)

// event
const (
	EventJoin        = "join"
	EventLeft        = "left"
	EventBroadcast   = "broadcast"
	EventNotify      = "notify"
	EventSession     = "session"
	EventListener    = "listener"
	EventTask        = "task"
	EventWebsite     = "website"
	EventTcpPipeline = "tcp"
	EventJob         = "job"
)
