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

	taskResponders         map[string]func(context.Context, any) (*clientpb.Task, error)
	emptyResponders        map[string]func(context.Context, any) (*clientpb.Empty, error)
	artifactResponders     map[string]func(context.Context, any) (*clientpb.Artifact, error)
	buildConfigResponders  map[string]func(context.Context, any) (*clientpb.BuildConfig, error)
	contextResponders      map[string]func(context.Context, any) (*clientpb.Context, error)
	taskContextResponders  map[string]func(context.Context, any) (*clientpb.TaskContext, error)
	taskContextsResponders map[string]func(context.Context, any) (*clientpb.TaskContexts, error)
	tasksResponders        map[string]func(context.Context, any) (*clientpb.Tasks, error)
	sessionResponders      map[string]func(context.Context, any) (*clientpb.Session, error)
	basicResponders        map[string]func(context.Context, any) (*clientpb.Basic, error)
	listenersResponders    map[string]func(context.Context, any) (*clientpb.Listeners, error)
	pipelinesResponders    map[string]func(context.Context, any) (*clientpb.Pipelines, error)
	licenseResponders      map[string]func(context.Context, any) (*clientpb.LicenseInfo, error)
	contextsResponders     map[string]func(context.Context, any) (*clientpb.Contexts, error)
	certsResponders        map[string]func(context.Context, any) (*clientpb.Certs, error)
	tlsResponders          map[string]func(context.Context, any) (*clientpb.TLS, error)
	acmeConfigResponders   map[string]func(context.Context, any) (*clientpb.AcmeConfig, error)
	sessions               map[string]*clientpb.Session
}

