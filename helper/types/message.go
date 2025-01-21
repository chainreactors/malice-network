package types

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
)

type MsgName string

const (
	// internal

	MsgInit  MsgName = consts.ModuleInit
	MsgPing  MsgName = consts.ModulePing
	MsgSleep MsgName = consts.ModuleSleep
	MsgLogin MsgName = consts.ModuleLogin
	MsgTask  MsgName = "task"
	MsgTasks MsgName = "tasks"
	// basic message
	MsgResponse MsgName = "response"
	MsgBlock    MsgName = "block"
	MsgAck      MsgName = "ack"
	MsgRegister MsgName = "register"
	MsgRequest  MsgName = "request"
	MsgUnknown  MsgName = "unknown"
	MsgNil      MsgName = "nil"
	MsgEmpty    MsgName = "empty"

	MsgSysInfo        MsgName = consts.ModuleSysInfo
	MsgUpload         MsgName = consts.ModuleUpload
	MsgDownload       MsgName = consts.ModuleDownload
	MsgCurl           MsgName = consts.ModuleCurl
	MsgExec           MsgName = consts.ModuleExecution
	MsgListModule     MsgName = consts.ModuleListModule
	MsgLoadModule     MsgName = consts.ModuleLoadModule
	MsgListAddon      MsgName = consts.ModuleListAddon
	MsgLoadAddon      MsgName = consts.ModuleLoadAddon
	MsgBinaryResponse MsgName = "assembly_response"
	MsgExecuteAddon   MsgName = consts.ModuleExecuteAddon
	MsgExecuteLocal   MsgName = consts.ModuleExecuteLocal
	//MsgExecuteSpawn     MsgName = "execute_spawn"
	MsgLs          MsgName = consts.ModuleLs
	MsgNetstat     MsgName = consts.ModuleNetstat
	MsgPs          MsgName = consts.ModulePs
	MsgKill        MsgName = consts.ModuleKill
	MsgBypass      MsgName = consts.ModuleBypass
	MsgRegistryAdd MsgName = consts.ModuleRegAdd

	MsgServicesResponse  MsgName = consts.ModuleServiceList
	MsgServiceResponse   MsgName = consts.ModuleServiceQuery
	MsgTaskSchdsResponse MsgName = consts.ModuleTaskSchdList
	MsgTaskSchdResponse  MsgName = consts.ModuleTaskSchdQuery
	MsgWmiQuery          MsgName = consts.ModuleWmiQuery
	MsgWmiExecute        MsgName = consts.ModuleWmiExec
)

func (r MsgName) String() string {
	return string(r)
}

// MessageType , parse response message
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
	case *implantpb.Spite_Ping:
		return MsgPing
	case *implantpb.Spite_Init:
		return MsgInit
	case *implantpb.Spite_TaskInfo:
		return MsgTask
	case *implantpb.Spite_TaskList:
		return MsgTasks
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
	case *implantpb.Spite_BinaryResponse:
		return MsgBinaryResponse
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
	case *implantpb.Spite_ServiceResponse:
		return MsgServiceResponse
	case *implantpb.Spite_ServicesResponse:
		return MsgServicesResponse
	case *implantpb.Spite_ScheduleResponse:
		return MsgTaskSchdResponse
	case *implantpb.Spite_SchedulesResponse:
		return MsgTaskSchdsResponse

	default:
		return MsgUnknown
	}
}
