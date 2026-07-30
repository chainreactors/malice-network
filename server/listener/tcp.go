package listener

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

var (
	tcpListen    = net.Listen
	tcpStartCmux = StartCmuxTCPListener
)

func NewTcpPipeline(rpc pipelineRPCClient, pipeline *clientpb.Pipeline) (*TCPPipeline, error) {
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
	stateMu   sync.RWMutex
	starting  bool
	startDone chan struct{}
	ln        net.Listener
	forward   *forwardSupervisor
	rpc       pipelineRPCClient
	Name      string
	Port      uint16
	Host      string
	Enable    bool
	Target    []string
	CertName  string
	parser    *parser.MessageParser
	*core.PipelineConfig
}

func (pipeline *TCPPipeline) ToProtobuf() *clientpb.Pipeline {
	pipeline.stateMu.RLock()
	enabled := pipeline.Enable
	pipeline.stateMu.RUnlock()
	p := &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     enabled,
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
	pipeline.stateMu.Lock()
	pipeline.Enable = false
	if pipeline.starting {
		done := pipeline.startDone
		pipeline.stateMu.Unlock()
		<-done
		return nil
	}
	ln := pipeline.ln
	pipeline.ln = nil
	forward := pipeline.forward
	pipeline.forward = nil
	pipeline.stateMu.Unlock()
	var forwardErr error
	if forward != nil {
		forwardErr = forward.Stop()
	}
	return errors.Join(forwardErr, closePipelineListener(ln))
}

func (pipeline *TCPPipeline) Start() (err error) {
	if !pipeline.beginStart() {
		return nil
	}

	forward, err := core.NewForward(pipeline.rpc, pipeline)
	if err != nil {
		pipeline.abortStart()
		return err
	}
	forward.ListenerId = pipeline.ListenerID
	committed := false
	registered := false
	defer func() {
		if !committed {
			if registered {
				err = errors.Join(err, pipeline.rollbackCommittedStart(forward))
			} else {
				err = errors.Join(err, forward.Abort())
			}
			pipeline.abortStart()
		}
	}()

	ln, tlsConfig, err := pipeline.prepareListener()
	if err != nil {
		return err
	}
	if !pipeline.commitStart(ln, forward) {
		return closePipelineListener(ln)
	}
	registered = true
	if err := pipeline.startServing(ln, tlsConfig); err != nil {
		return err
	}
	if !pipeline.finishStart() {
		return nil
	}
	committed = true
	pipeline.startForwardRecv(forward)
	logs.Log.Infof("pipeline.tcp - start host=%s port=%d parser=%s tls=%t",
		pipeline.Host, pipeline.Port, pipeline.Parser, pipeline.TLSConfig.Enable)
	return nil
}

func (pipeline *TCPPipeline) enabled() bool {
	pipeline.stateMu.RLock()
	defer pipeline.stateMu.RUnlock()
	return pipeline.Enable
}

func (pipeline *TCPPipeline) beginStart() bool {
	for {
		pipeline.stateMu.Lock()
		if pipeline.starting {
			done := pipeline.startDone
			pipeline.stateMu.Unlock()
			<-done
			continue
		}
		if pipeline.Enable {
			pipeline.stateMu.Unlock()
			return false
		}
		pipeline.Enable = true
		pipeline.starting = true
		pipeline.startDone = make(chan struct{})
		pipeline.stateMu.Unlock()
		return true
	}
}

func (pipeline *TCPPipeline) abortStart() {
	pipeline.stateMu.Lock()
	pipeline.starting = false
	pipeline.Enable = false
	done := pipeline.startDone
	pipeline.startDone = nil
	if done != nil {
		close(done)
	}
	pipeline.stateMu.Unlock()
}

func (pipeline *TCPPipeline) commitStart(ln net.Listener, forward *core.Forward) bool {
	pipeline.stateMu.Lock()
	defer pipeline.stateMu.Unlock()
	if !pipeline.Enable {
		return false
	}
	pipeline.ln = ln
	core.Forwarders.Add(forward)
	return true
}

