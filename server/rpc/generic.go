package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

var (
	Ver        = "latest"
	Commit     = ""
	Buildstamp = ""
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

func (r *GenericRequest) TaskContext(spite *implantpb.Spite) *clientpb.TaskContext {
	return &clientpb.TaskContext{
		Task:    r.Task.ToProtobuf(),
		Session: r.Session.ToProtobufLite(),
		Spite:   spite,
	}
}

func (r *GenericRequest) InitSpite(ctx context.Context) (*implantpb.Spite, error) {
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
	clientName := getClientName(ctx)
	r.Task.CallBy = clientName
	err = db.AddTask(r.Task.ToProtobuf())
	if err != nil {
		return nil, err
	}

	spiteBytes, err := proto.Marshal(spite)
	if err != nil {
		return nil, err
	}

	requestPath, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(r.Task.SessionId, consts.RequestPath, strconv.Itoa(int(r.Task.Id))))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(requestPath), 0o700); err != nil {
		return nil, err
	}
	err = os.WriteFile(requestPath, spiteBytes, 0o600)
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

func (r *GenericRequest) HandlerSpite(spite *implantpb.Spite) error {
	r.Session.AddMessage(spite, r.Task.Cur)
	r.Task.Done(spite, "")
	err := db.UpdateTask(r.Task.ToProtobuf())
	if err != nil {
		return err
	}
	err = r.Session.TaskLog(r.Task, spite)
	if err != nil {
		return err
	}
	return nil
}

func runTaskHandler(task *core.Task, fn func() error, cleanups ...func()) <-chan error {
	if task == nil {
		return core.GoGuarded("rpc-task", fn, core.LogGuardedError("rpc-task"), cleanups...)
	}
	return core.GoTask(task, "rpc-task:"+task.Name(), fn, cleanups...)
}

func (r *GenericRequest) HandlerResponse(ch chan *implantpb.Spite, typ types.MsgName, callbacks ...func(spite *implantpb.Spite)) {
	runTaskHandler(r.Task, func() error {
		resp := <-ch

		err := types.AssertStatusAndSpite(resp, typ)
		if err != nil {
			return buildTaskError(err)
		}

		err = r.HandlerSpite(resp)
		if err != nil {
			return fmt.Errorf("handler spite: %w", err)
		}
		if callbacks != nil {
			r.SetCallback(func() {
				for _, callback := range callbacks {
					callback(resp)
				}
			})
		}
		r.Task.Finish(resp, "")
		return nil
	}, r.Task.Close)
}

func buildTaskError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, types.ErrNilStatus):
		return types.ErrNilStatus
	case errors.Is(err, types.ErrAssertFailure):
		return types.ErrAssertFailure
	case errors.Is(err, types.ErrNilResponseBody):
		return types.ErrNilResponseBody
	case errors.Is(err, types.ErrMissingRequestField):
		return types.ErrMissingRequestField
	default:
		return err
	}
}

func (rpc *Server) GenericInternal(ctx context.Context, req proto.Message, expect types.MsgName, callbacks ...func(spite *implantpb.Spite)) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, expect, callbacks...)
	return greq.Task.ToProtobuf(), nil
}

func (rpc *Server) GenericHandler(ctx context.Context, req *GenericRequest) (chan *implantpb.Spite, error) {
	spite, err := req.InitSpite(ctx)
	if err != nil {
		logs.Log.Errorf("%s", err.Error())
		return nil, err
	}
	streamVal, ok := pipelinesCh.Load(req.Session.PipelineID)
	if !ok || streamVal == nil {
		return nil, types.ErrNotFoundPipeline
	}
	out, err := req.Session.RequestWithAsync(
		&clientpb.SpiteRequest{Session: req.Session.ToProtobufLite(), Task: req.Task.ToProtobuf(), Spite: spite},
		streamVal.(grpc.ServerStream),
		consts.MinTimeout)
	if err != nil {
		return nil, err
	}

	return out, nil
}

// StreamGenericHandler - Generic handler for async request/response's for beacon tasks
func (rpc *Server) StreamGenericHandler(ctx context.Context, req *GenericRequest) (*core.SpiteStreamWriter, chan *implantpb.Spite, error) {
	spite, err := req.InitSpite(ctx)
	if err != nil {
		logs.Log.Errorf("%s", err.Error())
		return nil, nil, err
	}
	streamVal, ok := pipelinesCh.Load(req.Session.PipelineID)
	if !ok || streamVal == nil {
		return nil, nil, types.ErrNotFoundPipeline
	}
	in, out, err := req.Session.RequestWithStream(
		&clientpb.SpiteRequest{Session: req.Session.ToProtobufLite(), Task: req.Task.ToProtobuf(), Spite: spite},
		streamVal.(grpc.ServerStream),
		consts.MinTimeout)
	if err != nil {
		return nil, nil, err
	}

	return in, out, nil
}

func (rpc *Server) GetBasic(ctx context.Context, _ *clientpb.Empty) (*clientpb.Basic, error) {
	timestamp, _ := strconv.ParseInt(Buildstamp, 10, 64)
	return &clientpb.Basic{
		Version:    Ver,
		Commit:     Commit,
		CompiledAt: timestamp,
		Os:         runtime.GOOS,
		Arch:       runtime.GOARCH,
	}, nil
}

func getSessionID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", types.ErrNotFoundSession
	}
	if sid := md.Get("session_id"); len(sid) > 0 {
		return sid[0], nil
	} else {
		return "", types.ErrNotFoundSession
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

	session, err := core.Sessions.Get(sid)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func getListenerID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", types.ErrNotFoundListener
	}
	if sid := md.Get("listener_id"); len(sid) > 0 {
		return sid[0], nil
	} else {
		return "", types.ErrNotFoundListener
	}
}

func getRemoteAddr(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}

	if p.Addr == nil {
		return ""
	}

	return p.Addr.String()
}

func getRemoteIp(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return ""
	}

	if p.Addr == nil {
		return ""
	}

	host, _, _ := net.SplitHostPort(p.Addr.String())
	return host
}

func getTimestamp(ctx context.Context) int64 {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return time.Now().Unix()
	}
	if timestamp := md.Get("timestamp"); len(timestamp) > 0 {
		if ts, err := strconv.ParseInt(timestamp[0], 10, 64); err == nil {
			return ts
		}
	}
	return time.Now().Unix()
}

func getPipelineID(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", types.ErrNotFoundPipeline
	}
	if sid := md.Get("pipeline_id"); len(sid) > 0 {
		return sid[0], nil
	} else {
		return "", types.ErrNotFoundPipeline
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

type contextRequestMeta struct {
	ContextType string
	Nonce       string
	Identifier  string
	FileName    string
	MediaKind   string
}

func getContextMeta(ctx context.Context) contextRequestMeta {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return contextRequestMeta{}
	}
	return contextRequestMeta{
		ContextType: getMetadataValue(md, "context"),
		Nonce:       getMetadataValue(md, "nonce"),
		Identifier:  getMetadataValue(md, "context-id"),
		FileName:    getMetadataValue(md, "context-name"),
		MediaKind:   getMetadataValue(md, "context-kind"),
	}
}

func getMetadataValue(md metadata.MD, key string) string {
	if values := md.Get(key); len(values) > 0 {
		return values[0]
	}
	return ""
}

func Handler(ctx context.Context, rpc *Server, req proto.Message, expect types.MsgName, callbacks ...func(spite *implantpb.Spite)) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}

	greq.HandlerResponse(ch, expect, callbacks...)
	return greq.Task.ToProtobuf(), nil
}