func NewRecorderRPC() *RecorderRPC {
	r := &RecorderRPC{
		taskResponders:         map[string]func(context.Context, any) (*clientpb.Task, error){},
		emptyResponders:        map[string]func(context.Context, any) (*clientpb.Empty, error){},
		artifactResponders:     map[string]func(context.Context, any) (*clientpb.Artifact, error){},
		buildConfigResponders:  map[string]func(context.Context, any) (*clientpb.BuildConfig, error){},
		contextResponders:      map[string]func(context.Context, any) (*clientpb.Context, error){},
		taskContextResponders:  map[string]func(context.Context, any) (*clientpb.TaskContext, error){},
		taskContextsResponders: map[string]func(context.Context, any) (*clientpb.TaskContexts, error){},
		tasksResponders:        map[string]func(context.Context, any) (*clientpb.Tasks, error){},
		sessionResponders:      map[string]func(context.Context, any) (*clientpb.Session, error){},
		basicResponders:        map[string]func(context.Context, any) (*clientpb.Basic, error){},
		listenersResponders:    map[string]func(context.Context, any) (*clientpb.Listeners, error){},
		pipelinesResponders:    map[string]func(context.Context, any) (*clientpb.Pipelines, error){},
		licenseResponders:      map[string]func(context.Context, any) (*clientpb.LicenseInfo, error){},
		contextsResponders:     map[string]func(context.Context, any) (*clientpb.Contexts, error){},
		certsResponders:        map[string]func(context.Context, any) (*clientpb.Certs, error){},
		tlsResponders:          map[string]func(context.Context, any) (*clientpb.TLS, error){},
		acmeConfigResponders:   map[string]func(context.Context, any) (*clientpb.AcmeConfig, error){},
		sessions:               map[string]*clientpb.Session{},
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

func (r *RecorderRPC) OnArtifact(method string, fn func(context.Context, any) (*clientpb.Artifact, error)) {
	r.artifactResponders[method] = fn
}

func (r *RecorderRPC) OnBuildConfig(method string, fn func(context.Context, any) (*clientpb.BuildConfig, error)) {
	r.buildConfigResponders[method] = fn
}

func (r *RecorderRPC) OnContext(method string, fn func(context.Context, any) (*clientpb.Context, error)) {
	r.contextResponders[method] = fn
}

func (r *RecorderRPC) OnTaskContext(method string, fn func(context.Context, any) (*clientpb.TaskContext, error)) {
	r.taskContextResponders[method] = fn
}

func (r *RecorderRPC) OnTaskContexts(method string, fn func(context.Context, any) (*clientpb.TaskContexts, error)) {
	r.taskContextsResponders[method] = fn
}

func (r *RecorderRPC) OnTasks(method string, fn func(context.Context, any) (*clientpb.Tasks, error)) {
	r.tasksResponders[method] = fn
}

func (r *RecorderRPC) OnSession(method string, fn func(context.Context, any) (*clientpb.Session, error)) {
	r.sessionResponders[method] = fn
}

func (r *RecorderRPC) OnBasic(method string, fn func(context.Context, any) (*clientpb.Basic, error)) {
	r.basicResponders[method] = fn
}

func (r *RecorderRPC) OnListeners(method string, fn func(context.Context, any) (*clientpb.Listeners, error)) {
	r.listenersResponders[method] = fn
}

func (r *RecorderRPC) OnPipelines(method string, fn func(context.Context, any) (*clientpb.Pipelines, error)) {
	r.pipelinesResponders[method] = fn
}

func (r *RecorderRPC) OnLicenseInfo(method string, fn func(context.Context, any) (*clientpb.LicenseInfo, error)) {
	r.licenseResponders[method] = fn
}

func (r *RecorderRPC) OnContexts(method string, fn func(context.Context, any) (*clientpb.Contexts, error)) {
	r.contextsResponders[method] = fn
}

func (r *RecorderRPC) OnCerts(method string, fn func(context.Context, any) (*clientpb.Certs, error)) {
	r.certsResponders[method] = fn
}

func (r *RecorderRPC) OnTLS(method string, fn func(context.Context, any) (*clientpb.TLS, error)) {
	r.tlsResponders[method] = fn
}

func (r *RecorderRPC) OnAcmeConfig(method string, fn func(context.Context, any) (*clientpb.AcmeConfig, error)) {
	r.acmeConfigResponders[method] = fn
}

func (r *RecorderRPC) GetBasic(ctx context.Context, in *clientpb.Empty, opts ...grpc.CallOption) (*clientpb.Basic, error) {
	r.recordPrimary(ctx, "GetBasic", in)
	if responder, ok := r.basicResponders["GetBasic"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.Basic{Version: "test-version", Os: "windows", Arch: "amd64"}, nil
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

func (r *RecorderRPC) ListModule(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ListModule", in)
}

func (r *RecorderRPC) LoadModule(ctx context.Context, in *implantpb.LoadModule, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "LoadModule", in)
}

func (r *RecorderRPC) RefreshModule(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "RefreshModule", in)
}

func (r *RecorderRPC) ExecuteModule(ctx context.Context, in *implantpb.ExecuteModuleRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ExecuteModule", in)
}

func (r *RecorderRPC) ListAddon(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ListAddon", in)
}

func (r *RecorderRPC) LoadAddon(ctx context.Context, in *implantpb.LoadAddon, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "LoadAddon", in)
}

func (r *RecorderRPC) ExecuteAddon(ctx context.Context, in *implantpb.ExecuteAddon, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ExecuteAddon", in)
}

func (r *RecorderRPC) ListTasks(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "ListTasks", in)
}

func (r *RecorderRPC) QueryTask(ctx context.Context, in *implantpb.TaskCtrl, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "QueryTask", in)
}

func (r *RecorderRPC) CancelTask(ctx context.Context, in *implantpb.TaskCtrl, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "CancelTask", in)
}

func (r *RecorderRPC) Clear(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Clear", in)
}

func (r *RecorderRPC) Execute(ctx context.Context, in *implantpb.ExecRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Execute", in)
}

func (r *RecorderRPC) Upload(ctx context.Context, in *implantpb.UploadRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Upload", in)
}

func (r *RecorderRPC) Download(ctx context.Context, in *implantpb.DownloadRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Download", in)
}

func (r *RecorderRPC) DownloadDir(ctx context.Context, in *implantpb.DownloadRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "DownloadDir", in)
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

