package consts

const (
	CalleeCMD = "cmd"
	CalleeMal = "mal"
	CalleeSDK = "sdk"
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
	CtrlSessionConsole  = "session_done"
	CtrlSessionError    = "session_finish"
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
