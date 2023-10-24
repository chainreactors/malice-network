package rpc

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/constant"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/server/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"net"
	"runtime"
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
	ErrInvalidName      = status.Error(codes.InvalidArgument, "Invalid session name, alphanumerics and _-. only")
	ErrNotFoundSession  = status.Error(codes.InvalidArgument, "Session ID not found")
	ErrNotFoundListener = status.Error(codes.InvalidArgument, "Listener not found")
	//ErrInvalidBeaconTaskCancelState = status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid task state, must be '%s' to cancel", models.PENDING))
)

var listenersCh = make(map[string]grpc.ServerStream)

// StartClientListener - Start a mutual TLS listener
func StartClientListener(port uint16) (*grpc.Server, net.Listener, error) {
	logs.Log.Importantf("Starting gRPC console on 0.0.0.0:%d", port)

	//tlsConfig := getOperatorServerTLSConfig("multiplayer")

	//creds := credentials.NewTLS(tlsConfig)
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
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
	clientrpc.RegisterMaliceRPCServer(grpcServer, NewServer())
	listenerrpc.RegisterImplantRPCServer(grpcServer, NewServer())
	listenerrpc.RegisterListenerRPCServer(grpcServer, NewServer())
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

type Server struct {
	// Magical methods to break backwards compatibility
	// Here be dragons: https://github.com/grpc/grpc-go/issues/3794
	clientrpc.UnimplementedMaliceRPCServer
	listenerrpc.UnimplementedImplantRPCServer
	listenerrpc.UnimplementedListenerRPCServer
}

// NewServer - Create new server instance
func NewServer() *Server {
	// todo event
	return &Server{}
}

func (rpc *Server) sessionID(ctx context.Context) (string, error) {
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

func (rpc *Server) listenerID(ctx context.Context) (string, error) {
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

// GenericHandler - Pass the request to the Sliver/Session
func (rpc *Server) GenericHandler(ctx context.Context, req proto.Message) (proto.Message, error) {
	var err error
	//if req.Async {
	//	err = rpc.asyncGenericHandler(req, resp)
	//	return err
	//}

	// Sync request

	sid, err := rpc.sessionID(ctx)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}

	session := core.Sessions.Get(sid)
	if session == nil {
		return nil, ErrInvalidSessionID
	}

	spite := &commonpb.Spite{
		Timeout: uint64(constant.MinTimeout.Seconds()),
	}
	spite, err = types.BuildSpite(spite, req)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return nil, err
	}
	data, err := session.RequestAndWait(
		&lispb.SpiteSession{SessionId: sid, Spite: spite},
		listenersCh[session.ListenerId],
		constant.MinTimeout)
	if err != nil {
		return nil, err
	}

	resp, err := types.ParseSpite(data)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

//
//// asyncGenericHandler - Generic handler for async request/response's for beacon tasks
//func (rpc *Server) asyncGenericHandler(req GenericRequest, resp GenericResponse) error {
//	// VERY VERBOSE
//	// rpcLog.Debugf("Async Generic Handler: %#v", req)
//	//request := req.GetRequest()
//	//if request == nil {
//	//	return ErrMissingRequestField
//	//}
//	//
//	//beacon, err := db.BeaconByID(request.BeaconID)
//	//if beacon == nil || err != nil {
//	//	rpcLog.Errorf("Invalid beacon ID in request: %s", err)
//	//	return ErrInvalidBeaconID
//	//}
//	//
//	//// Overwrite unused implant fields before re-serializing
//	//request.SessionID = ""
//	//request.BeaconID = ""
//	//reqData, err := proto.Marshal(req)
//	//if err != nil {
//	//	return err
//	//}
//	//taskResponse := resp.GetResponse()
//	//taskResponse.Async = true
//	//taskResponse.BeaconID = beacon.ID.String()
//	//task, err := beacon.Task(&sliverpb.Envelope{
//	//	Type: sliverpb.MsgNumber(req),
//	//	Data: reqData,
//	//})
//	//if err != nil {
//	//	rpcLog.Errorf("Database error: %s", err)
//	//	return ErrDatabaseFailure
//	//}
//	//parts := strings.Split(string(req.ProtoReflect().Descriptor().FullName().Name()), ".")
//	//name := parts[len(parts)-1]
//	//task.Description = name
//	//err = db.Session().Save(task).Error
//	//if err != nil {
//	//	rpcLog.Errorf("Database error: %s", err)
//	//	return ErrDatabaseFailure
//	//}
//	//taskResponse.TaskID = task.ID.String()
//	//rpcLog.Debugf("Successfully tasked beacon: %#v", taskResponse)
//	return nil
//}

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

//func (rpc *Server) handler(ctx context.Context, msg proto.Message) (proto.Message, error) {
//	switch msg.(type) {
//	case *pluginpb.ExecRequest:
//		return rpc.Exec(ctx, msg.(*pluginpb.ExecRequest))
//	}
//}
