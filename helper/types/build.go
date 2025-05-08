package types

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
)

var (
	ErrUnknownSpite = errors.New("unknown spite body")
	ErrUnknownJob   = errors.New("unknown job body")
)

func BuildPingSpite() *implantpb.Spite {
	return &implantpb.Spite{
		Name: consts.ModulePing,
		Body: &implantpb.Spite_Ping{},
	}
}

func BuildPingSpites() *implantpb.Spites {
	return BuildOneSpites(BuildPingSpite())
}

// BuildSpite build spite request
func BuildSpite(spite *implantpb.Spite, msg proto.Message) (*implantpb.Spite, error) {
	switch msg := msg.(type) {
	case *implantpb.Request:
		spite.Name = msg.Name
		spite.Body = &implantpb.Spite_Request{Request: msg}
	case *implantpb.TaskCtrl:
		spite.Name = msg.Op
		spite.Body = &implantpb.Spite_Task{Task: msg}
	case *implantpb.Init:
		spite.Name = MsgInit.String()
		spite.Body = &implantpb.Spite_Init{Init: msg}
	case *implantpb.Ping:
		spite.Name = MsgPing.String()
		spite.Body = &implantpb.Spite_Ping{Ping: msg}
	case *implantpb.Timer:
		spite.Name = MsgSleep.String()
		spite.Body = &implantpb.Spite_SleepRequest{SleepRequest: msg}
	case *implantpb.ACK:
		spite.Name = MsgAck.String()
		spite.Body = &implantpb.Spite_Ack{Ack: msg}
	case *implantpb.Block:
		spite.Name = MsgBlock.String()
		spite.Body = &implantpb.Spite_Block{Block: msg}
	case *implantpb.Register:
		spite.Name = MsgRegister.String()
		spite.Body = &implantpb.Spite_Register{Register: msg}
	case *implantpb.ExecRequest:
		spite.Name = MsgExec.String()
		spite.Body = &implantpb.Spite_ExecRequest{ExecRequest: msg}
	case *implantpb.ExecResponse:
		spite.Name = MsgExec.String()
		spite.Body = &implantpb.Spite_ExecResponse{ExecResponse: msg}
	case *implantpb.UploadRequest:
		spite.Name = MsgUpload.String()
		spite.Body = &implantpb.Spite_UploadRequest{UploadRequest: msg}
	case *implantpb.DownloadRequest:
		spite.Name = MsgDownload.String()
		spite.Body = &implantpb.Spite_DownloadRequest{DownloadRequest: msg}
	case *implantpb.ExecuteBinary:
		spite.Name = msg.Type
		spite.Body = &implantpb.Spite_ExecuteBinary{ExecuteBinary: msg}
	case *implantpb.CurlRequest:
		spite.Name = MsgCurl.String()
		spite.Body = &implantpb.Spite_CurlRequest{CurlRequest: msg}
	case *implantpb.BypassRequest:
		spite.Name = MsgBypass.String()
		spite.Body = &implantpb.Spite_BypassRequest{BypassRequest: msg}
	case *implantpb.ExecuteAddon:
		spite.Name = MsgExecuteAddon.String()
		spite.Body = &implantpb.Spite_ExecuteAddon{ExecuteAddon: msg}
	case *implantpb.LoadModule:
		spite.Name = MsgLoadModule.String()
		spite.Body = &implantpb.Spite_LoadModule{LoadModule: msg}
	case *implantpb.LoadAddon:
		spite.Name = MsgLoadAddon.String()
		spite.Body = &implantpb.Spite_LoadAddon{LoadAddon: msg}
	case *implantpb.RegistryRequest:
		spite.Name = msg.Type
		spite.Body = &implantpb.Spite_RegistryRequest{RegistryRequest: msg.Registry}
	case *implantpb.RegistryWriteRequest:
		spite.Name = MsgRegistryAdd.String()
		spite.Body = &implantpb.Spite_RegistryWriteRequest{RegistryWriteRequest: msg}
	case *implantpb.ServiceRequest:
		spite.Name = msg.Type
		spite.Body = &implantpb.Spite_ServiceRequest{ServiceRequest: msg.Service}
	case *implantpb.TaskScheduleRequest:
		spite.Name = msg.Type
		spite.Body = &implantpb.Spite_ScheduleRequest{ScheduleRequest: msg.Taskschd}
	case *implantpb.WmiQueryRequest:
		spite.Name = MsgWmiQuery.String()
		spite.Body = &implantpb.Spite_WmiRequest{WmiRequest: msg}
	case *implantpb.WmiMethodRequest:
		spite.Name = MsgWmiExecute.String()
		spite.Body = &implantpb.Spite_WmiMethodRequest{WmiMethodRequest: msg}
	case *implantpb.PipeRequest:
		spite.Name = msg.Type
		spite.Body = &implantpb.Spite_PipeRequest{PipeRequest: msg.Pipe}
	case *implantpb.Login:
		spite.Name = MsgLogin.String()
		spite.Body = &implantpb.Spite_LoginRequest{LoginRequest: msg}
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
