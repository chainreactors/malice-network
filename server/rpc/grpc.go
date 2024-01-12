package rpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/core"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"net"
	"runtime"
	"runtime/debug"
)

var (
	// ErrInvalidBeaconID - Invalid Beacon ID in request
	ErrInvalidBeaconID = status.Error(codes.InvalidArgument, "Invalid beacon ID")
	// ErrInvalidBeaconTaskID - Invalid Beacon ID in request
	ErrInvalidBeaconTaskID = status.Error(codes.InvalidArgument, "Invalid beacon task ID")

	// ErrInvalidSessionID - Invalid Session ID in request
	ErrInvalidSessionID = status.Error(codes.InvalidArgument, "Invalid session ID")

	// ErrMissingRequestField - Returned when a request does not contain a commonpb.Request
	ErrMissingRequestField = status.Error(codes.InvalidArgument, "Missing session request field")
	// ErrAsyncNotSupported - Unsupported mode / command type
	ErrAsyncNotSupported = status.Error(codes.Unavailable, "Async not supported for this command")
	// ErrDatabaseFailure - Generic database failure error (real error is logged)
	ErrDatabaseFailure = status.Error(codes.Internal, "Database operation failed")

	// ErrInvalidName - Invalid name
	ErrInvalidName        = status.Error(codes.InvalidArgument, "Invalid session name, alphanumerics and _-. only")
	ErrNotFoundSession    = status.Error(codes.InvalidArgument, "Session ID not found")
	ErrNotFoundListener   = status.Error(codes.InvalidArgument, "Pipeline not found")
	ErrNotFoundClientName = status.Error(codes.InvalidArgument, "Client name not found")
	//ErrInvalidBeaconTaskCancelState = status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid task state, must be '%s' to cancel", models.PENDING))
)

var (
	listenersCh     = make(map[string]grpc.ServerStream)
	authLog, rpcLog *logs.Logger
)

func InitLogs(debug bool) {
	if debug {
		authLog = configs.NewDebugLog("auth")
		rpcLog = configs.NewDebugLog("rpc")
	} else {
		authLog = configs.NewFileLog("auth")
		rpcLog = configs.NewFileLog("rpc")
	}
}

