package types

import (
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
)

type MsgName string

const (
	MsgUnknown          MsgName = "unknown"
	MsgNil              MsgName = "nil"
	MsgRequest          MsgName = "request"
	MsgBlock            MsgName = "block"
	MsgRegister         MsgName = "register"
	MsgUpload           MsgName = "upload"
	MsgDownload         MsgName = "download"
	MsgExec             MsgName = "exec"
	MsgAck              MsgName = "ack"
	MsgModules          MsgName = "modules"
	MsgLoadModule       MsgName = "load_module"
	MsgExtensions       MsgName = "extensions"
	MsgLoadExtension    MsgName = "load_extension"
	MsgAssemblyResponse MsgName = "assembly_response"
	MsgExecuteExtension MsgName = "execute_extension"
	MsgExecuteAssembly  MsgName = "execute_assembly"
	MsgExecuteShellcode MsgName = "execute_shellcode"
	MsgExecuteSpawn     MsgName = "execute_spawn"
	MsgExecuteSideLoad  MsgName = "execute_sideload"
	MsgExecuteBof       MsgName = "execute_bof"
)

func (r MsgName) String() string {
	return string(r)
}

func MessageType(message *commonpb.Spite) MsgName {
	switch message.Body.(type) {
	case nil:
		return MsgNil
	case *commonpb.Spite_Request:
		return MsgRequest
	case *commonpb.Spite_Register:
		return MsgRegister
	case *commonpb.Spite_ExecRequest, *commonpb.Spite_ExecResponse:
		return MsgExec
	case *commonpb.Spite_UploadRequest:
		return MsgUpload
	case *commonpb.Spite_DownloadRequest:
		return MsgDownload
	case *commonpb.Spite_AsyncAck:
		return MsgAck
	case *commonpb.Spite_Block:
		return MsgBlock
	case *commonpb.Spite_AssemblyResponse:
		return MsgAssemblyResponse
	case *commonpb.Spite_ExecuteAssembly:
		return MsgExecuteAssembly
	case *commonpb.Spite_ExecuteShellcode:
		return MsgExecuteShellcode
	case *commonpb.Spite_ExecuteSpawn:
		return MsgExecuteSpawn
	case *commonpb.Spite_ExecuteSideload:
		return MsgExecuteSideLoad
	case *commonpb.Spite_ExecuteBof:
		return MsgExecuteBof
	case *commonpb.Spite_Extensions:
		return MsgExtensions
	case *commonpb.Spite_LoadExtension:
		return MsgLoadExtension
	case *commonpb.Spite_ExecuteExtension:
		return MsgExecuteExtension
	case *commonpb.Spite_LoadModule:
		return MsgLoadModule
	case *commonpb.Spite_Modules:
		return MsgModules
	default:
		return MsgUnknown
	}
}
