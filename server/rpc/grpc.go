package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/gookit/config/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"net"
	"runtime"
	"runtime/debug"
)

var (
	// ErrInvalidSessionID - Invalid Session ID in request
	ErrInvalidSessionID = status.Error(codes.InvalidArgument, "Invalid session ID")

	// ErrMissingRequestField - Returned when a request does not contain a  implantpb.Request
	ErrMissingRequestField = status.Error(codes.InvalidArgument, "Missing session request field")
	// ErrAsyncNotSupported - Unsupported mode / command type
	ErrAsyncNotSupported = status.Error(codes.Unavailable, "Async not supported for this command")
	// ErrDatabaseFailure - Generic database failure error (real error is logged)
	ErrDatabaseFailure = status.Error(codes.Internal, "Database operation failed")
	ErrNilStatus       = status.Error(codes.InvalidArgument, "Nil status or unknown error")
	ErrAssertFailure   = status.Error(codes.InvalidArgument, "Assert spite type failure")
	ErrNilResponseBody = status.Error(codes.InvalidArgument, "Must return spite body")
	// ErrInvalidName - Invalid name
	ErrInvalidName     = status.Error(codes.InvalidArgument, "Invalid session name, alphanumerics and _-. only")
	ErrNotFoundSession = status.Error(codes.NotFound, "Session ID not found")
	ErrNotFoundTask    = status.Error(codes.NotFound, "Task ID not found")

	ErrNotFoundListener    = status.Error(codes.NotFound, "Pipeline not found")
	ErrNotFoundClientName  = status.Error(codes.NotFound, "Client name not found")
	ErrNotFoundTaskContent = status.Error(codes.NotFound, "Task content not found")
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

	InitLogs(config.Bool("debug"))
	tlsConfig := certs.GetOperatorServerMTLSConfig("server")
	creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, nil, err
	}
	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(consts.ServerMaxMessageSize),
		grpc.MaxSendMsgSize(consts.ServerMaxMessageSize),
	}

	//options = append(options, authInterceptor()...)
	//rootOptions := buildOptions(options, authInterceptor()...)
	grpcServer := grpc.NewServer(buildOptions(
		options,
		logInterceptor(rpcLog),
		auditInterceptor(),
		authInterceptor(rpcLog))...)
	clientrpc.RegisterMaliceRPCServer(grpcServer, NewServer())
	clientrpc.RegisterRootRPCServer(grpcServer, NewServer())
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

//
//func DaemonStart(server *configs.ServerConfig, cfg *configs.ListenerConfig) {
//	_, ln, err := StartClientListener(server.GRPCPort)
//	if err != nil {
//		logs.Log.Errorf("cannot start gRPC server, %s", err.Error())
//		return
//	}
//	err = listener.NewListener(server, cfg)
//	if err != nil {
//		logs.Log.Errorf("cannot start listeners , %s ", err.Error())
//		return
//	}
//	done := make(chan bool)
//	signals := make(chan os.Signal, 1)
//	signal.Notify(signals, syscall.SIGTERM)
//	go func() {
//		<-signals
//		logs.Log.Infof("Received SIGTERM, exiting ...")
//		ln.Close()
//		done <- true
//	}()
//	<-done
//}

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

// genericHandler - Pass the request to the Sliver/Session
//func (rpc *Server) genericHandler(ctx context.Context, req *GenericRequest) (proto.Message, error) {
//	spite, err := req.NewSpite(req.Message)
//	if err != nil {
//		logs.Log.Errorf(err.Error())
//		return nil, err
//	}
//	data, err := req.Session.RequestAndWait(
//		&lispb.SpiteSession{SessionId: req.Session.ID, TaskId: req.Task.Id, Spite: spite},
//		listenersCh[req.Session.ListenerId],
//		consts.MinTimeout)
//	if err != nil {
//		return nil, err
//	}
//	req.Session.DeleteResp(req.Task.Id)
//	resp, err := types.ParseSpite(data)
//	if err != nil {
//		return nil, err
//	}
//	return resp, nil
//}

func (rpc *Server) asyncGenericHandler(ctx context.Context, req *GenericRequest) (chan *implantpb.Spite, error) {
	spite, err := req.NewSpite(req.Message)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}

	out, err := req.Session.RequestWithAsync(
		&lispb.SpiteSession{SessionId: req.Session.ID, TaskId: req.Task.Id, Spite: spite},
		listenersCh[req.Session.ListenerId],
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
		listenersCh[req.Session.ListenerId],
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
