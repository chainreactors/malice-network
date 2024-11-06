package listener

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
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

func NewBindPipeline(conn *grpc.ClientConn, pipeline *clientpb.Pipeline) (*BindPipeline, error) {
	pp := &BindPipeline{
		grpcConn:       conn,
		Name:           pipeline.Name,
		Enable:         true,
		PipelineConfig: core.FromProtobuf(pipeline),
	}
	return pp, nil
}

type BindPipeline struct {
	grpcConn *grpc.ClientConn
	Name     string
	//Port   uint16
	//Target string
	Enable bool
	*core.PipelineConfig
}

func (pipeline *BindPipeline) ID() string {
	return pipeline.Name
}

func (pipeline *BindPipeline) ToProtobuf() *clientpb.Pipeline {
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

func (pipeline *BindPipeline) Start() error {
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

func (pipeline *BindPipeline) Close() error {
	return nil
}

func (pipeline *BindPipeline) handler() error {
	defer logs.Log.Errorf("bind pipeline exit!!!")
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

func (pipeline *BindPipeline) handlerReq(req *clientpb.SpiteRequest) error {
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

func (pipeline *BindPipeline) initConnection(conn *peek.Conn, req *clientpb.SpiteRequest) (*core.Connection, error) {
	p := &parser.MessageParser{
		Implant:      consts.ImplantMalefic,
		PacketParser: &malefic.MaleficParser{},
	}
	go p.WritePacket(conn, types.BuildOneSpites(req.Spite), req.Session.RawId)
	connect := core.NewConnection(p, req.Session.RawId, pipeline.ID())
	core.Connections.Add(connect)
	return connect, nil
}

func (pipeline *BindPipeline) getConnection(conn *peek.Conn, sid uint32) (*core.Connection, error) {
	p, err := parser.NewParser(conn)
	if err != nil {
		return nil, err
	}
	_, _, err = p.PeekHeader(conn)
	if err != nil {
		return nil, err
	}
	if newC := core.Connections.Get(hash.Md5Hash(encoders.Uint32ToBytes(sid))); newC != nil {
		return newC, nil
	} else {
		newC := core.NewConnection(p, sid, pipeline.ID())
		core.Connections.Add(newC)
		return newC, nil
	}
}
