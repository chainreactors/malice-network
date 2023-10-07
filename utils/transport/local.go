package transport

import (
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 2 * mb

//var (
//	bufConnLog = log.NamedLogger("transport", "local")
//)

// LocalListener - Bind gRPC server to an in-memory listener, which is
//
//	typically used for unit testing, but ... it should be fine
func LocalListener() (*grpc.Server, *bufconn.Listener, error) {
	// TODO - log binding gRPC/bufconn to listener
	//bufConnLog.Infof("Binding gRPC/bufconn to listener ...")
	ln := bufconn.Listen(bufSize)
	options := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(ServerMaxMessageSize),
		grpc.MaxSendMsgSize(ServerMaxMessageSize),
	}
	options = append(options, initMiddleware(false)...)
	grpcServer := grpc.NewServer(options...)
	// TODO - RegisterSliverRPCServer
	//rpcpb.RegisterSliverRPCServer(grpcServer, rpc.NewServer())
	go func() {
		panicked := true
		defer func() {
			if panicked {
				// TODO - log stacktrace from panic
				fmt.Println(debug.Stack())
				//bufConnLog.Errorf("stacktrace from panic: %s", string(debug.Stack()))
			}
		}()
		if err := grpcServer.Serve(ln); err != nil {
			// TODO - log gRPC local listener error
			//bufConnLog.Fatalf("gRPC local listener error: %v", err)
		} else {
			panicked = false
		}
	}()
	return grpcServer, ln, nil
}
