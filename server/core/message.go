package core

import (
	"google.golang.org/protobuf/proto"
)

type Message struct {
	proto.Message
	End       bool
	SessionID string
	TaskID    uint32
	MessageID string
}
