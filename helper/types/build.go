package types

import (
	"errors"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"google.golang.org/protobuf/proto"
)

var (
	ErrUnknownSpite = errors.New("unknown spite body")
)

func BuildSpite(spite *commonpb.Spite, msg proto.Message) (*commonpb.Spite, error) {
	switch msg.(type) {
	case *commonpb.Register:
		spite.Body = &commonpb.Spite_Register{Register: msg.(*commonpb.Register)}
	case *pluginpb.ExecRequest:
		spite.Body = &commonpb.Spite_ExecRequest{ExecRequest: msg.(*pluginpb.ExecRequest)}
	case *pluginpb.ExecResponse:
		spite.Body = &commonpb.Spite_ExecResponse{ExecResponse: msg.(*pluginpb.ExecResponse)}
	default:
		return spite, ErrUnknownSpite
	}
	return spite, nil
}

func BuildSpites(spites []*commonpb.Spite) *commonpb.Spites {
	return &commonpb.Spites{Spites: spites}
}

func ParseSpite(spite *commonpb.Spite) (proto.Message, error) {
	switch spite.Body.(type) {
	case *commonpb.Spite_Register:
		return spite.GetRegister(), nil
	case *commonpb.Spite_ExecRequest:
		return spite.GetExecRequest(), nil
	case *commonpb.Spite_ExecResponse:
		return spite.GetExecResponse(), nil
	default:
		return nil, ErrUnknownSpite
	}
}
