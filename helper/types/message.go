package types

import "github.com/chainreactors/malice-network/proto/implant/implantpb"

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

func MessageType(message *implantpb.Spite) MsgName {
	switch message.Body.(type) {
	case nil:
		return MsgNil
	case *implantpb.Spite_Request:
		return MsgRequest
	case *implantpb.Spite_Register:
		return MsgRegister
	case *implantpb.Spite_ExecRequest, *implantpb.Spite_ExecResponse:
		return MsgExec
	case *implantpb.Spite_UploadRequest:
		return MsgUpload
	case *implantpb.Spite_DownloadRequest:
		return MsgDownload
	case *implantpb.Spite_AsyncAck:
		return MsgAck
	case *implantpb.Spite_Block:
		return MsgBlock
	case *implantpb.Spite_AssemblyResponse:
		return MsgAssemblyResponse
	case *implantpb.Spite_ExecuteAssembly:
		return MsgExecuteAssembly
	case *implantpb.Spite_ExecuteShellcode:
		return MsgExecuteShellcode
	case *implantpb.Spite_ExecuteSpawn:
		return MsgExecuteSpawn
	case *implantpb.Spite_ExecuteSideload:
		return MsgExecuteSideLoad
	case *implantpb.Spite_ExecuteBof:
		return MsgExecuteBof
	case *implantpb.Spite_Extensions:
		return MsgExtensions
	case *implantpb.Spite_LoadExtension:
		return MsgLoadExtension
	case *implantpb.Spite_ExecuteExtension:
		return MsgExecuteExtension
	case *implantpb.Spite_LoadModule:
		return MsgLoadModule
	case *implantpb.Spite_Modules:
		return MsgModules
	default:
		return MsgUnknown
	}
}