// StartClientListener - Start a mutual TLS listener
func StartClientListener(port uint16) (*grpc.Server, net.Listener, error) {
	logs.Log.Importantf("Starting gRPC console on 0.0.0.0:%d", port)

	tlsConfig := getOperatorServerMTLSConfig("operator")

	creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, err
	}
	InitLogs(false)
	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(consts.ServerMaxMessageSize),
		grpc.MaxSendMsgSize(consts.ServerMaxMessageSize),
	}
	commonOptions := append(options, middleware.CommonMiddleware(rpcLog)...)
	rootOptions := append(options, middleware.AuthMiddleware(authLog)...)
	grpcServer := grpc.NewServer(commonOptions...)
	rootGrpcServer := grpc.NewServer(rootOptions...)
	clientrpc.RegisterMaliceRPCServer(grpcServer, NewServer())
	clientrpc.RegisterRootRPCServer(rootGrpcServer, NewServer())
	listenerrpc.RegisterImplantRPCServer(grpcServer, NewServer())
	listenerrpc.RegisterListenerRPCServer(grpcServer, NewServer())
	go func() {
		panicked := true
		defer func() {
			if panicked {
				logs.Log.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			logs.Log.Warnf("gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}

func newGenericRequest(msg proto.Message, opts ...int) *GenericRequest {
	req := &GenericRequest{
		Message: msg,
	}
	if opts == nil {
		req.Task = req.NewTask(1)
	} else {
		req.Task = req.NewTask(opts[0])
	}

	dbSession := db.Session()
	err := dbSession.Create(models.ConvertToTaskDB(req.Task)).Error
	if err != nil {
		logs.Log.Errorf("cannot create task %s , %s in db", req.Task.Id, err.Error())
	}
	return req
}

type GenericRequest struct {
	proto.Message
	Task  *core.Task
	Spite *commonpb.Spite
}

func (r *GenericRequest) NewTask(total int) *core.Task {
	task := core.NewTask(string(proto.MessageName(r.Message).Name()), total)
	return task
}

func (r *GenericRequest) NewSpite(msg proto.Message) (*commonpb.Spite, error) {
	r.Spite = &commonpb.Spite{
		Timeout: uint64(consts.MinTimeout.Seconds()),
		TaskId:  r.Task.Id,
	}
	var err error
	r.Spite, err = types.BuildSpite(r.Spite, msg)
	if err != nil {
		return nil, err
	}
	return r.Spite, nil
}

func (r *GenericRequest) SetTotal(total int) {
	r.Task.Total = total
}

func (r *GenericRequest) SetCallback(callback func()) {
	r.Task.Callback = callback
}

type Server struct {
	// Magical methods to break backwards compatibility
	// Here be dragons: https://github.com/grpc/grpc-go/issues/3794
	clientrpc.UnimplementedMaliceRPCServer
	listenerrpc.UnimplementedImplantRPCServer
	listenerrpc.UnimplementedListenerRPCServer
	clientrpc.UnimplementedRootRPCServer
}

// NewServer - Create new server instance
func NewServer() *Server {
	// todo event
	return &Server{}
}

func (rpc *Server) getSessionID(ctx context.Context) (string, error) {
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

func (rpc *Server) getListenerID(ctx context.Context) (string, error) {
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

func (rpc *Server) getClientName(ctx context.Context) string {
	//md, ok := metadata.FromIncomingContext(ctx)
	//if !ok {
	//	return "", ErrNotFoundClientName
	//}
	//if sid := md.Get("client_name"); len(sid) > 0 {
	//	return sid[0], nil
	//} else {
	//	return "", ErrNotFoundClientName
	//}
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

// genericHandler - Pass the request to the Sliver/Session
func (rpc *Server) genericHandler(ctx context.Context, req *GenericRequest) (proto.Message, error) {
	var err error

	sid, err := rpc.getSessionID(ctx)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}

	session, ok := core.Sessions.Get(sid)
	if !ok {
		return nil, ErrInvalidSessionID
	}
	session.Tasks.Add(req.Task)
	spite, err := req.NewSpite(req.Message)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}

	data, err := session.RequestAndWait(
		&lispb.SpiteSession{SessionId: sid, TaskId: req.Task.Id, Spite: spite},
		listenersCh[session.ListenerId],
		consts.MinTimeout)
	if err != nil {
		return nil, err
	}
	session.DeleteResp(req.Task.Id)
	resp, err := types.ParseSpite(data)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (rpc *Server) asyncGenericHandler(ctx context.Context, req *GenericRequest) (*commonpb.AsyncStatus, chan *commonpb.Spite, error) {
	var err error
	sid, err := rpc.getSessionID(ctx)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, err
	}

	session, ok := core.Sessions.Get(sid)
	if !ok {
		return nil, nil, ErrInvalidSessionID
	}
	session.Tasks.Add(req.Task)
	spite, err := req.NewSpite(req.Message)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, err
	}
	spite.Async = true
	stat, out, err := session.RequestWithAsync(
		&lispb.SpiteSession{SessionId: sid, TaskId: req.Task.Id, Spite: spite},
		listenersCh[session.ListenerId],
		consts.MinTimeout)
	if err != nil {
		return nil, nil, err
	}

	return stat, out, nil
}

// streamGenericHandler - Generic handler for async request/response's for beacon tasks
func (rpc *Server) streamGenericHandler(ctx context.Context, req *GenericRequest) (chan *commonpb.Spite, chan *commonpb.Spite, *commonpb.AsyncStatus, error) {
	var err error
	sid, err := rpc.getSessionID(ctx)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, nil, err
	}

	session, ok := core.Sessions.Get(sid)
	if !ok {
		return nil, nil, nil, ErrInvalidSessionID
	}
	session.Tasks.Add(req.Task)
	spite, err := req.NewSpite(req.Message)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, nil, err
	}
	spite.Async = true
	in, out, stat, err := session.RequestWithStream(
		&lispb.SpiteSession{SessionId: sid, TaskId: req.Task.Id, Spite: spite},
		listenersCh[session.ListenerId],
		consts.MinTimeout)
	if err != nil {
		return nil, nil, stat, err
	}

	return in, out, stat, nil
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

func (rpc *Server) getClientCommonName(ctx context.Context) string {
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

// getOperatorServerMTLSConfig - Get the TLS config for the operator server
func getOperatorServerMTLSConfig(host string) *tls.Config {
	caCert, _, err := certs.GetCertificateAuthority(certs.SERVERCA)
	if err != nil {
		logs.Log.Errorf("Failed to load CA %s", err)
		return nil
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)
	certPEM, keyPEM, err := certs.OperatorServerGenerateCertificate(host)
	if err != nil {
		logs.Log.Errorf("Failed to load certificate %s", err)
	}
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		logs.Log.Errorf("Error loading server certificate: %v", err)
	}

	tlsConfig := &tls.Config{
		RootCAs:      caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
	}

	return tlsConfig
}
