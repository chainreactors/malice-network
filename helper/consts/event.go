package consts

const (
	CtrlTaskCallback  = "task_callback"
	CtrlTaskFinish    = "task_finish"
	CtrlTaskCancel    = "task_cancel"
	CtrlTaskError     = "task_error"
	CtrlWebUpload     = "web_upload"
	CtrlListenerStart = "listener_start"
	CtrlListenerStop  = "listener_stop"
	CtrlSessionStart  = "session_start"
	CtrlSessionStop   = "session_stop"
	CtrlJobStart      = "job_start"
	CtrlJobStop       = "job_stop"
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
