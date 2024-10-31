package listener

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"net"
)

func NewBindPipeline(conn *grpc.ClientConn, pipeline *clientpb.Pipeline) (*TCPBindPipeline, error) {
	pp := &TCPBindPipeline{
		grpcConn:       conn,
		Name:           pipeline.Name,
		Enable:         true,
		PipelineConfig: core.FromProtobuf(pipeline),
	}
	return pp, nil
}

type TCPBindPipeline struct {
	grpcConn *grpc.ClientConn
	Name     string
	//Port   uint16
	//Target string
	Enable bool
	*core.PipelineConfig
}

func (pipeline *TCPBindPipeline) ID() string {
	return pipeline.Name
}

func (pipeline *TCPBindPipeline) ToProtobuf() *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     pipeline.Enable,
		ListenerId: pipeline.ListenerID,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{},
		},
		Tls:        pipeline.Tls.ToProtobuf(),
		Encryption: pipeline.Encryption.ToProtobuf(),
	}
}

func (pipeline *TCPBindPipeline) Start() error {
	if !pipeline.Enable {
		return nil
	}
	forward, err := core.NewForward(pipeline.grpcConn, pipeline)
	if err != nil {
		return err
	}
	core.Forwarders.Add(forward)

	logs.Log.Infof("[pipeline] starting TCP Bind pipeline")
	go pipeline.handler()

	return nil
}

func (pipeline *TCPBindPipeline) Close() error {
	return nil
}

func (pipeline *TCPBindPipeline) handler() error {
	for {
		forward := core.Forwarders.Get(pipeline.ID())
		msg, err := forward.Stream.Recv()
		if err != nil {
			return err
		}
		err = pipeline.handlerReq(msg)
		if err != nil {
			logs.Log.Errorf("[pipeline] %s", err.Error())
			continue
		}
	}
}

func (pipeline *TCPBindPipeline) handlerReq(req *clientpb.SpiteRequest) error {
	conn, err := net.Dial("tcp", req.Session.Target)
	if err != nil {
		return err
	}
	peekConn, err := pipeline.WrapConn(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var connect *core.Connection
	if connect = core.Connections.Get(req.Session.SessionId); connect == nil {
		connect, err = core.Connections.NeedConnection(peekConn, pipeline.ID())
		if err != nil {
			return err
		}
	}

	connect.C <- req
	err = connect.Handler(ctx, peekConn)
	if err != nil {
		return err
	}
	return nil
}