func (r *RecorderRPC) Switch(ctx context.Context, in *implantpb.Switch, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Switch", in)
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

func (r *RecorderRPC) GetTasks(ctx context.Context, in *clientpb.TaskRequest, opts ...grpc.CallOption) (*clientpb.Tasks, error) {
	r.recordPrimary(ctx, "GetTasks", in)
	if responder, ok := r.tasksResponders["GetTasks"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.Tasks{}, nil
}

func (r *RecorderRPC) GetListeners(ctx context.Context, in *clientpb.Empty, opts ...grpc.CallOption) (*clientpb.Listeners, error) {
	r.recordPrimary(ctx, "GetListeners", in)
	if responder, ok := r.listenersResponders["GetListeners"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.Listeners{}, nil
}

func (r *RecorderRPC) ListJobs(ctx context.Context, in *clientpb.Empty, opts ...grpc.CallOption) (*clientpb.Pipelines, error) {
	r.recordPrimary(ctx, "ListJobs", in)
	if responder, ok := r.pipelinesResponders["ListJobs"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.Pipelines{}, nil
}

func (r *RecorderRPC) GetLicenseInfo(ctx context.Context, in *clientpb.Empty, opts ...grpc.CallOption) (*clientpb.LicenseInfo, error) {
	r.recordPrimary(ctx, "GetLicenseInfo", in)
	if responder, ok := r.licenseResponders["GetLicenseInfo"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.LicenseInfo{Type: consts.LicenseCommunity}, nil
}

func (r *RecorderRPC) GetContexts(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Contexts, error) {
	r.recordPrimary(ctx, "GetContexts", in)
	if responder, ok := r.contextsResponders["GetContexts"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.Contexts{}, nil
}

func (r *RecorderRPC) Sync(ctx context.Context, in *clientpb.Sync, opts ...grpc.CallOption) (*clientpb.Context, error) {
	r.recordPrimary(ctx, "Sync", in)
	if responder, ok := r.contextResponders["Sync"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.Context{Id: in.GetContextId()}, nil
}

func (r *RecorderRPC) GetAllTaskContent(ctx context.Context, in *clientpb.Task, opts ...grpc.CallOption) (*clientpb.TaskContexts, error) {
	r.recordPrimary(ctx, "GetAllTaskContent", in)
	if responder, ok := r.taskContextsResponders["GetAllTaskContent"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.TaskContexts{}, nil
}

func (r *RecorderRPC) Broadcast(ctx context.Context, in *clientpb.Event, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "Broadcast", in)
}

func (r *RecorderRPC) Notify(ctx context.Context, in *clientpb.Event, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "Notify", in)
}

func (r *RecorderRPC) SessionEvent(ctx context.Context, in *clientpb.Event, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.recordSessionEvent(ctx, "SessionEvent", in)
	return &clientpb.Empty{}, nil
}

func (r *RecorderRPC) DeleteCertificate(ctx context.Context, in *clientpb.Cert, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "DeleteCertificate", in)
}

func (r *RecorderRPC) UpdateCertificate(ctx context.Context, in *clientpb.TLS, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "UpdateCertificate", in)
}

func (r *RecorderRPC) GetAllCertificates(ctx context.Context, in *clientpb.Empty, opts ...grpc.CallOption) (*clientpb.Certs, error) {
	r.recordPrimary(ctx, "GetAllCertificates", in)
	if responder, ok := r.certsResponders["GetAllCertificates"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.Certs{}, nil
}

func (r *RecorderRPC) DownloadCertificate(ctx context.Context, in *clientpb.Cert, opts ...grpc.CallOption) (*clientpb.TLS, error) {
	r.recordPrimary(ctx, "DownloadCertificate", in)
	if responder, ok := r.tlsResponders["DownloadCertificate"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.TLS{
		Cert: &clientpb.Cert{},
		Ca:   &clientpb.Cert{},
	}, nil
}

func (r *RecorderRPC) ObtainAcmeCert(ctx context.Context, in *clientpb.AcmeRequest, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "ObtainAcmeCert", in)
}

func (r *RecorderRPC) AddContext(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "AddContext", in)
}

func (r *RecorderRPC) AddScreenShot(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "AddScreenShot", in)
}

func (r *RecorderRPC) AddCredential(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "AddCredential", in)
}

func (r *RecorderRPC) AddKeylogger(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "AddKeylogger", in)
}

func (r *RecorderRPC) AddPort(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "AddPort", in)
}

