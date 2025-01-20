package rpc

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gookit/config/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"net"
	"runtime/debug"
)

var (
	pipelinesCh     = make(map[string]grpc.ServerStream)
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
func StartClientListener(address string) (*grpc.Server, net.Listener, error) {
	logs.Log.Importantf("[server] starting gRPC console on %s", address)

	InitLogs(config.Bool("debug"))
	tlsConfig := certutils.GetOperatorServerMTLSConfig(configs.GetServerConfig().IP)
	creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", address)
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
