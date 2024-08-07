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
	MsgModules          MsgName = "list_module"
	MsgLoadModule       MsgName = "load_module"
	MsgExtensions       MsgName = "extensions"
	MsgLoadExtension    MsgName = "load_extension"
	MsgAssemblyResponse MsgName = "assembly_response"
	MsgExecuteExtension MsgName = "execute_extension"
	MsgExecuteAssembly  MsgName = consts.ModuleExecuteAssembly
	MsgExecuteShellcode MsgName = consts.ModuleExecuteShellcode
	MsgExecutePE        MsgName = consts.ModuleExecutePE
	//MsgExecuteSpawn     MsgName = "execute_spawn"
	MsgExecuteBof MsgName = consts.ModuleExecuteBof
	MsgPowershell MsgName = consts.ModulePowershell
	MsgPwd        MsgName = consts.ModulePwd
	MsgLs         MsgName = consts.ModuleLs
	MsgNetstat    MsgName = consts.ModuleNetstat
	MsgPs         MsgName = consts.ModulePs
	MsgCp         MsgName = consts.ModuleCp
	MsgMv         MsgName = consts.ModuleMv
	MsgMkdir      MsgName = consts.ModuleMkdir
	MsgRm         MsgName = consts.ModuleRm
	MsgCat        MsgName = consts.ModuleCat
	MsgCd         MsgName = consts.ModuleCd
	MsgChmod      MsgName = consts.ModuleChmod
	MsgChown      MsgName = consts.ModuleChown
	MsgKill       MsgName = consts.ModuleKill
	MsgEnv        MsgName = consts.ModuleEnv
	MsgSetEnv     MsgName = consts.ModuleSetEnv
	MsgUnsetEnv   MsgName = consts.ModuleUnsetEnv
	MsgWhoami     MsgName = consts.ModuleWhoami
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
	case *implantpb.Spite_Response:
		return MsgResponse
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
	//case *implantpb.Spite_ExecuteSpawn:
	//	return MsgExecuteSpawn
	//case *implantpb.Spite_ExecuteSideload:
	//	return MsgExecuteSideLoad
	case *implantpb.Spite_ExecutePe:
		return MsgExecutePE
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
	case *implantpb.Spite_PsResponse:
		return MsgPs
	case *implantpb.Spite_LsResponse:
		return MsgLs
	case *implantpb.Spite_Empty:
		return MsgEmpty
	default:
		return MsgUnknown
	}
}
