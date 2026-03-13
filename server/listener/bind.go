package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/IoM-go/types"
	"net"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
)

type bindRPCClient interface {
	SpiteStream(ctx context.Context, opts ...grpc.CallOption) (listenerrpc.ListenerRPC_SpiteStreamClient, error)
	Register(ctx context.Context, in *clientpb.RegisterSession, opts ...grpc.CallOption) (*clientpb.Empty, error)
	Checkin(ctx context.Context, in *implantpb.Ping, opts ...grpc.CallOption) (*clientpb.Empty, error)
}

func NewBindPipeline(rpc bindRPCClient, pipeline *clientpb.Pipeline) (*BindPipeline, error) {
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
	rpc      bindRPCClient
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
	if err != nil {
		return err
	}
	forward.ListenerId = pipeline.ListenerID
	core.Forwarders.Add(forward)

	logs.Log.Infof("[pipeline] starting TCP Bind pipeline")
	core.GoGuarded("bind-handler:"+pipeline.Name, pipeline.handler, pipeline.runtimeErrorHandler("handler loop"))

	return nil
}

func (pipeline *BindPipeline) Close() error {
	return nil
}

func (pipeline *BindPipeline) handler() error {
	defer logs.Log.Debugf("bind pipeline %s exit", pipeline.Name)
	for {
		forward := core.Forwarders.Get(pipeline.ID())
		if forward == nil {
			return fmt.Errorf("bind pipeline %s forwarder missing", pipeline.Name)
		}
		msg, err := forward.Stream.Recv()
		if err != nil {
			return fmt.Errorf("bind pipeline %s recv failed: %w", pipeline.Name, err)
		}
		core.GoGuarded("bind-request:"+pipeline.Name, func() error {
			return pipeline.handlerReq(msg)
		}, core.LogGuardedError("bind-request:"+pipeline.Name))
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
		if err := core.Connections.Push(connect.SessionID, req); err != nil {
			return fmt.Errorf("bind pipeline %s push: %w", pipeline.Name, err)
		}
	}

	err = connect.Handler(ctx, peekConn)
	if err != nil {
		return err
	}
	return nil
}

func (pipeline *BindPipeline) runtimeErrorHandler(scope string) core.GoErrorHandler {
	label := fmt.Sprintf("bind pipeline %s %s", pipeline.Name, scope)
	return core.CombineErrorHandlers(
		core.LogGuardedError(label),
		func(err error) {
			pipeline.Enable = false
			if core.EventBroker != nil {
				core.EventBroker.Publish(core.Event{
					EventType: consts.EventListener,
					Op:        consts.CtrlPipelineStop,
					Listener:  &clientpb.Listener{Id: pipeline.ListenerID},
					Message:   label,
					Err:       core.ErrorText(err),
					Important: true,
				})
			}
		},
	)
}
