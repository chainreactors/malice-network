package types

import (
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"google.golang.org/protobuf/proto"
)

var (
	ErrUnknownSpite = errors.New("unknown spite body")
	ErrUnknownJob   = errors.New("unknown job body")
)

func BuildSpite(spite *commonpb.Spite, msg proto.Message) (*commonpb.Spite, error) {
	if spite == nil {
		spite = &commonpb.Spite{}
	}
	switch msg.(type) {
	case *commonpb.Block:
		spite.Name = "block"
		spite.Body = &commonpb.Spite_Block{Block: msg.(*commonpb.Block)}
	case *commonpb.Register:
		spite.Body = &commonpb.Spite_Register{Register: msg.(*commonpb.Register)}
	case *pluginpb.ExecRequest:
		spite.Name = "exec"
		spite.Body = &commonpb.Spite_ExecRequest{ExecRequest: msg.(*pluginpb.ExecRequest)}
	case *pluginpb.ExecResponse:
		spite.Body = &commonpb.Spite_ExecResponse{ExecResponse: msg.(*pluginpb.ExecResponse)}
	case *pluginpb.UploadRequest:
		spite.Name = "upload"
		spite.Body = &commonpb.Spite_UploadRequest{UploadRequest: msg.(*pluginpb.UploadRequest)}
	case *pluginpb.DownloadRequest:
		spite.Name = "download"
		spite.Body = &commonpb.Spite_DownloadRequest{DownloadRequest: msg.(*pluginpb.DownloadRequest)}
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

func BuildPipeline(msg proto.Message) *lispb.Pipeline {
	var pipeline = &lispb.Pipeline{}
	switch msg.(type) {
	case *lispb.TCPPipeline:
		pipeline.Body = &lispb.Pipeline_Tcp{Tcp: msg.(*lispb.TCPPipeline)}
	default:
		logs.Log.Debug(ErrUnknownJob.Error())
		return pipeline
	}
	return pipeline
}
