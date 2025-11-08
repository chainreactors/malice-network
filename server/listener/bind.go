package listener

import (
	"context"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/IoM-go/types"
	"net"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

func NewBindPipeline(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*BindPipeline, error) {
	pp := &BindPipeline{
		rpc:            rpc,
		Name:           pipeline.Name,
		Enable:         true,
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
		connect, err = pipeline.getConnection(req.Session.RawId)
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

func (pipeline *BindPipeline) initConnection(conn *cryptostream.Conn, req *clientpb.SpiteRequest) (*core.Connection, error) {
	p := &parser.MessageParser{
		Implant:      consts.ImplantMalefic,
		PacketParser: &malefic.MaleficParser{},
	}
	go p.WritePacket(conn, types.BuildOneSpites(req.Spite), req.Session.RawId)

	keyPair := core.GetKeyPairForSession(req.Session.RawId, pipeline.SecureConfig)

	connect := core.NewConnection(p, req.Session.RawId, pipeline.ID(), keyPair)
	core.Connections.Add(connect)
	return connect, nil
}

// getConnection Bind pipeline 特殊的连接获取实现
// Bind pipeline 不使用 SecureConfig 交换密钥对，只从 session 获取
func (pipeline *BindPipeline) getConnection(sid uint32) (*core.Connection, error) {
	sessionID := hash.Md5Hash(encoders.Uint32ToBytes(sid))

	// 尝试从现有连接池获取连接
	if existingConn := core.Connections.Get(sessionID); existingConn != nil {
		return existingConn, nil
	}

	// 创建新的 parser
	p, err := parser.NewParser(pipeline.Parser)
	if err != nil {
		return nil, err
	}

	keyPair := core.GetKeyPairForSession(sid, pipeline.SecureConfig)

	// 创建新连接
	newConn := core.NewConnection(p, sid, pipeline.ID(), keyPair)
	core.Connections.Add(newConn)
	return newConn, nil
}
