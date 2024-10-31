package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	cryptostream "github.com/chainreactors/malice-network/helper/cryptography/stream"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"net"
)

func StartTcpPipeline(conn *grpc.ClientConn, pipeline *clientpb.Pipeline) (*TCPPipeline, error) {
	tcp := pipeline.GetTcp()

	pp := &TCPPipeline{
		Name:           tcp.Name,
		Port:           uint16(tcp.Port),
		Host:           tcp.Host,
		Enable:         true,
		PipelineConfig: FromProtobuf(pipeline),
	}
	err := pp.Start()
	if err != nil {
		return nil, err
	}
	forward, err := core.NewForward(conn, pp)
	if err != nil {
		return nil, err
	}
	core.Forwarders.Add(forward)
	return pp, nil
}

type TCPPipeline struct {
	ln     net.Listener
	Name   string
	Port   uint16
	Host   string
	Enable bool
	*PipelineConfig
}

func (l *TCPPipeline) ToProtobuf() *clientpb.Pipeline {
	p := l.PipelineConfig.ToProtobuf()
	p.Body = &clientpb.Pipeline_Tcp{
		Tcp: &clientpb.TCPPipeline{
			Name: l.Name,
			Port: uint32(l.Port),
			Host: l.Host,
		},
	}
	return p
}

func (l *TCPPipeline) ID() string {
	return fmt.Sprintf(l.Name)
}

func (l *TCPPipeline) Close() error {
	err := l.ln.Close()
	if err != nil {
		return err
	}
	return nil
}

func (l *TCPPipeline) Start() error {
	if !l.Enable {
		return nil
	}
	var err error
	l.ln, err = l.handler()
	if err != nil {
		return err
	}

	return nil
}

func (l *TCPPipeline) handler() (net.Listener, error) {
	logs.Log.Infof("[pipeline] starting TCP pipeline on %s:%d", l.Host, l.Port)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", l.Host, l.Port))
	if err != nil {
		return nil, err
	}
	if l.TlsConfig != nil && l.TlsConfig.Enable {
		ln, err = certutils.WrapWithTls(ln, l.TlsConfig)
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
			go l.handleRead(conn)
		}
	}()
	return ln, nil
}

func (l *TCPPipeline) handleRead(conn net.Conn) {
	defer conn.Close()
	cry, err := l.Encryption.NewCrypto()
	if err != nil {
		logs.Log.Errorf("new crypto error: %v", err)
		return
	}
	conn = cryptostream.NewCryptoConn(conn, cry)
	peekConn := peek.WrapPeekConn(conn)
	connect, err := core.Connections.NeedConnection(peekConn)
	if err != nil {
		logs.Log.Debugf("peek read header error: %s %v", conn.RemoteAddr(), err)
		return
	}
	var msg *implantpb.Spites
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		var err error
		_, length, err := connect.Parser.ReadHeader(peekConn)
		if err != nil {
			//logs.Log.Debugf("Error reading header: %s %v", conn.RemoteAddr(), err)
			return
		}

		go connect.Send(ctx, peekConn)
		if length != 1 {
			msg, err = connect.Parser.ReadMessage(peekConn, length)
			if err != nil {
				logs.Log.Debugf("Error reading message:%s %v", conn.RemoteAddr(), err)
				return
			}
			if msg.Spites == nil {
				msg = types.BuildPingSpite()
			}
		} else {
			msg = types.BuildPingSpite()
		}

		core.Forwarders.Send(l.ID(), &core.Message{
			Message:    msg,
			SessionID:  connect.SessionID,
			RemoteAddr: conn.RemoteAddr().String(),
		})
	}
}
