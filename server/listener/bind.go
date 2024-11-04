package listener

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
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
		go func() {
			err = pipeline.handlerReq(msg)
			if err != nil {
				logs.Log.Errorf("[pipeline] %s", err.Error())
			}
		}()
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
	if req.Spite.Name == consts.ModuleInit {
		connect, err = pipeline.initConnection(peekConn, req)
		if err != nil {
			return err
		}
	} else {
		connect, err = pipeline.getConnection(peekConn, req.Session.RawId)
		if err != nil {
			return err
		}
		connect.C <- req
	}

	err = connect.Handler(ctx, peekConn)
	if err != nil {
		return err
	}
	return nil
}

func (pipeline *TCPBindPipeline) newConnection(p *parser.MessageParser, sid []byte, pipelineID string) *core.Connection {
	go func() {

	}()
	return core.NewConnection(p, sid, pipelineID)
}

func (pipeline *TCPBindPipeline) initConnection(conn *peek.Conn, req *clientpb.SpiteRequest) (*core.Connection, error) {
	p := &parser.MessageParser{
		Implant:      consts.ImplantMalefic,
		PacketParser: &malefic.MaleficParser{},
	}
	go p.WritePacket(conn, types.BuildOneSpites(req.Spite), req.Session.RawId)
	connect := core.NewConnection(p, req.Session.RawId, pipeline.ID())
	core.Connections.Add(connect)
	return connect, nil
}

func (pipeline *TCPBindPipeline) getConnection(conn *peek.Conn, sid []byte) (*core.Connection, error) {
	p, err := parser.NewParser(conn)
	if err != nil {
		return nil, err
	}
	_, _, err = p.PeekHeader(conn)
	if err != nil {
		return nil, err
	}
	if newC := core.Connections.Get(hash.Md5Hash(sid)); newC != nil {
		return newC, nil
	} else {
		newC := core.NewConnection(p, sid, pipeline.ID())
		core.Connections.Add(newC)
		return newC, nil
	}
}
