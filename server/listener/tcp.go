package listener

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/IoM-go/types"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

func NewTcpPipeline(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*TCPPipeline, error) {
	tcp := pipeline.GetTcp()

	pp := &TCPPipeline{
		rpc:            rpc,
		Name:           pipeline.Name,
		Port:           uint16(tcp.Port),
		Host:           tcp.Host,
		PipelineConfig: core.FromPipeline(pipeline),
		CertName:       pipeline.CertName,
	}

	return pp, nil
}

type TCPPipeline struct {
	ln       net.Listener
	rpc      listenerrpc.ListenerRPCClient
	Name     string
	Port     uint16
	Host     string
	Enable   bool
	Target   []string
	CertName string
	parser   *parser.MessageParser
	*core.PipelineConfig
}

func (pipeline *TCPPipeline) ToProtobuf() *clientpb.Pipeline {
	p := &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     pipeline.Enable,
		Type:       consts.TCPPipeline,
		ListenerId: pipeline.ListenerID,
		Parser:     pipeline.Parser,
		CertName:   pipeline.CertName,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       pipeline.Name,
				ListenerId: pipeline.ListenerID,
				Port:       uint32(pipeline.Port),
				Host:       pipeline.Host,
			},
		},
		Tls:        pipeline.TLSConfig.ToProtobuf(),
		Encryption: pipeline.Encryption.ToProtobuf(),
		Secure:     pipeline.SecureConfig.ToProtobuf(),
	}
	return p
}

func (pipeline *TCPPipeline) ID() string {
	return pipeline.Name
}

func (pipeline *TCPPipeline) Close() error {
	pipeline.Enable = false
	if pipeline.ln == nil {
		return nil
	}
	err := pipeline.ln.Close()
	if err != nil {
		return err
	}
	return nil
}

func (pipeline *TCPPipeline) Start() error {
	if pipeline.Enable {
		return nil
	}
	forward, err := core.NewForward(pipeline.rpc, pipeline)
	if err != nil {
		return err
	}
	forward.ListenerId = pipeline.ListenerID
	core.Forwarders.Add(forward)
	core.GoGuarded("tcp-forward-recv:"+pipeline.Name, func() error {
		for {
			msg, err := forward.Stream.Recv()
			if err != nil {
				if !pipeline.Enable {
					return nil
				}
				return fmt.Errorf("tcp pipeline %s forward recv: %w", pipeline.Name, err)
			}
			connect := core.Connections.Get(msg.Session.SessionId)
			if connect == nil {
				logs.Log.Errorf("connection %s not found", msg.Session.SessionId)
				continue
			}
			connect.C <- msg
		}
	}, pipeline.runtimeErrorHandler("forward recv loop"))

	pipeline.ln, err = pipeline.handler()
	if err != nil {
		return err
	}
	logs.Log.Infof("[pipeline] starting TCP pipeline on %s:%d, parser: %s, tls: %t",
		pipeline.Host, pipeline.Port, pipeline.Parser, pipeline.TLSConfig.Enable)
	pipeline.Enable = true
	return nil
}

func (pipeline *TCPPipeline) handler() (net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", pipeline.Host, pipeline.Port))
	if err != nil {
		return nil, err
	}

	// 如果启用了 TLS，使用 cmux 实现 TLS 和非 TLS 的端口复用
	if pipeline.TLSConfig != nil && pipeline.TLSConfig.Enable {
		return pipeline.handleWithCmux(ln)
	}

	// 非 TLS 模式，使用原有逻辑
	core.GoGuarded("tcp-accept:"+pipeline.Name, func() error {
		return pipeline.startAcceptLoop(ln, "tcp pipeline")
	}, pipeline.runtimeErrorHandler("accept loop"))
	return ln, nil
}

