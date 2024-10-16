package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
	"runtime"
)

func newGenericRequest(ctx context.Context, msg proto.Message, opts ...int) (*GenericRequest, error) {
	req := &GenericRequest{
		Message: msg,
		Callee:  getCallee(ctx),
	}
	if session, err := getSession(ctx); err == nil {
		req.Session = session
	} else {
		return nil, err
	}

	if opts == nil {
		req.Count = 1
	} else {
		req.Count = opts[0]
	}
	return req, nil
}

type GenericRequest struct {
	proto.Message
	Task    *core.Task
	Count   int
	Session *core.Session
	Callee  string
}

func (r *GenericRequest) InitSpite() (*implantpb.Spite, error) {
	spite := &implantpb.Spite{
		Timeout: uint64(consts.MinTimeout.Seconds()),
		Async:   true,
	}
	var err error
	spite, err = types.BuildSpite(spite, r.Message)
	if err != nil {
		return nil, err
	}
	r.Task = r.Session.NewTask(spite.Name, r.Count)
	r.Task.Callee = r.Callee
	spite.TaskId = r.Task.Id
	err = db.AddTask(r.Task)
	if err != nil {
		return nil, err
	}
	return spite, nil
}

func (r *GenericRequest) NewSpite(msg proto.Message) (*implantpb.Spite, error) {
	spite := &implantpb.Spite{
		Timeout: uint64(consts.MinTimeout.Seconds()),
		Async:   true,
	}
	var err error
	spite, err = types.BuildSpite(spite, msg)
	if err != nil {
		return nil, err
	}
	return spite, nil
}

func (r *GenericRequest) SetCallback(callback func()) {
	r.Task.Callback = callback
}

func (r *GenericRequest) HandlerResponse(ch chan *implantpb.Spite, typ types.MsgName, callbacks ...func(spite *implantpb.Spite)) {
	resp := <-ch
	r.Session.AddMessage(resp, r.Task.Cur)
	err := handler.AssertStatusAndResponse(resp, typ)
	if err != nil {
		logs.Log.Debug(err)
		r.Task.Panic(buildErrorEvent(r.Task, err))
		return
	}
	if callbacks != nil {
		r.SetCallback(func() {
			for _, callback := range callbacks {
				callback(resp)
			}
		})
	}
	r.Task.Done(resp, "")
	r.Task.Finish(resp, "")
	err = db.UpdateTask(r.Task)
	if err != nil {
		logs.Log.Errorf("update task cur failed %s", err)
		return
	}
	respByte, err := proto.Marshal(resp)
	if err != nil {
		logs.Log.Errorf("Failed to marshal resp to byte: %v", err)
		return
	}
	err = r.Session.TaskLog(r.Task, respByte)
	if err != nil {
		logs.Log.Errorf("Failed to log task: %v", err)
	}
	return

}

func buildErrorEvent(task *core.Task, err error) core.Event {
	var eventErr string

	switch {
	case errors.Is(err, handler.ErrNilStatus):
		eventErr = handler.ErrNilStatus.Error()
	case errors.Is(err, handler.ErrAssertFailure):
		eventErr = handler.ErrAssertFailure.Error()
	case errors.Is(err, handler.ErrNilResponseBody):
		eventErr = handler.ErrNilResponseBody.Error()
	case errors.Is(err, ErrMissingRequestField):
		eventErr = ErrMissingRequestField.Error()
	default:
		eventErr = err.Error()
	}

	return core.Event{
		EventType: consts.EventTask,
		Op:        consts.CtrlTaskError,
		Task:      task.ToProtobuf(),
		Err:       eventErr,
	}
}

func (rpc *Server) GenericHandler(ctx context.Context, req *GenericRequest) (chan *implantpb.Spite, error) {
	spite, err := req.InitSpite()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}
	if pipelinesCh[req.Session.PipelineID] == nil {
		return nil, ErrNotFoundPipeline
	}
	out, err := req.Session.RequestWithAsync(
		&lispb.SpiteSession{SessionId: req.Session.ID, TaskId: req.Task.Id, Spite: spite},
		pipelinesCh[req.Session.PipelineID],
		consts.MinTimeout)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// StreamGenericHandler - Generic handler for async request/response's for beacon tasks
func (rpc *Server) StreamGenericHandler(ctx context.Context, req *GenericRequest) (chan *implantpb.Spite, chan *implantpb.Spite, error) {
	spite, err := req.InitSpite()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, err
	}
	if pipelinesCh[req.Session.PipelineID] == nil {
		return nil, nil, ErrNotFoundPipeline
	}
	in, out, err := req.Session.RequestWithStream(
		&lispb.SpiteSession{SessionId: req.Session.ID, TaskId: req.Task.Id, Spite: spite},
		pipelinesCh[req.Session.PipelineID],
		consts.MinTimeout)
	if err != nil {
		return nil, nil, err
	}

	return in, out, nil
}

func (rpc *Server) GetBasic(ctx context.Context, _ *clientpb.Empty) (*clientpb.Basic, error) {
	return &clientpb.Basic{
		Major: 0,
		Minor: 0,
		Patch: 2,
		Os:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}, nil
}

func getSessionID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrNotFoundSession
	}
	if sid := md.Get("session_id"); len(sid) > 0 {
		return sid[0], nil
	} else {
		return "", ErrNotFoundSession
	}
}

func getCallee(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return consts.CalleeCMD
	}
	if callee := md.Get("callee"); len(callee) > 0 {
		return callee[0]
	} else {
		return consts.CalleeCMD
	}
}

func getSession(ctx context.Context) (*core.Session, error) {
	sid, err := getSessionID(ctx)
	if err != nil {
		return nil, err
	}

	session, ok := core.Sessions.Get(sid)
	if !ok {
		return nil, ErrInvalidSessionID
	}
	return session, nil
}

func getListenerID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrNotFoundListener
	}
	if sid := md.Get("listener_id"); len(sid) > 0 {
		return sid[0], nil
	} else {
		return "", ErrNotFoundListener
	}
}

func getRemoteAddr(ctx context.Context) string {
	// Extract peer information from context
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}

	// Check if the peer address is a net.Addr
	if p.Addr == nil {
		return ""
	}

	// Return the remote address as a string
	return p.Addr.String()
}

func getPipelineID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrNotFoundPipeline
	}
	if sid := md.Get("pipeline_id"); len(sid) > 0 {
		return sid[0], nil
	} else {
		return "", ErrNotFoundPipeline
	}
}

func getClientName(ctx context.Context) string {
	client, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}
	tlsAuth, ok := client.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return ""
	}
	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
		return ""
	}
	if tlsAuth.State.VerifiedChains[0][0].Subject.CommonName != "" {
		return tlsAuth.State.VerifiedChains[0][0].Subject.CommonName
	}
	return ""
}
