package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser"
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
		defer logs.Log.Errorf("forwarder stream exit!!!")
		for {
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
		defer logs.Log.Errorf("tcp pipeline exit!!!")
		for {
			conn, err := ln.Accept()
			if err != nil {
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
	connect, err := pipeline.getConnection(peekConn)
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

func (pipeline *TCPPipeline) getConnection(conn *peek.Conn) (*core.Connection, error) {
	p, err := parser.NewParser(conn)
	if err != nil {
		return nil, err
	}
	sid, _, err := p.PeekHeader(conn)
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
