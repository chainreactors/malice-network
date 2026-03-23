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
	"github.com/gookit/config/v2"
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

// genericAddTask is the function used to persist a new task to the database.
// It is a package-level variable to allow test injection.
var genericAddTask = func(task *clientpb.Task) error {
	return db.AddTask(task)
}

// genericWriteTaskRequest is the function used to cache the serialized request
// to disk. It is a package-level variable to allow test injection.
var genericWriteTaskRequest = func(task *core.Task, spite *implantpb.Spite) error {
	spiteBytes, err := proto.Marshal(spite)
	if err != nil {
		return err
	}
	path, err := taskRequestPath(task)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, spiteBytes, 0o600)
}

// taskRequestPath returns the filesystem path where a task's request spite is cached.
func taskRequestPath(task *core.Task) (string, error) {
	return fileutils.SafeJoin(configs.ContextPath, filepath.Join(task.SessionId, consts.RequestPath, strconv.Itoa(int(task.Id))))
}

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
	if err = genericAddTask(r.Task.ToProtobuf()); err != nil {
		r.Session.Tasks.Remove(r.Task.Id)
		return nil, err
	}

	if err = genericWriteTaskRequest(r.Task, spite); err != nil {
		r.Session.Tasks.Remove(r.Task.Id)
		_ = db.NewTaskQuery().WhereSessionID(r.Task.SessionId).WhereSeq(r.Task.Id).Delete()
		return nil, err
	}
	return spite, nil
}

// rollbackTask removes the task from runtime, database and request cache.
func (r *GenericRequest) rollbackTask() {
	if r.Task == nil || r.Session == nil {
		return
	}
	r.Session.Tasks.Remove(r.Task.Id)
	_ = db.NewTaskQuery().WhereSessionID(r.Task.SessionId).WhereSeq(r.Task.Id).Delete()
	r.Session.DeleteResp(r.Task.Id)
	if p, err := taskRequestPath(r.Task); err == nil {
		_ = os.Remove(p)
	}
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
	label := "rpc-task"
	if task != nil {
		label = "rpc-task:" + task.Name()
	}
	handler := core.LogGuardedError(label)
	if task != nil {
		handler = core.CombineErrorHandlers(handler, func(err error) {
			if core.EventBroker == nil {
				return
			}
			core.EventBroker.Publish(core.Event{
				EventType: consts.EventTask,
				Op:        consts.CtrlTaskError,
				Task:      task.ToProtobuf(),
				Err:       core.ErrorText(err),
			})
		})
	}
	return core.GoGuarded(label, fn, handler, cleanups...)
}

func (r *GenericRequest) HandlerResponse(ch chan *implantpb.Spite, typ types.MsgName, callbacks ...func(spite *implantpb.Spite)) {
	runTaskHandler(r.Task, func() error {
		resp, ok := recvSpite(r.Task.Ctx, ch)
		if !ok {
			return ErrTaskContextCancelled
		}

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
	case errors.Is(err, types.ErrMissingSessionRequestField):
		return types.ErrMissingSessionRequestField
	case errors.Is(err, types.ErrMissingRequestField):
		return types.ErrMissingRequestField
	default:
		return err
	}
}

var ErrTaskContextCancelled = fmt.Errorf("task context cancelled")

// recvSpite receives a single Spite from ch, respecting ctx cancellation.
// Returns (nil, false) if context is done or channel is closed.
func recvSpite(ctx context.Context, ch <-chan *implantpb.Spite) (*implantpb.Spite, bool) {
	select {
	case resp, ok := <-ch:
		return resp, ok
	case <-ctx.Done():
		return nil, false
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
		req.rollbackTask()
		return nil, types.ErrNotFoundPipeline
	}
	out, err := req.Session.RequestWithAsync(
		&clientpb.SpiteRequest{Session: req.Session.ToProtobufLite(), Task: req.Task.ToProtobuf(), Spite: spite},
		streamVal.(grpc.ServerStream),
		consts.MinTimeout)
	if err != nil {
		req.rollbackTask()
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
		req.rollbackTask()
		return nil, nil, types.ErrNotFoundPipeline
	}
	in, out, err := req.Session.RequestWithStream(
		&clientpb.SpiteRequest{Session: req.Session.ToProtobufLite(), Task: req.Task.ToProtobuf(), Spite: spite},
		streamVal.(grpc.ServerStream),
		consts.MinTimeout)
	if err != nil {
		req.rollbackTask()
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

// getPacketLength resolves the per-pipeline packet length from session context,
// falling back to the global config value.
func getPacketLength(ctx context.Context) int {
	session, err := getSession(ctx)
	if err != nil {
		return config.Int(consts.ConfigMaxPacketLength)
	}
	return session.GetPacketLength()
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

// AssertAndHandle covers the common pattern for *implantpb.Request handlers:
// assert request name → newGenericRequest → GenericHandler → HandlerResponse.
func (rpc *Server) AssertAndHandle(ctx context.Context, req *implantpb.Request, module types.MsgName, expect types.MsgName, callbacks ...func(*implantpb.Spite)) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	if err := types.AssertRequestName(req, module); err != nil {
		return nil, err
	}
	return rpc.GenericInternal(ctx, req, expect, callbacks...)
}

// GenericInternalWithSession is like GenericInternal but passes the GenericRequest
// to the callback so it can access Session, Task, and other request context.
func (rpc *Server) GenericInternalWithSession(ctx context.Context, req proto.Message, expect types.MsgName, callback func(*GenericRequest, *implantpb.Spite)) (*clientpb.Task, error) {
	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	ch, err := rpc.GenericHandler(ctx, greq)
	if err != nil {
		return nil, err
	}
	greq.HandlerResponse(ch, expect, func(spite *implantpb.Spite) {
		callback(greq, spite)
	})
	return greq.Task.ToProtobuf(), nil
}

// AssertAndHandleWithSession combines AssertRequestName with GenericInternalWithSession.
func (rpc *Server) AssertAndHandleWithSession(ctx context.Context, req *implantpb.Request, module types.MsgName, expect types.MsgName, callback func(*GenericRequest, *implantpb.Spite)) (*clientpb.Task, error) {
	if req == nil {
		return nil, types.ErrMissingRequestField
	}
	if err := types.AssertRequestName(req, module); err != nil {
		return nil, err
	}
	return rpc.GenericInternalWithSession(ctx, req, expect, callback)
}
