package consts

import "time"

// Default config
const (
	MinTimeout                  = time.Duration(30 * time.Second)
	KB                          = 1024
	MB                          = KB * 1024
	GB                          = MB * 1024
	BufSize                     = 2 * MB
	ClientMaxReceiveMessageSize = 256 * MB
	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2 * GB
	DefaultTimeout       = 10 // second
	DefaultDuration      = time.Duration(DefaultTimeout * time.Second)
)

// UI
const (
	ClientPrompt = "IoM"
)

const (
	ClientMenu  = "client"
	ImplantMenu = "implant"
)

// Group
const (
	GenericGroup  = "generic"
	ImplantGroup  = "implant"
	ArmoryGroup   = "armory"
	AddonGroup    = "addon"
	MalGroup      = "mal"
	ListenerGroup = "listener"
)

// config
const (
	MaxPacketLength = "server.config.packet_length"
	AuditLevel      = "server.audit"
)

const (
	UnknownFile = iota
	EXEFile
	DLLFile
)

// Time
const (
	DefaultMaxBodyLength   = 2 * 1024 * 1024 * 1024 // 2Gb
	DefaultHTTPTimeout     = time.Minute
	DefaultLongPollTimeout = time.Second
	DefaultLongPollJitter  = time.Second
	minPollTimeout         = time.Second
	DefaultCacheJitter     = 60 * 60
)