func (r *RecorderRPC) AddUpload(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "AddUpload", in)
}

func (r *RecorderRPC) AddDownload(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "AddDownload", in)
}

func (r *RecorderRPC) DeleteContext(ctx context.Context, in *clientpb.Context, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "DeleteContext", in)
}

func (r *RecorderRPC) GetAcmeConfig(ctx context.Context, in *clientpb.Empty, opts ...grpc.CallOption) (*clientpb.AcmeConfig, error) {
	r.recordPrimary(ctx, "GetAcmeConfig", in)
	if responder, ok := r.acmeConfigResponders["GetAcmeConfig"]; ok {
		return responder(ctx, in)
	}
	return &clientpb.AcmeConfig{Credentials: map[string]string{}}, nil
}

func (r *RecorderRPC) UpdateAcmeConfig(ctx context.Context, in *clientpb.AcmeConfig, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "UpdateAcmeConfig", in)
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

func (r *RecorderRPC) Pwd(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Pwd", in)
}

func (r *RecorderRPC) Ls(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Ls", in)
}

func (r *RecorderRPC) Cd(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Cd", in)
}

func (r *RecorderRPC) Rm(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Rm", in)
}

func (r *RecorderRPC) Mv(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Mv", in)
}

func (r *RecorderRPC) Cp(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Cp", in)
}

func (r *RecorderRPC) Cat(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Cat", in)
}

func (r *RecorderRPC) Mkdir(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Mkdir", in)
}

func (r *RecorderRPC) Touch(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Touch", in)
}

func (r *RecorderRPC) EnumDrivers(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "EnumDrivers", in)
}

func (r *RecorderRPC) Whoami(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Whoami", in)
}

func (r *RecorderRPC) Runas(ctx context.Context, in *implantpb.RunAsRequest, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Runas", in)
}

func (r *RecorderRPC) Privs(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Privs", in)
}

func (r *RecorderRPC) Rev2Self(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "Rev2Self", in)
}

