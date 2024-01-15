package common

import (
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"google.golang.org/protobuf/proto"
)

type Message int

func (r Message) Message() proto.Message {
	if MessageMap[r] != nil {
		return MessageMap[r]
	} else {
		return nil
	}
}

const (
	MsgKnown Message = 0 + iota
	MsgRegister
	MsgExec
	MsgUpload
	MsgDownload
	MsgAsyncStatus
	MsgAsyncAck
	MsgBlock
	MsgPing
)

var (
	MessageMap = map[Message]proto.Message{
		MsgRegister: &commonpb.Register{},
	}
)

func MessageType(message *commonpb.Spite) Message {
	switch message.Body.(type) {
	case *commonpb.Spite_Register:
		return MsgRegister
	case *commonpb.Spite_ExecRequest, *commonpb.Spite_ExecResponse:
		return MsgExec
	case *commonpb.Spite_UploadRequest:
		return MsgUpload
	case *commonpb.Spite_DownloadRequest:
		return MsgDownload
	//case *commonpb.Spite_AsyncStatus:
	//	return MsgAsyncStatus
	case *commonpb.Spite_AsyncAck:
		return MsgAsyncAck
	case *commonpb.Spite_Block:
		return MsgBlock
	default:
		return MsgKnown
	}
}
