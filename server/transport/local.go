package transport

import (
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/rpc"
	"github.com/chainreactors/malice-network/utils/constant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func LocalListener() (*grpc.Server, *bufconn.Listener, error) {
	//bufConnLog.Infof("Binding gRPC to listener ...")
	ln := bufconn.Listen(constant.BufSize)
	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(constant.ServerMaxMessageSize),
		grpc.MaxSendMsgSize(constant.ServerMaxMessageSize),
	}
	options = append(options)
	grpcServer := grpc.NewServer(options...)
	clientrpc.RegisterMaliceRPCServer(grpcServer, rpc.NewServer())
	go func() {
		panicked := true
		defer func() {
			if panicked {
				//bufConnLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			//bufConnLog.Fatalf("gRPC local listener error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}
