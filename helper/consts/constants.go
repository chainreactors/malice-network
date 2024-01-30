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
)

// config
const (
	MaxPacketLength = "server.config.packet_length"
	AuditLevel      = "server.audit"
)

// plugin
const (
	CSharpPlugin    = "csharp"
	SideloadPlugin  = "sideload"
	SpawnPlugin     = "spawn"
	ShellcodePlugin = "shellcode"
)