func (pipeline *TCPPipeline) finishStart() bool {
	pipeline.stateMu.Lock()
	defer pipeline.stateMu.Unlock()
	if !pipeline.Enable {
		return false
	}
	pipeline.starting = false
	done := pipeline.startDone
	pipeline.startDone = nil
	close(done)
	return true
}

func (pipeline *TCPPipeline) startForwardRecv(forward *core.Forward) {
	supervisor := newForwardSupervisor(
		pipeline.rpc,
		pipeline,
		"tcp",
		forward,
		func(msg *clientpb.SpiteRequest) {
			dispatchForwardTaskRequest("tcp", pipeline.Name, msg)
		},
	)
	pipeline.stateMu.Lock()
	if !pipeline.Enable || pipeline.forward != nil {
		pipeline.stateMu.Unlock()
		_ = supervisor.Stop()
		return
	}
	pipeline.forward = supervisor
	pipeline.stateMu.Unlock()
	supervisor.Start()
}

func (pipeline *TCPPipeline) prepareListener() (net.Listener, *tls.Config, error) {
	ln, err := tcpListen("tcp", fmt.Sprintf("%s:%d", pipeline.Host, pipeline.Port))
	if err != nil {
		return nil, nil, err
	}

	var tlsConfig *tls.Config
	if pipeline.TLSConfig != nil && pipeline.TLSConfig.Enable && pipeline.TLSConfig.Cert != nil {
		if pipeline.TLSConfig.MTLS && pipeline.TLSConfig.CA != nil {
			tlsConfig, err = certutils.GetMTlsConfig(pipeline.TLSConfig.Cert, pipeline.TLSConfig.CA)
			logs.Log.Infof("pipeline.tcp - mtls_enabled pipeline=%s", pipeline.Name)
		} else {
			tlsConfig, err = certutils.GetTlsConfig(pipeline.TLSConfig.Cert)
		}
		if err != nil {
			return nil, nil, errors.Join(err, closePipelineListener(ln))
		}
	}
	return ln, tlsConfig, nil
}

func (pipeline *TCPPipeline) startServing(ln net.Listener, tlsConfig *tls.Config) error {
	if pipeline.TLSConfig != nil && pipeline.TLSConfig.Enable {
		_, err := tcpStartCmux(ln, tlsConfig, pipeline.HandleConnection, pipeline.runtimeErrorHandler("cmux"))
		return err
	}
	core.GoGuarded("tcp-accept:"+pipeline.Name, func() error {
		return pipeline.startAcceptLoop(ln, "tcp pipeline")
	}, pipeline.runtimeErrorHandler("accept loop"))
	return nil
}

func closePipelineListener(ln net.Listener) error {
	if ln == nil {
		return nil
	}
	err := ln.Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (pipeline *TCPPipeline) rollbackCommittedStart(forward *core.Forward) error {
	forwardErr := core.Forwarders.RemoveIfSame(forward.RuntimeKey(), forward)
	pipeline.stateMu.Lock()
	ln := pipeline.ln
	pipeline.ln = nil
	pipeline.Enable = false
	pipeline.stateMu.Unlock()
	return errors.Join(forwardErr, closePipelineListener(ln))
}

// startAcceptLoop 启动连接接受循环 (用于非 cmux 模式)
func (pipeline *TCPPipeline) startAcceptLoop(ln net.Listener, logPrefix string) error {
	defer logs.Log.Debugf("%s exit", logPrefix)
	for {
		conn, err := ln.Accept()
		if err != nil {
			if !pipeline.enabled() || errors.Is(err, net.ErrClosed) {
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

	logs.Log.Debugf("pipeline.tcp - accept pipeline=%s remote=%s", pipeline.Name, conn.RemoteAddr())
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
	connect, err := core.GetConnection(conn, core.PipelineRuntimeKey(pipeline.ListenerID, pipeline.ID()), pipeline.SecureConfig)
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
			_ = pipeline.Close()
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
