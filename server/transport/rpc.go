package transport

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/server/rpc"
	"github.com/chainreactors/malice-network/utils/constant"
	"google.golang.org/grpc"
	"net"
)

// StartClientListener - Start a mutual TLS listener
func StartClientListener(host string, port uint16) (*grpc.Server, net.Listener, error) {
	logs.Log.Importantf("Starting gRPC console on %s:%d", host, port)

	//tlsConfig := getOperatorServerTLSConfig("multiplayer")

	//creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		//mtlsLog.Error(err)
		return nil, nil, err
	}
	options := []grpc.ServerOption{
		//grpc.Creds(creds),
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
				//mtlsLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			//mtlsLog.Warnf("gRPC server exited with error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}
