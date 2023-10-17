package core

import (
	"google.golang.org/protobuf/proto"
)

func NewMessage(msg proto.Message, remoteAddr string, taskID string, messageID string, end bool) *Message {
	return &Message{
		Message: msg,
		//RemoteAddr: remoteAddr,
		TaskID:    taskID,
		MessageID: messageID,
		End:       end,
	}
}

type Message struct {
	proto.Message
	End       bool
	SessionID string
	TaskID    string
	MessageID string
}
