package forwardrpc

import (
	"context"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	ServiceName = "forwardrpc.ForwardListener"

	ControlStreamMethod = "/" + ServiceName + "/ControlStream"
	TaskStreamMethod    = "/" + ServiceName + "/TaskStream"
)

type ForwardListenerClient interface {
	ControlStream(ctx context.Context, opts ...grpc.CallOption) (ForwardListener_ControlStreamClient, error)
	TaskStream(ctx context.Context, opts ...grpc.CallOption) (ForwardListener_TaskStreamClient, error)
}

type forwardListenerClient struct {
	cc grpc.ClientConnInterface
}

func NewForwardListenerClient(cc grpc.ClientConnInterface) ForwardListenerClient {
	return &forwardListenerClient{cc: cc}
}

func (c *forwardListenerClient) ControlStream(ctx context.Context, opts ...grpc.CallOption) (ForwardListener_ControlStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &ForwardListener_ServiceDesc.Streams[0], ControlStreamMethod, opts...)
	if err != nil {
		return nil, err
	}
	return &forwardListenerControlStreamClient{ClientStream: stream}, nil
}

func (c *forwardListenerClient) TaskStream(ctx context.Context, opts ...grpc.CallOption) (ForwardListener_TaskStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &ForwardListener_ServiceDesc.Streams[1], TaskStreamMethod, opts...)
	if err != nil {
		return nil, err
	}
	return &forwardListenerTaskStreamClient{ClientStream: stream}, nil
}

type ForwardListener_ControlStreamClient interface {
	Send(*clientpb.JobCtrl) error
	Recv() (*clientpb.JobStatus, error)
	grpc.ClientStream
}

type forwardListenerControlStreamClient struct {
	grpc.ClientStream
}

func (x *forwardListenerControlStreamClient) Send(m *clientpb.JobCtrl) error {
	return x.ClientStream.SendMsg(m)
}

func (x *forwardListenerControlStreamClient) Recv() (*clientpb.JobStatus, error) {
	m := new(clientpb.JobStatus)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type ForwardListener_TaskStreamClient interface {
	Send(*clientpb.SpiteRequest) error
	Recv() (*clientpb.SpiteRequest, error)
	grpc.ClientStream
}

type forwardListenerTaskStreamClient struct {
	grpc.ClientStream
}

func (x *forwardListenerTaskStreamClient) Send(m *clientpb.SpiteRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *forwardListenerTaskStreamClient) Recv() (*clientpb.SpiteRequest, error) {
	m := new(clientpb.SpiteRequest)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type ForwardListenerServer interface {
	ControlStream(ForwardListener_ControlStreamServer) error
	TaskStream(ForwardListener_TaskStreamServer) error
}

type UnimplementedForwardListenerServer struct{}

func (UnimplementedForwardListenerServer) ControlStream(ForwardListener_ControlStreamServer) error {
	return status.Error(codes.Unimplemented, "method ControlStream not implemented")
}

func (UnimplementedForwardListenerServer) TaskStream(ForwardListener_TaskStreamServer) error {
	return status.Error(codes.Unimplemented, "method TaskStream not implemented")
}

func RegisterForwardListenerServer(s grpc.ServiceRegistrar, srv ForwardListenerServer) {
	s.RegisterService(&ForwardListener_ServiceDesc, srv)
}

type ForwardListener_ControlStreamServer interface {
	Send(*clientpb.JobStatus) error
	Recv() (*clientpb.JobCtrl, error)
	grpc.ServerStream
}

type forwardListenerControlStreamServer struct {
	grpc.ServerStream
}

func (x *forwardListenerControlStreamServer) Send(m *clientpb.JobStatus) error {
	return x.ServerStream.SendMsg(m)
}

func (x *forwardListenerControlStreamServer) Recv() (*clientpb.JobCtrl, error) {
	m := new(clientpb.JobCtrl)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

type ForwardListener_TaskStreamServer interface {
	Send(*clientpb.SpiteRequest) error
	Recv() (*clientpb.SpiteRequest, error)
	grpc.ServerStream
}

type forwardListenerTaskStreamServer struct {
	grpc.ServerStream
}

func (x *forwardListenerTaskStreamServer) Send(m *clientpb.SpiteRequest) error {
	return x.ServerStream.SendMsg(m)
}

func (x *forwardListenerTaskStreamServer) Recv() (*clientpb.SpiteRequest, error) {
	m := new(clientpb.SpiteRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _ForwardListener_ControlStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ForwardListenerServer).ControlStream(&forwardListenerControlStreamServer{ServerStream: stream})
}

func _ForwardListener_TaskStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ForwardListenerServer).TaskStream(&forwardListenerTaskStreamServer{ServerStream: stream})
}

var ForwardListener_ServiceDesc = grpc.ServiceDesc{
	ServiceName: ServiceName,
	HandlerType: (*ForwardListenerServer)(nil),
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "ControlStream",
			Handler:       _ForwardListener_ControlStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "TaskStream",
			Handler:       _ForwardListener_TaskStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "forwardrpc/forward_listener.proto",
}
