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
	CtrlClientJoin      = "client_join"
	CtrlClientLeft      = "client_left"
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
	CtrlSessionDead     = "session_dead"
	CtrlSessionInit     = "session_init"
	CtrlSessionReborn   = "session_reborn"
	CtrlSessionLog      = "session_log"
	CtrlSessionTask     = "session_task"
	CtrlSessionError    = "session_error"
	CtrlSessionStop     = "session_stop"
	CtrlSessionCheckin  = "session_checkin"
)

const (
	CtrlHeartbeat1s  = "heartbeat_1s"  // 每秒触发
	CtrlHeartbeat5s  = "heartbeat_5s"  // 每5秒触发
	CtrlHeartbeat10s = "heartbeat_10s" // 每10秒触发
	CtrlHeartbeat15s = "heartbeat_15s" // 每15秒触发
	CtrlHeartbeat30s = "heartbeat_30s" // 每30秒触发
	CtrlHeartbeat1m  = "heartbeat_1m"  // 每分钟触发
	CtrlHeartbeat5m  = "heartbeat_5m"  // 每5分钟触发
	CtrlHeartbeat10m = "heartbeat_10m" // 每10分钟触发
	CtrlHeartbeat15m = "heartbeat_15m" // 每15分钟触发
	CtrlHeartbeat20m = "heartbeat_20m" // 每20分钟触发
	CtrlHeartbeat30m = "heartbeat_30m" // 每30分钟触发
	CtrlHeartbeat60m = "heartbeat_60m" // 每60分钟触发
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
	EventClient    = "client"
	EventBroadcast = "broadcast"
	EventNotify    = "notify"
	EventSession   = "session"
	EventListener  = "listener"
	EventTask      = "task"
	EventWebsite   = "website"
	EventPipeline  = "pipeline"
	EventJob       = "job"
	EventHeartbeat = "heartbeat"
)
