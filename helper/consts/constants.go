package consts

import "time"

const (
	MinTimeout = time.Duration(30 * time.Second)
)

const (
	ClientPrompt = "IoM"
)

const (
	KB                          = 1024
	MB                          = KB * 1024
	GB                          = MB * 1024
	BufSize                     = 2 * MB
	ClientMaxReceiveMessageSize = 2 * GB
	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2 * GB
)
