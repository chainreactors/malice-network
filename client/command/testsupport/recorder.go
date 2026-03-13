package testsupport

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

type RecordedCall struct {
	Method   string
	Request  any
	Metadata metadata.MD
}

type RecorderRPC struct {
	clientrpc.MaliceRPCClient
	listenerrpc.ListenerRPCClient

	mu            sync.Mutex
	calls         []RecordedCall
	sessionEvents []RecordedCall
	taskID        atomic.Uint32

	taskResponders        map[string]func(context.Context, any) (*clientpb.Task, error)
	emptyResponders       map[string]func(context.Context, any) (*clientpb.Empty, error)
	taskContextResponders map[string]func(context.Context, any) (*clientpb.TaskContext, error)
	sessionResponders     map[string]func(context.Context, any) (*clientpb.Session, error)
	sessions              map[string]*clientpb.Session
}

func NewRecorderRPC() *RecorderRPC {
	r := &RecorderRPC{
		taskResponders:        map[string]func(context.Context, any) (*clientpb.Task, error){},
		emptyResponders:       map[string]func(context.Context, any) (*clientpb.Empty, error){},
		taskContextResponders: map[string]func(context.Context, any) (*clientpb.TaskContext, error){},
		sessionResponders:     map[string]func(context.Context, any) (*clientpb.Session, error){},
		sessions:              map[string]*clientpb.Session{},
	}
	r.taskID.Store(100)
	return r
}

func (r *RecorderRPC) Calls() []RecordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]RecordedCall, len(r.calls))
	copy(out, r.calls)
	return out
}

func (r *RecorderRPC) SessionEvents() []RecordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]RecordedCall, len(r.sessionEvents))
	copy(out, r.sessionEvents)
	return out
}

func (r *RecorderRPC) SetSession(session *clientpb.Session) {
	if session == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.GetSessionId()] = cloneRequest(session).(*clientpb.Session)
}

func (r *RecorderRPC) OnTask(method string, fn func(context.Context, any) (*clientpb.Task, error)) {
	r.taskResponders[method] = fn
}

func (r *RecorderRPC) OnEmpty(method string, fn func(context.Context, any) (*clientpb.Empty, error)) {
	r.emptyResponders[method] = fn
}

func (r *RecorderRPC) OnTaskContext(method string, fn func(context.Context, any) (*clientpb.TaskContext, error)) {
	r.taskContextResponders[method] = fn
}

func (r *RecorderRPC) OnSession(method string, fn func(context.Context, any) (*clientpb.Session, error)) {
	r.sessionResponders[method] = fn
}

func (r *RecorderRPC) Sleep(ctx context.Context, in *implantpb.Timer, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Sleep", in)
}

func (r *RecorderRPC) Keepalive(ctx context.Context, in *implantpb.CommonBody, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Keepalive", in)
}

func (r *RecorderRPC) Suicide(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Suicide", in)
}

func (r *RecorderRPC) Ping(ctx context.Context, in *implantpb.Ping, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Ping", in)
}

func (r *RecorderRPC) WaitTaskFinish(ctx context.Context, in *clientpb.Task, opts ...grpc.CallOption) (*clientpb.TaskContext, error) {
	r.recordPrimary(ctx, "WaitTaskFinish", in)
	if responder, ok := r.taskContextResponders["WaitTaskFinish"]; ok {
		return responder(ctx, in)
	}
	if in == nil {
		return nil, fmt.Errorf("wait task request is nil")
	}
	return &clientpb.TaskContext{
		Task: cloneRequest(in).(*clientpb.Task),
		Spite: &implantpb.Spite{
			Body: &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
		},
	}, nil
}

func (r *RecorderRPC) Polling(ctx context.Context, in *clientpb.Polling, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "Polling", in)
}

