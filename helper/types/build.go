package types

import (
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"google.golang.org/protobuf/proto"
)

var (
	ErrUnknownSpite = errors.New("unknown spite body")
	ErrUnknownJob   = errors.New("unknown job body")
)

func BuildPingSpite() *implantpb.Spites {
	return BuildOneSpites(&implantpb.Spite{
		Body: &implantpb.Spite_Ping{},
	})
}

func BuildSpite(spite *implantpb.Spite, msg proto.Message) (*implantpb.Spite, error) {
	switch msg.(type) {
	case *implantpb.Request:
		spite.Name = msg.(*implantpb.Request).Name
		spite.Body = &implantpb.Spite_Request{Request: msg.(*implantpb.Request)}
	case *implantpb.ImplantTask:
		spite.Name = msg.(*implantpb.ImplantTask).Op
		spite.Body = &implantpb.Spite_Task{Task: msg.(*implantpb.ImplantTask)}
	case *implantpb.Block:
		spite.Name = MsgBlock.String()
		spite.Body = &implantpb.Spite_Block{Block: msg.(*implantpb.Block)}
	case *implantpb.Register:
		spite.Name = MsgRegister.String()
		spite.Body = &implantpb.Spite_Register{Register: msg.(*implantpb.Register)}
	case *implantpb.ExecRequest:
		spite.Name = MsgExec.String()
		spite.Body = &implantpb.Spite_ExecRequest{ExecRequest: msg.(*implantpb.ExecRequest)}
	case *implantpb.ExecResponse:
		spite.Name = MsgExec.String()
		spite.Body = &implantpb.Spite_ExecResponse{ExecResponse: msg.(*implantpb.ExecResponse)}
	case *implantpb.UploadRequest:
		spite.Name = MsgUpload.String()
		spite.Body = &implantpb.Spite_UploadRequest{UploadRequest: msg.(*implantpb.UploadRequest)}
	case *implantpb.DownloadRequest:
		spite.Name = MsgDownload.String()
		spite.Body = &implantpb.Spite_DownloadRequest{DownloadRequest: msg.(*implantpb.DownloadRequest)}
	case *implantpb.ExecuteBinary:
		spite.Name = msg.(*implantpb.ExecuteBinary).Type
		spite.Body = &implantpb.Spite_ExecuteBinary{ExecuteBinary: msg.(*implantpb.ExecuteBinary)}
	//case *implantpb.CurlRequest:
	//	spite.Name = MsgCurl.String()
	//	spite.Body = &implantpb.Spite_CurlRequest{CurlRequest: msg.(*implantpb.CurlRequest)}
	case *implantpb.BypassRequest:
		spite.Name = MsgBypass.String()
		spite.Body = &implantpb.Spite_BypassRequest{BypassRequest: msg.(*implantpb.BypassRequest)}
	case *implantpb.ExecuteAddon:
		spite.Name = MsgExecuteAddon.String()
		spite.Body = &implantpb.Spite_ExecuteAddon{ExecuteAddon: msg.(*implantpb.ExecuteAddon)}
	case *implantpb.LoadModule:
		spite.Name = MsgLoadModule.String()
		spite.Body = &implantpb.Spite_LoadModule{LoadModule: msg.(*implantpb.LoadModule)}
	case *implantpb.LoadAddon:
		spite.Name = MsgLoadAddon.String()
		spite.Body = &implantpb.Spite_LoadAddon{LoadAddon: msg.(*implantpb.LoadAddon)}
	case *implantpb.ACK:
		spite.Name = MsgAck.String()
		spite.Body = &implantpb.Spite_Ack{Ack: msg.(*implantpb.ACK)}
	default:
		return spite, ErrUnknownSpite
	}
	return spite, nil
}

func BuildSpites(spites []*implantpb.Spite) *implantpb.Spites {
	return &implantpb.Spites{Spites: spites}
}

func BuildOneSpites(spite *implantpb.Spite) *implantpb.Spites {
	return BuildSpites([]*implantpb.Spite{spite})
}

//func ParseSpite(spite *implantpb.Spite) (proto.Message, error) {
//	switch spite.Body.(type) {
//	case *implantpb.Spite_Register:
//		return spite.GetRegister(), nil
//	case *implantpb.Spite_ExecResponse:
//		return spite.GetExecResponse(), nil
//	case *implantpb.Spite_AssemblyResponse:
//		return spite.GetAssemblyResponse(), nil
//	default:
//		return nil, ErrUnknownSpite
//	}
//}

func BuildPipeline(msg proto.Message, tls proto.Message) *lispb.Pipeline {
	var pipeline = &lispb.Pipeline{}
	pipeline.Tls = tls.(*lispb.TLS)
	switch msg.(type) {
	case *lispb.TCPPipeline:
		pipeline.Body = &lispb.Pipeline_Tcp{Tcp: msg.(*lispb.TCPPipeline)}
	default:
		logs.Log.Debug(ErrUnknownJob.Error())
		return pipeline
	}
	return pipeline
}
