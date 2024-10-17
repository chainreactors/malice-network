package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/listener/encryption"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"net"
)

func StartTcpPipeline(conn *grpc.ClientConn, pipeline *lispb.Pipeline) (*TCPPipeline, error) {
	tcp := pipeline.GetTcp()
	pp := &TCPPipeline{
		Name:   tcp.Name,
		Port:   uint16(tcp.Port),
		Host:   tcp.Host,
		Enable: true,
		TlsConfig: &configs.CertConfig{
			Cert:   pipeline.GetTls().Cert,
			Key:    pipeline.GetTls().Key,
			Enable: pipeline.GetTls().Enable,
		},
		Encryption: &configs.EncryptionConfig{
			Enable: pipeline.GetEncryption().Enable,
			Type:   pipeline.GetEncryption().Type,
			Key:    pipeline.GetEncryption().Key,
		},
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

func ToTcpConfig(pipeline *lispb.TCPPipeline, tls *lispb.TLS) *configs.TcpPipelineConfig {
	return &configs.TcpPipelineConfig{
		Name:   pipeline.Name,
		Port:   uint16(pipeline.Port),
		Host:   pipeline.Host,
		Enable: true,
		TlsConfig: &configs.TlsConfig{
			Name:     fmt.Sprintf("%s_%v", pipeline.Name, uint16(pipeline.Port)),
			Enable:   true,
			CertFile: tls.Cert,
			KeyFile:  tls.Key,
		},
	}
}

type TCPPipeline struct {
	ln         net.Listener
	Name       string
	Port       uint16
	Host       string
	Enable     bool
	TlsConfig  *configs.CertConfig
	Encryption *configs.EncryptionConfig
}

func (l *TCPPipeline) ToProtobuf() proto.Message {
	return &lispb.TCPPipeline{
		Name: l.Name,
		Port: uint32(l.Port),
		Host: l.Host,
	}
}

func (l *TCPPipeline) ToTLSProtobuf() proto.Message {
	return &lispb.TLS{
		Cert: l.TlsConfig.Cert,
		Key:  l.TlsConfig.Key,
	}
}
func (l *TCPPipeline) ID() string {
	return fmt.Sprintf(l.Name)
}

func (l *TCPPipeline) Addr() string {
	return ""
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
		ln, err = encryption.WrapWithTls(ln, l.TlsConfig)
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
			go l.handleRead(l.wrapConn(conn))
		}
	}()
	return ln, nil
}

func (l *TCPPipeline) handleRead(conn net.Conn) {
	defer conn.Close()

	peekConn := peek.WrapPeekConn(conn)
	connect, err := core.Connections.NeedConnection(peekConn)
	if err != nil {
		logs.Log.Debugf("peek read header error: %s %v", conn.RemoteAddr(), err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		var err error
		var msg proto.Message
		_, length, err := connect.Parser.ReadHeader(peekConn)
		if err != nil {
			//logs.Log.Debugf("Error reading header: %s %v", conn.RemoteAddr(), err)
			return
		}

		go connect.Send(ctx, conn)
		if length != 1 {
			msg, err = connect.Parser.ReadMessage(peekConn, length)
			if err != nil {
				logs.Log.Debugf("Error reading message:%s %v", conn.RemoteAddr(), err)
				return
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

func (l *TCPPipeline) wrapConn(conn net.Conn) net.Conn {
	if l.Encryption != nil && l.Encryption.Enable {
		eConn, err := encryption.WrapWithEncryption(conn, []byte(l.Encryption.Key))
		if err != nil {
			return conn
		}
		return eConn
	}
	return conn
}
