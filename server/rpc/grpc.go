package rpc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/gookit/config/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	pipelinesCh     sync.Map
	authLog, rpcLog *logs.Logger
)

var serveClientGRPC = func(server *grpc.Server, ln net.Listener) error {
	return server.Serve(ln)
}

func runClientListener(grpcServer *grpc.Server, ln net.Listener) error {
	if err := serveClientGRPC(grpcServer, ln); err != nil {
		return fmt.Errorf("gRPC server exited with error: %w", err)
	}
	return nil
}

func normalizeClientListenerError(err error) error {
	if errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return err
}

var newClientListenerRuntime = func(address, serverIP string, debug bool) (*grpc.Server, net.Listener, error) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	grpcServer, err := NewClientGRPCServer(serverIP, debug)
	if err != nil {
		_ = ln.Close()
		return nil, nil, err
	}
	return grpcServer, ln, nil
}

func InitLogs(debug bool) {
	if debug {
		authLog = configs.NewDebugLog("auth")
		rpcLog = configs.NewDebugLog("rpc")
	} else {
		authLog = configs.NewFileLog("auth")
		rpcLog = configs.NewFileLog("rpc")
	}
}

func CloseLogs() {
	if authLog != nil {
		authLog.Close(false)
		authLog = nil
	}
	if rpcLog != nil {
		rpcLog.Close(false)
		rpcLog = nil
	}
}

// ResetTransientRPCState clears process-global RPC stream state that should
// not leak across isolated in-process test harnesses.
func ResetTransientRPCState() {
	pipelinesCh = sync.Map{}
	ptyStreamingSessions = sync.Map{}
}

func RegisterRPCServices(grpcServer *grpc.Server) {
	clientrpc.RegisterMaliceRPCServer(grpcServer, NewServer())
	clientrpc.RegisterRootRPCServer(grpcServer, NewServer())
	listenerrpc.RegisterListenerRPCServer(grpcServer, NewServer())
}

func NewClientGRPCServer(serverIP string, debug bool) (*grpc.Server, error) {
	InitLogs(debug)
	tlsConfig := certutils.GetOperatorServerMTLSConfig(serverIP)
	if tlsConfig == nil {
		return nil, fmt.Errorf("failed to build operator server TLS config")
	}
	creds := credentials.NewTLS(tlsConfig)
	options := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.MaxRecvMsgSize(consts.ServerMaxMessageSize),
		grpc.MaxSendMsgSize(consts.ServerMaxMessageSize),
	}

	grpcServer := grpc.NewServer(buildOptions(
		options,
		[]grpc.UnaryServerInterceptor{
			logInterceptor(rpcLog),
			auditInterceptor(),
			authInterceptor(authLog),
		},
		[]grpc.StreamServerInterceptor{
			authStreamInterceptor(authLog),
		},
	)...)
	RegisterRPCServices(grpcServer)
	return grpcServer, nil
}

// StartClientListener - Start a mutual TLS listener
func StartClientListener(address string) (*grpc.Server, net.Listener, error) {
	logs.Log.Importantf("[server] starting gRPC console on %s", address)

	serverIP := ""
	if serverConfig := configs.GetServerConfig(); serverConfig != nil {
		serverIP = serverConfig.IP
	}
	grpcServer, ln, err := newClientListenerRuntime(address, serverIP, config.Bool("debug"))
	if err != nil {
		return nil, nil, err
	}
	core.GoGuarded("grpc-client-listener", func() error {
		return normalizeClientListenerError(runClientListener(grpcServer, ln))
	}, func(err error) {
		core.LogGuardedError("grpc-client-listener")(err)
		logs.Log.Errorf("grpc-client-listener fatal, exiting")
		os.Exit(1)
	})
	return grpcServer, ln, nil
}

type Server struct {
	// Magical methods to break backwards compatibility
	// Here be dragons: https://github.com/grpc/grpc-go/issues/3794
	clientrpc.UnimplementedMaliceRPCServer
	listenerrpc.UnimplementedListenerRPCServer
	clientrpc.UnimplementedRootRPCServer
}

// NewServer - Create new server instance
func NewServer() *Server {
	// todo event
	return &Server{}
}

//func (rpc *Server) genericHandler(ctx context.Context, req *GenericRequest) (proto.Message, error) {
//	spite, err := req.NewSpite(req.Message)
//	if err != nil {
//		logs.Log.Errorf(err.Error())
//		return nil, err
//	}
//	data, err := req.Session.RequestAndWait(
//		&clientpb.SpiteSession{SessionId: req.Session.ID, TaskId: req.Task.Id, Spite: spite},
//		pipelinesCh[req.Session.PipelineID],
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
