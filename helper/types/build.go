package types

import (
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
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
		spite.Name = consts.PluginBlock
		spite.Body = &commonpb.Spite_Block{Block: msg.(*commonpb.Block)}
	case *commonpb.Register:
		spite.Body = &commonpb.Spite_Register{Register: msg.(*commonpb.Register)}
	case *pluginpb.ExecRequest:
		spite.Name = consts.PluginExec
		spite.Body = &commonpb.Spite_ExecRequest{ExecRequest: msg.(*pluginpb.ExecRequest)}
	case *pluginpb.ExecResponse:
		spite.Body = &commonpb.Spite_ExecResponse{ExecResponse: msg.(*pluginpb.ExecResponse)}
	case *pluginpb.UploadRequest:
		spite.Name = consts.PluginUpload
		spite.Body = &commonpb.Spite_UploadRequest{UploadRequest: msg.(*pluginpb.UploadRequest)}
	case *pluginpb.DownloadRequest:
		spite.Name = consts.PluginDownload
		spite.Body = &commonpb.Spite_DownloadRequest{DownloadRequest: msg.(*pluginpb.DownloadRequest)}
	case *pluginpb.ExecuteAssembly:
		spite.Name = consts.PluginExecuteAssembly
		spite.Body = &commonpb.Spite_ExecuteAssembly{ExecuteAssembly: msg.(*pluginpb.ExecuteAssembly)}
	case *pluginpb.ExecuteShellcode:
		spite.Name = consts.PluginExecuteShellcode
		spite.Body = &commonpb.Spite_ExecuteShellcode{ExecuteShellcode: msg.(*pluginpb.ExecuteShellcode)}
	case *pluginpb.ExecuteSpawn:
		spite.Name = consts.PluginExecuteSpawn
		spite.Body = &commonpb.Spite_ExecuteSpawn{ExecuteSpawn: msg.(*pluginpb.ExecuteSpawn)}
	case *pluginpb.ExecuteSideLoad:
		spite.Name = consts.PluginExecuteSideload
		spite.Body = &commonpb.Spite_ExecuteSideload{ExecuteSideload: msg.(*pluginpb.ExecuteSideLoad)}
	case *pluginpb.ExecuteBof:
		spite.Name = consts.PluginExecuteBof
		spite.Body = &commonpb.Spite_ExecuteBof{ExecuteBof: msg.(*pluginpb.ExecuteBof)}
	case *pluginpb.AssemblyResponse:
		spite.Body = &commonpb.Spite_AssemblyResponse{AssemblyResponse: msg.(*pluginpb.AssemblyResponse)}
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
	case *commonpb.Spite_ExecResponse:
		return spite.GetExecResponse(), nil
	case *commonpb.Spite_AssemblyResponse:
		return spite.GetAssemblyResponse(), nil
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