func (r *RecorderRPC) GetSession(ctx context.Context, in *clientpb.SessionRequest, opts ...grpc.CallOption) (*clientpb.Session, error) {
	r.recordPrimary(ctx, "GetSession", in)
	if responder, ok := r.sessionResponders["GetSession"]; ok {
		return responder(ctx, in)
	}
	if in == nil {
		return nil, fmt.Errorf("session request is nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessions[in.GetSessionId()]
	if !ok {
		return nil, fmt.Errorf("session %s not found", in.GetSessionId())
	}
	return cloneRequest(session).(*clientpb.Session), nil
}

func (r *RecorderRPC) SessionEvent(ctx context.Context, in *clientpb.Event, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.recordSessionEvent(ctx, "SessionEvent", in)
	return &clientpb.Empty{}, nil
}

func (r *RecorderRPC) ServiceList(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ServiceList", in)
}

func (r *RecorderRPC) ServiceCreate(ctx context.Context, in *implantpb.ServiceRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ServiceCreate", in)
}

func (r *RecorderRPC) ServiceStart(ctx context.Context, in *implantpb.ServiceRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ServiceStart", in)
}

func (r *RecorderRPC) ServiceStop(ctx context.Context, in *implantpb.ServiceRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ServiceStop", in)
}

func (r *RecorderRPC) ServiceQuery(ctx context.Context, in *implantpb.ServiceRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ServiceQuery", in)
}

func (r *RecorderRPC) ServiceDelete(ctx context.Context, in *implantpb.ServiceRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ServiceDelete", in)
}

func (r *RecorderRPC) RegQuery(ctx context.Context, in *implantpb.RegistryRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "RegQuery", in)
}

func (r *RecorderRPC) RegAdd(ctx context.Context, in *implantpb.RegistryWriteRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "RegAdd", in)
}

func (r *RecorderRPC) RegDelete(ctx context.Context, in *implantpb.RegistryRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "RegDelete", in)
}

func (r *RecorderRPC) RegListKey(ctx context.Context, in *implantpb.RegistryRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "RegListKey", in)
}

func (r *RecorderRPC) RegListValue(ctx context.Context, in *implantpb.RegistryRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "RegListValue", in)
}

func (r *RecorderRPC) TaskSchdList(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "TaskSchdList", in)
}

func (r *RecorderRPC) TaskSchdCreate(ctx context.Context, in *implantpb.TaskScheduleRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "TaskSchdCreate", in)
}

func (r *RecorderRPC) TaskSchdStart(ctx context.Context, in *implantpb.TaskScheduleRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "TaskSchdStart", in)
}

func (r *RecorderRPC) TaskSchdStop(ctx context.Context, in *implantpb.TaskScheduleRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "TaskSchdStop", in)
}

func (r *RecorderRPC) TaskSchdDelete(ctx context.Context, in *implantpb.TaskScheduleRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "TaskSchdDelete", in)
}

func (r *RecorderRPC) TaskSchdQuery(ctx context.Context, in *implantpb.TaskScheduleRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "TaskSchdQuery", in)
}

func (r *RecorderRPC) TaskSchdRun(ctx context.Context, in *implantpb.TaskScheduleRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "TaskSchdRun", in)
}

func (r *RecorderRPC) Whoami(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Whoami", in)
}

func (r *RecorderRPC) Kill(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Kill", in)
}

func (r *RecorderRPC) Ps(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Ps", in)
}

func (r *RecorderRPC) Env(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Env", in)
}

func (r *RecorderRPC) SetEnv(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "SetEnv", in)
}

func (r *RecorderRPC) UnsetEnv(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "UnsetEnv", in)
}

func (r *RecorderRPC) Netstat(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Netstat", in)
}

func (r *RecorderRPC) Info(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Info", in)
}

func (r *RecorderRPC) Bypass(ctx context.Context, in *implantpb.BypassRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Bypass", in)
}

func (r *RecorderRPC) WmiQuery(ctx context.Context, in *implantpb.WmiQueryRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "WmiQuery", in)
}

func (r *RecorderRPC) WmiExecute(ctx context.Context, in *implantpb.WmiMethodRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "WmiExecute", in)
}

