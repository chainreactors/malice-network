package consts

import "time"

// Default config
const (
	MinTimeout                  = time.Duration(30 * time.Second)
	KB                          = 1024
	MB                          = KB * 1024
	GB                          = MB * 1024
	BufSize                     = 2 * MB
	ClientMaxReceiveMessageSize = 2 * GB
	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2 * GB
	DefaultTimeout       = 10 // second
	DefaultDuration      = time.Duration(DefaultTimeout * time.Second)
)

// UI
const (
	ClientPrompt = "IoM"
)

// Group
const (
	GenericGroup   = "generic"
	ImplantGroup   = "implant"
	AliasesGroup   = "alias"
	ExtensionGroup = "extension"
	SessionGroup   = "session"
)

// config
const (
	MaxPacketLength = "server.config.packet_length"
	AuditLevel      = "server.audit"
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