func (r *RecorderRPC) GetSystem(ctx context.Context, in *implantpb.Request, opts ...grpc.CallOption) (*clientpb.Task, error) {
	return r.taskResponse(ctx, "GetSystem", in)
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

func (r *RecorderRPC) GenerateSelfCert(ctx context.Context, in *clientpb.Pipeline, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return r.emptyResponse(ctx, "GenerateSelfCert", in)
}

func (r *RecorderRPC) Build(ctx context.Context, in *clientpb.BuildConfig, opts ...grpc.CallOption) (*clientpb.Artifact, error) {
	return r.artifactResponse(ctx, "Build", in)
}

func (r *RecorderRPC) SyncBuild(ctx context.Context, in *clientpb.BuildConfig, opts ...grpc.CallOption) (*clientpb.Artifact, error) {
	return r.artifactResponse(ctx, "SyncBuild", in)
}

func (r *RecorderRPC) CheckSource(ctx context.Context, in *clientpb.BuildConfig, opts ...grpc.CallOption) (*clientpb.BuildConfig, error) {
	r.recordPrimary(ctx, "CheckSource", in)
	if responder, ok := r.buildConfigResponders["CheckSource"]; ok {
		return responder(ctx, in)
	}
	if in == nil {
		return &clientpb.BuildConfig{Source: consts.ArtifactFromDocker}, nil
	}
	cfg := cloneRequest(in).(*clientpb.BuildConfig)
	if cfg.Source == "" {
		cfg.Source = consts.ArtifactFromDocker
	}
	return cfg, nil
}

func (r *RecorderRPC) DownloadArtifact(ctx context.Context, in *clientpb.Artifact, opts ...grpc.CallOption) (*clientpb.Artifact, error) {
	return r.artifactResponse(ctx, "DownloadArtifact", in)
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

func (r *RecorderRPC) artifactResponse(ctx context.Context, method string, request any) (*clientpb.Artifact, error) {
	r.recordPrimary(ctx, method, request)
	if responder, ok := r.artifactResponders[method]; ok {
		return responder(ctx, request)
	}
	if in, ok := request.(*clientpb.BuildConfig); ok && in != nil {
		return &clientpb.Artifact{
			Name:   fmt.Sprintf("%s-%s", in.BuildType, in.Target),
			Type:   in.BuildType,
			Target: in.Target,
			Source: in.Source,
			Bin:    []byte("artifact-bin"),
		}, nil
	}
	if in, ok := request.(*clientpb.Artifact); ok && in != nil {
		return &clientpb.Artifact{
			Name: in.Name,
			Bin:  []byte("artifact-bin"),
		}, nil
	}
	return &clientpb.Artifact{Name: method, Bin: []byte("artifact-bin")}, nil
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
	"Sleep":          consts.ModuleSleep,
	"Keepalive":      consts.ModuleKeepalive,
	"Suicide":        consts.ModuleSuicide,
	"Ping":           consts.ModulePing,
	"ListModule":     consts.ModuleListModule,
	"LoadModule":     consts.ModuleLoadModule,
	"RefreshModule":  consts.ModuleRefreshModule,
	"ListAddon":      consts.ModuleListAddon,
	"LoadAddon":      consts.ModuleLoadAddon,
	"ExecuteAddon":   consts.ModuleExecuteAddon,
	"ListTasks":      consts.ModuleListTask,
	"QueryTask":      consts.ModuleQueryTask,
	"CancelTask":     consts.ModuleCancelTask,
	"Clear":          consts.ModuleClear,
	"Execute":        consts.ModuleExecute,
	"Upload":         consts.ModuleUpload,
	"Download":       consts.ModuleDownload,
	"DownloadDir":    consts.ModuleDownload,
	"Switch":         consts.ModuleSwitch,
	"ServiceList":    consts.ModuleServiceList,
	"ServiceCreate":  consts.ModuleServiceCreate,
	"ServiceStart":   consts.ModuleServiceStart,
	"ServiceStop":    consts.ModuleServiceStop,
	"ServiceQuery":   consts.ModuleServiceQuery,
	"ServiceDelete":  consts.ModuleServiceDelete,
	"RegQuery":       consts.ModuleRegQuery,
	"RegAdd":         consts.ModuleRegAdd,
	"RegDelete":      consts.ModuleRegDelete,
	"RegListKey":     consts.ModuleRegListKey,
	"RegListValue":   consts.ModuleRegListValue,
	"TaskSchdList":   consts.ModuleTaskSchdList,
	"TaskSchdCreate": consts.ModuleTaskSchdCreate,
	"TaskSchdStart":  consts.ModuleTaskSchdStart,
	"TaskSchdStop":   consts.ModuleTaskSchdStop,
	"TaskSchdDelete": consts.ModuleTaskSchdDelete,
	"TaskSchdQuery":  consts.ModuleTaskSchdQuery,
	"TaskSchdRun":    consts.ModuleTaskSchdRun,
	"Pwd":            consts.ModulePwd,
	"Ls":             consts.ModuleLs,
	"Cd":             consts.ModuleCd,
	"Rm":             consts.ModuleRm,
	"Mv":             consts.ModuleMv,
	"Cp":             consts.ModuleCp,
	"Cat":            consts.ModuleCat,
	"Mkdir":          consts.ModuleMkdir,
	"Touch":          consts.ModuleTouch,
	"EnumDrivers":    consts.ModuleEnumDrivers,
	"Runas":          consts.ModuleRunas,
	"Privs":          consts.ModulePrivs,
	"Rev2Self":       consts.ModuleRev2Self,
	"GetSystem":      consts.ModuleGetSystem,
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
