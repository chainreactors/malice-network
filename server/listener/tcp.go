package listener

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	core2 "github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/listener/encryption"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"net"
)

func StartTcpPipeline(conn *grpc.ClientConn, cfg *configs.TcpPipelineConfig) (*TCPPipeline, error) {
	pp := &TCPPipeline{
		Name:      cfg.Name,
		Port:      cfg.Port,
		Host:      cfg.Host,
		Enable:    cfg.Enable,
		TlsConfig: cfg.TlsConfig,
	}
	err := pp.Start()
	if err != nil {
		return nil, err
	}
	forward, err := core2.NewForward(conn, pp)
	if err != nil {
		return nil, err
	}
	core2.Forwarders.Add(forward)
	return pp, nil
}

func ToTcpConfig(pipeline *lispb.TCPPipeline) *configs.TcpPipelineConfig {
	return &configs.TcpPipelineConfig{
		Name:   pipeline.Name,
		Port:   uint16(pipeline.Port),
		Host:   pipeline.Host,
		Enable: true,
	}
}

type TCPPipeline struct {
	ln        net.Listener
	Name      string
	Port      uint16
	Host      string
	Enable    bool
	TlsConfig *configs.TlsConfig
}

func (l *TCPPipeline) ToProtobuf() proto.Message {
	return &lispb.TCPPipeline{
		Name: l.Name,
		Port: uint32(l.Port),
		Host: l.Host,
	}
}

func (l *TCPPipeline) ID() string {
	return fmt.Sprintf("tcp_%s_%s_%d", l.Name, l.Host, l.Port)
}

func (l *TCPPipeline) Addr() string {
	return ""
}

func (l *TCPPipeline) Close() error {
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
	logs.Log.Infof("Starting TCP listener on %s:%d", l.Host, l.Port)
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
			logs.Log.Infof("accept from %s", conn.RemoteAddr())
			go l.handleRead(conn)
		}
	}()
	return ln, nil
}

func (l *TCPPipeline) handleRead(conn net.Conn) {
	defer conn.Close()
	var err error
	var connect *core2.Connection
	var rawID []byte
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		var msg proto.Message
		var length int
		rawID, length, err = packet.ReadHeader(conn)
		if err != nil {
			logs.Log.Debugf("Error reading header: %s %v", conn.RemoteAddr(), err)
			return
		}
		sid := hash.Md5Hash(rawID)
		connect = core2.Connections.Get(sid)
		if connect == nil {
			connect = core2.NewConnection(rawID)
		}

		go connect.Send(ctx, conn)
		msg, err = packet.ReadMessage(conn, length)
		if err != nil {
			logs.Log.Debugf("Error reading message:%s %v", conn.RemoteAddr(), err)
			return
		}
		core2.Forwarders.Send(l.ID(), &core2.Message{
			Message:   msg,
			SessionID: hash.Md5Hash(rawID),
			//RemoteAddr: conn.RemoteAddr().String(),
		})
	}

}
func (l *TCPPipeline) handleWrite(conn net.Conn, ch chan *commonpb.Spites, rawid []byte) {
	msg := <-ch
	err := packet.WritePacket(conn, msg, rawid)
	if err != nil {
		logs.Log.Debugf(err.Error())
		ch <- msg
	}
	return
}

func handleShellcode(conn net.Conn, data []byte) {
	logs.Log.Infof("Accepted incoming connection: %s", conn.RemoteAddr())
	// Send shellcode size
	dataSize := uint32(len(data))
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, dataSize)
	logs.Log.Infof("Shellcode size: %d\n", dataSize)
	final := append(lenBuf, data...)
	logs.Log.Infof("Sending shellcode (%d)\n", len(final))
	// Send shellcode
	conn.Write(final)
	// Closing connection
	conn.Close()
}