// handleWithCmux 使用 cmux 实现 TLS 和非 TLS 的端口复用
func (pipeline *TCPPipeline) handleWithCmux(ln net.Listener) (net.Listener, error) {
	var tlsConfig *tls.Config
	if pipeline.TLSConfig.Cert != nil {
		var err error
		tlsConfig, err = certutils.GetTlsConfig(pipeline.TLSConfig.Cert)
		if err != nil {
			return nil, err
		}
	}

	return StartCmuxTCPListener(ln, tlsConfig, pipeline.HandleConnection, pipeline.runtimeErrorHandler("cmux"))
}

// startAcceptLoop 启动连接接受循环 (用于非 cmux 模式)
func (pipeline *TCPPipeline) startAcceptLoop(ln net.Listener, logPrefix string) error {
	defer logs.Log.Debugf("%s exit", logPrefix)
	for {
		conn, err := ln.Accept()
		if err != nil {
			if !pipeline.Enable || errors.Is(err, net.ErrClosed) {
				logs.Log.Importantf("%s already disable, break accept", ln.Addr().String())
				return nil
			}
			return fmt.Errorf("tcp pipeline %s accept failed: %w", pipeline.Name, err)
		}
		core.GoGuarded("tcp-conn:"+pipeline.Name, func() error {
			pipeline.HandleConnection(conn)
			return nil
		}, core.LogGuardedError("tcp-conn:"+pipeline.Name))
	}
}

// HandleConnection 处理单个连接
func (pipeline *TCPPipeline) HandleConnection(conn net.Conn) {
	defer conn.Close()
	peekConn, err := pipeline.WrapConn(conn)
	if err != nil {
		logs.Log.Errorf("%s wrap conn error: %v", pipeline.Name, err)
		return
	}

	logs.Log.Debugf("[pipeline.%s] accept from %s", pipeline.Name, conn.RemoteAddr())
	switch peekConn.Parser.Implant {
	case consts.ImplantMalefic:
		pipeline.handleBeacon(peekConn)
	case consts.ImplantPulse:
		pipeline.handlePulse(peekConn)
	default:
		logs.Log.Warnf("tcp pipeline %s unsupported implant from %s: %s",
			pipeline.Name, conn.RemoteAddr(), peekConn.Parser.Implant)
	}
}

func (pipeline *TCPPipeline) handlePulse(conn *cryptostream.Conn) {
	magic, artifactId, err := conn.Parser.ReadHeader(conn)
	if err != nil {
		logs.Log.Errorf("%s", err.Error())
		return
	}
	builder, err := pipeline.rpc.GetArtifact(context.Background(), &clientpb.Artifact{
		Id:       artifactId,
		Pipeline: pipeline.Name,
		Format:   consts.FormatRaw,
	})
	if err != nil {
		logs.Log.Errorf("not found artifact %d, %s", artifactId, err.Error())
		return
	} else {
		logs.Log.Infof("send artifact %d %s", builder.Id, builder.Name)
	}
	err = conn.Parser.WritePacket(conn, types.BuildOneSpites(&implantpb.Spite{
		Name: consts.ModuleInit,
		Body: &implantpb.Spite_Init{
			Init: &implantpb.Init{Data: builder.Bin},
		},
	}), magic)
	if err != nil {
		logs.Log.Errorf("%s", err.Error())
		return
	}
}

func (pipeline *TCPPipeline) handleBeacon(conn *cryptostream.Conn) {
	connect, err := core.GetConnection(conn, pipeline.ID(), pipeline.SecureConfig)
	if err != nil {
		logs.Log.Warnf("tcp pipeline %s peek read header error from %s: %v", pipeline.Name, conn.RemoteAddr(), err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		err = connect.Handler(ctx, conn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logs.Log.Warnf("tcp pipeline %s handler error from %s: %s", pipeline.Name, conn.RemoteAddr(), err.Error())
			}
			return
		}
	}
}

func (pipeline *TCPPipeline) runtimeErrorHandler(scope string) core.GoErrorHandler {
	label := fmt.Sprintf("tcp pipeline %s %s", pipeline.Name, scope)
	return core.CombineErrorHandlers(
		core.LogGuardedError(label),
		func(err error) {
			pipeline.Enable = false
			if pipeline.ln != nil {
				_ = pipeline.ln.Close()
			}
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
