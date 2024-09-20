package types

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
)

type MsgName string

const (
	MsgUnknown          MsgName = "unknown"
	MsgNil              MsgName = "nil"
	MsgEmpty            MsgName = "empty"
	MsgRequest          MsgName = "request"
	MsgResponse         MsgName = "response"
	MsgBlock            MsgName = "block"
	MsgRegister         MsgName = "register"
	MsgUpload           MsgName = consts.ModuleUpload
	MsgDownload         MsgName = consts.ModuleDownload
	MsgCurl             MsgName = consts.ModuleCurl
	MsgExec             MsgName = consts.ModuleExecution
	MsgAck              MsgName = "ack"
	MsgListModule       MsgName = consts.ModuleListModule
	MsgLoadModule       MsgName = consts.ModuleLoadModule
	MsgListAddon        MsgName = consts.ModuleListAddon
	MsgLoadAddon        MsgName = consts.ModuleLoadAddon
	MsgAssemblyResponse MsgName = "assembly_response"
	MsgExecuteAddon     MsgName = consts.ModuleExecuteAddon
	MsgExecuteLocal     MsgName = consts.ModuleExecuteLocal
	//MsgExecuteSpawn     MsgName = "execute_spawn"
	MsgLs      MsgName = consts.ModuleLs
	MsgNetstat MsgName = consts.ModuleNetstat
	MsgPs      MsgName = consts.ModulePs
	MsgKill    MsgName = consts.ModuleKill
	MsgBypass  MsgName = consts.ModuleBypass
	MsgSysInfo MsgName = "sysinfo"
)

func (r MsgName) String() string {
	return string(r)
}

func MessageType(message *implantpb.Spite) MsgName {
	switch message.Body.(type) {
	case nil:
		return MsgNil
	case *implantpb.Spite_Request:
		return MsgName(message.Name)
	case *implantpb.Spite_ExecuteBinary:
		return MsgName(message.GetExecuteBinary().Type)
	case *implantpb.Spite_Response:
		return MsgResponse
	case *implantpb.Spite_Register:
		return MsgRegister
	case *implantpb.Spite_Empty:
		return MsgEmpty
	case *implantpb.Spite_Sysinfo:
		return MsgSysInfo
	case *implantpb.Spite_ExecRequest, *implantpb.Spite_ExecResponse:
		return MsgExec
	case *implantpb.Spite_UploadRequest:
		return MsgUpload
	case *implantpb.Spite_DownloadRequest, *implantpb.Spite_DownloadResponse:
		return MsgDownload
	case *implantpb.Spite_Ack:
		return MsgAck
	case *implantpb.Spite_Block:
		return MsgBlock
	case *implantpb.Spite_AssemblyResponse:
		return MsgAssemblyResponse
	case *implantpb.Spite_LoadAddon:
		return MsgLoadAddon
	case *implantpb.Spite_LoadModule:
		return MsgLoadModule
	case *implantpb.Spite_Modules:
		return MsgListModule
	case *implantpb.Spite_PsResponse:
		return MsgPs
	case *implantpb.Spite_LsResponse:
		return MsgLs
	case *implantpb.Spite_Addons:
		return MsgListAddon
	case *implantpb.Spite_BypassRequest:
		return MsgBypass
	case *implantpb.Spite_NetstatResponse:
		return MsgNetstat
	default:
		return MsgUnknown
	}
}
