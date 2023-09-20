package transport

import (
	"fmt"
	"github.com/chainreactors/malice-network/proto/services"
	"github.com/chainreactors/malice-network/server/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	"net"
)

const (
	kb                          = 1024
	mb                          = kb * 1024
	gb                          = mb * 1024
	bufSize                     = 2 * mb
	ClientMaxReceiveMessageSize = 2 * gb
	// ServerMaxMessageSize - Server-side max GRPC message size
	ServerMaxMessageSize = 2 * gb
)

func LocalListener() (*grpc.Server, *bufconn.Listener, error) {
	//bufConnLog.Infof("Binding gRPC to listener ...")
	ln := bufconn.Listen(bufSize)
	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options)
	grpcServer := grpc.NewServer(options...)
	services.RegisterMaliceRPCServer(grpcServer, rpc.NewServer())
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

// StartClientListener - Start a mutual TLS listener
func StartClientListener(host string, port uint16) (*grpc.Server, net.Listener, error) {
	//mtlsLog.Infof("Starting gRPC  listener on %s:%d", host, port)

	//tlsConfig := getOperatorServerTLSConfig("multiplayer")

	//creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		//mtlsLog.Error(err)
		return nil, nil, err
	}
	options := []grpc.ServerOption{
		//grpc.Creds(creds),
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options)
	grpcServer := grpc.NewServer(options...)
	services.RegisterMaliceRPCServer(grpcServer, rpc.NewServer())
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