func (r *RecorderRPC) InitBindSession(ctx context.Context, in *implantpb.Init, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "InitBindSession", in)
}

func (r *RecorderRPC) taskResponse(ctx context.Context, method string, request any) (*clientpb.Task, error) {
	r.recordPrimary(ctx, method, request)
	if responder, ok := r.taskResponders[method]; ok {
		return responder(ctx, request)
	}
	return r.defaultTask(ctx, method), nil
}

func (r *RecorderRPC) emptyResponse(ctx context.Context, method string, request any) (*clientpb.Empty, error) {
	r.recordPrimary(ctx, method, request)
	if responder, ok := r.emptyResponders[method]; ok {
		return responder(ctx, request)
	}
	return &clientpb.Empty{}, nil
}

func (r *RecorderRPC) defaultTask(ctx context.Context, method string) *clientpb.Task {
	id := r.taskID.Add(1)
	md, _ := metadata.FromOutgoingContext(ctx)
	return &clientpb.Task{
		TaskId:    id,
		SessionId: first(md.Get("session_id")),
		Type:      methodTaskTypes[method],
		Cur:       1,
		Total:     1,
	}
}

func (r *RecorderRPC) recordPrimary(ctx context.Context, method string, request any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, RecordedCall{
		Method:   method,
		Request:  cloneRequest(request),
		Metadata: cloneMetadata(ctx),
	})
}

func (r *RecorderRPC) recordSessionEvent(ctx context.Context, method string, request any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionEvents = append(r.sessionEvents, RecordedCall{
		Method:   method,
		Request:  cloneRequest(request),
		Metadata: cloneMetadata(ctx),
	})
}

func cloneRequest(request any) any {
	message, ok := request.(proto.Message)
	if !ok || message == nil {
		return request
	}
	return proto.Clone(message)
}

func cloneMetadata(ctx context.Context) metadata.MD {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok || md == nil {
		return metadata.MD{}
	}
	return md.Copy()
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

var methodTaskTypes = map[string]string{
	"Sleep":         consts.ModuleSleep,
	"Keepalive":     consts.ModuleKeepalive,
	"Suicide":       consts.ModuleSuicide,
	"Ping":          consts.ModulePing,
	"ServiceList":   consts.ModuleServiceList,
	"ServiceCreate": consts.ModuleServiceCreate,
	"ServiceStart":  consts.ModuleServiceStart,
	"ServiceStop":   consts.ModuleServiceStop,
	"ServiceQuery":  consts.ModuleServiceQuery,
	"ServiceDelete": consts.ModuleServiceDelete,
	"RegQuery":      consts.ModuleRegQuery,
	"RegAdd":        consts.ModuleRegAdd,
	"RegDelete":     consts.ModuleRegDelete,
	"RegListKey":    consts.ModuleRegListKey,
	"RegListValue":  consts.ModuleRegListValue,
	"TaskSchdList":  consts.ModuleTaskSchdList,
	"TaskSchdCreate": consts.ModuleTaskSchdCreate,
	"TaskSchdStart":  consts.ModuleTaskSchdStart,
	"TaskSchdStop":   consts.ModuleTaskSchdStop,
	"TaskSchdDelete": consts.ModuleTaskSchdDelete,
	"TaskSchdQuery":  consts.ModuleTaskSchdQuery,
	"TaskSchdRun":    consts.ModuleTaskSchdRun,
	"Whoami":         consts.ModuleWhoami,
	"Kill":           consts.ModuleKill,
	"Ps":             consts.ModulePs,
	"Env":            consts.ModuleEnv,
	"SetEnv":         consts.ModuleSetEnv,
	"UnsetEnv":       consts.ModuleUnsetEnv,
	"Netstat":        consts.ModuleNetstat,
	"Info":           consts.ModuleSysInfo,
	"Bypass":         consts.ModuleBypass,
	"WmiQuery":       consts.ModuleWmiQuery,
	"WmiExecute":     consts.ModuleWmiExec,
}
