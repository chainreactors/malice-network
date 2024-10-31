package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"net"
)

func NewTcpPipeline(conn *grpc.ClientConn, pipeline *clientpb.Pipeline) (*TCPPipeline, error) {
	tcp := pipeline.GetTcp()

	pp := &TCPPipeline{
		grpcConn:       conn,
		Name:           pipeline.Name,
		Port:           uint16(tcp.Port),
		Host:           tcp.Host,
		Enable:         true,
		PipelineConfig: core.FromProtobuf(pipeline),
	}
	return pp, nil
}

type TCPPipeline struct {
	ln       net.Listener
	grpcConn *grpc.ClientConn
	Name     string
	Port     uint16
	Host     string
	Enable   bool
	*core.PipelineConfig
}

func (pipeline *TCPPipeline) ToProtobuf() *clientpb.Pipeline {
	p := &clientpb.Pipeline{
		Name:       pipeline.Name,
		Enable:     pipeline.Enable,
		ListenerId: pipeline.ListenerID,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Port: uint32(pipeline.Port),
				Host: pipeline.Host,
			},
		},
		Tls:        pipeline.Tls.ToProtobuf(),
		Encryption: pipeline.Encryption.ToProtobuf(),
	}
	return p
}

func (pipeline *TCPPipeline) ID() string {
	return pipeline.Name
}

func (pipeline *TCPPipeline) Close() error {
	err := pipeline.ln.Close()
	if err != nil {
		return err
	}
	return nil
}

func (pipeline *TCPPipeline) Start() error {
	if !pipeline.Enable {
		return nil
	}
	forward, err := core.NewForward(pipeline.grpcConn, pipeline)
	if err != nil {
		return err
	}
	core.Forwarders.Add(forward)
	go func() {
		// recv message from server and send to implant
		for {
			forward := core.Forwarders.Get(pipeline.ID())
			msg, err := forward.Stream.Recv()
			if err != nil {
				return
			}
			connect := core.Connections.Get(msg.Session.SessionId)
			if connect == nil {
				logs.Log.Errorf("connection %s not found", msg.Session.SessionId)
				continue
			}
			connect.C <- msg
		}
	}()

	pipeline.ln, err = pipeline.handler()
	if err != nil {
		return err
	}
	logs.Log.Infof("[pipeline] starting TCP pipeline on %s:%d", pipeline.Host, pipeline.Port)

	return nil
}

func (pipeline *TCPPipeline) handler() (net.Listener, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", pipeline.Host, pipeline.Port))
	if err != nil {
		return nil, err
	}
	if pipeline.Tls != nil && pipeline.Tls.Enable {
		ln, err = certutils.WrapWithTls(ln, pipeline.Tls)
		if err != nil {
			return nil, err
		}
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
					break
				}
				logs.Log.Errorf("Accept failed: %v", err)
				continue
			}
			logs.Log.Debugf("accept from %s", conn.RemoteAddr())
			go pipeline.handleAccept(conn)
		}
	}()
	return ln, nil
}

func (pipeline *TCPPipeline) handleAccept(conn net.Conn) {
	defer conn.Close()
	peekConn, err := pipeline.WrapConn(conn)
	if err != nil {
		logs.Log.Debugf("wrap conn error: %s %v", conn.RemoteAddr(), err)
		return
	}
	connect, err := core.Connections.NeedConnection(peekConn)
	if err != nil {
		logs.Log.Debugf("peek read header error: %s %v", conn.RemoteAddr(), err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		err := connect.Handler(ctx, peekConn)
		if err != nil {
			logs.Log.Debugf("handler error: %s", err.Error())
			return
		}
	}
}
