package rpc

import (
	"context"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/handler"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
	"runtime"
)

func newGenericRequest(ctx context.Context, msg proto.Message, opts ...int) (*GenericRequest, error) {
	req := &GenericRequest{
		Message: msg,
	}
	if session, err := getSession(ctx); err == nil {
		req.Session = session
	} else {
		return nil, err
	}

	if opts == nil {
		req.Task = req.NewTask(1)
	} else {
		req.Task = req.NewTask(opts[0])
	}
	return req, nil
}

type GenericRequest struct {
	proto.Message
	Task    *core.Task
	Session *core.Session
}

func (r *GenericRequest) NewTask(total int) *core.Task {
	return r.Session.NewTask(string(proto.MessageName(r.Message).Name()), total)
}

func (r *GenericRequest) NewSpite(msg proto.Message) (*implantpb.Spite, error) {
	spite := &implantpb.Spite{
		Timeout: uint64(consts.MinTimeout.Seconds()),
		TaskId:  r.Task.Id,
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

func (r *GenericRequest) HandlerAsyncResponse(ch chan *implantpb.Spite, typ types.MsgName, callbacks ...func(spite *implantpb.Spite)) {
	resp := <-ch
	r.Session.AddMessage(resp, r.Task.Cur)
	err := handler.AssertStatusAndResponse(resp, typ)
	if err != nil {
		logs.Log.Debug(err)
		r.Task.Panic(buildErrorEvent(r.Task, err))
		return
	}
	r.SetCallback(func() {
		if callbacks != nil {
			for _, callback := range callbacks {
				callback(resp)
			}
		}
	})
	r.Task.Done(core.Event{
		EventType: consts.EventTaskFinish,
		Task:      r.Task,
	})
}

func buildErrorEvent(task *core.Task, err error) core.Event {
	if errors.Is(err, handler.ErrNilStatus) {
		return core.Event{
			EventType: consts.EventTaskFinish,
			Task:      task,
			Err:       handler.ErrNilStatus.Error(),
		}
	} else if errors.Is(err, handler.ErrAssertFailure) {
		return core.Event{
			EventType: consts.EventTaskFinish,
			Task:      task,
			Err:       handler.ErrAssertFailure.Error(),
		}
	} else if errors.Is(err, handler.ErrNilResponseBody) {
		return core.Event{
			EventType: consts.EventTaskFinish,
			Task:      task,
			Err:       handler.ErrNilResponseBody.Error(),
		}
	} else if errors.Is(err, ErrMissingRequestField) {
		return core.Event{
			EventType: consts.EventTaskFinish,
			Task:      task,
			Err:       ErrMissingRequestField.Error(),
		}
	} else {
		return core.Event{
			EventType: consts.EventTaskFinish,
			Task:      task,
			Err:       err.Error(),
		}
	}
}

func (rpc *Server) asyncGenericHandler(ctx context.Context, req *GenericRequest) (chan *implantpb.Spite, error) {
	spite, err := req.NewSpite(req.Message)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
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

// streamGenericHandler - Generic handler for async request/response's for beacon tasks
func (rpc *Server) streamGenericHandler(ctx context.Context, req *GenericRequest) (chan *implantpb.Spite, chan *implantpb.Spite, error) {
	spite, err := req.NewSpite(req.Message)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, err
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
		Patch: 1,
		Os:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}, nil
}

// getTimeout - Get the specified timeout from the request or the default
//func (rpc *Server) getTimeout(req GenericRequest) time.Duration {
//
//	d := req.ProtoReflect().Descriptor().Fields().ByName("timeout")
//	timeout := req.ProtoReflect().Get(d).Int()
//	if time.Duration(timeout) < time.Second {
//		return constant.MinTimeout
//	}
//	return time.Duration(timeout)
//}

// // getError - Check an implant's response for Err and convert it to an `error` type
//func (rpc *Server) getError(resp GenericResponse) error {
//	respHeader := resp.GetResponse()
//	if respHeader != nil && respHeader.Err != "" {
//		return errors.New(respHeader.Err)
//	}
//	return nil
//}

//func (rpc *Server) getClientCommonName(ctx context.Context) string {
//	client, ok := peer.FromContext(ctx)
//	if !ok {
//		return ""
//	}
//	tlsAuth, ok := client.AuthInfo.(credentials.TLSInfo)
//	if !ok {
//		return ""
//	}
//	if len(tlsAuth.State.VerifiedChains) == 0 || len(tlsAuth.State.VerifiedChains[0]) == 0 {
//		return ""
//	}
//	if tlsAuth.State.VerifiedChains[0][0].Subject.CommonName != "" {
//		return tlsAuth.State.VerifiedChains[0][0].Subject.CommonName
//	}
//	return ""
//}

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
