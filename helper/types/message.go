package types

import (
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"google.golang.org/protobuf/proto"
)

type NMessage int

func (r NMessage) Message() proto.Message {
	if MessageMap[r] != nil {
		return MessageMap[r]
	} else {
		return nil
	}
}

const (
	MsgKnown NMessage = 0
	MsgSpite NMessage = 1 + iota
	MsgPromise
	MsgRegister
	MsgPing
)

var (
	MessageMap = map[NMessage]proto.Message{
		MsgRegister: &commonpb.Register{},
	}
)

func MessageType(message proto.Message) NMessage {
	switch message.(type) {
	case *commonpb.Register:
		return MsgRegister
	default:
		return MsgKnown
	}
}
