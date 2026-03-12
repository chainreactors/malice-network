package listener

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/IoM-go/types"
	"net"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func NewBindPipeline(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*BindPipeline, error) {
	pp := &BindPipeline{
		rpc:            rpc,
		Name:           pipeline.Name,
		Enable:         pipeline.Enable,
		CertName:       pipeline.CertName,
		PipelineConfig: core.FromPipeline(pipeline),
	}
	return pp, nil
}

type BindPipeline struct {
	rpc      listenerrpc.ListenerRPCClient
	Name     string
	Enable   bool
	CertName string
	*core.PipelineConfig
}

func (pipeline *BindPipeline) ID() string {
	return pipeline.Name
}

func (pipeline *BindPipeline) ToProtobuf() *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     pipeline.Enable,
		Type:       consts.BindPipeline,
		ListenerId: pipeline.ListenerID,
		CertName:   pipeline.CertName,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{
				Name:       pipeline.Name,
				ListenerId: pipeline.ListenerID,
			},
		},
		Tls:        pipeline.TLSConfig.ToProtobuf(),
		Encryption: pipeline.Encryption.ToProtobuf(),
		Secure:     pipeline.SecureConfig.ToProtobuf(),
	}
}

func (pipeline *BindPipeline) Start() error {
	if !pipeline.Enable {
		return nil
	}
	forward, err := core.NewForward(pipeline.rpc, pipeline)
	forward.ListenerId = pipeline.ListenerID
	if err != nil {
		return err
	}
	core.Forwarders.Add(forward)

	logs.Log.Infof("[pipeline] starting TCP Bind pipeline")
	core.SafeGo(func() { pipeline.handler() })

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
		core.SafeGo(func() {
			err = pipeline.handlerReq(msg)
			if err != nil {
				logs.Log.Errorf("[pipeline] %s", err.Error())
			}
		})
	}
}

func (pipeline *BindPipeline) handlerReq(req *clientpb.SpiteRequest) error {
	conn, err := net.Dial("tcp", req.Session.Target)
	if err != nil {
		return err
	}

	// Bind mode: Use WrapBindConn which doesn't pre-read
	// Server needs to send data first, then receive response
	peekConn, err := pipeline.WrapBindConn(conn)
	if err != nil {
		logs.Log.Errorf("wrap bind conn error: %v", err)
		conn.Close()
		return err
	}
	defer peekConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var connect *core.Connection
	if req.Spite.Name == consts.ModuleInit {
		err := peekConn.Parser.WritePacket(peekConn, types.BuildOneSpites(req.Spite), req.Session.RawId)
		if err != nil {
			logs.Log.Errorf("failed to send init packet: %v", err)
			return err
		}
		keyPair := core.GetKeyPairForSession(req.Session.RawId, pipeline.SecureConfig)
		connect = core.NewConnection(peekConn.Parser, req.Session.RawId, pipeline.Name, keyPair)
		core.Connections.Add(connect)
	} else {
		connect, err = core.GetConnection(peekConn, pipeline.Name, pipeline.SecureConfig)
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
