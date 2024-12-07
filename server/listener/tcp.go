package listener

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/peek"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/parser"
	"io"
	"net"
)

func NewTcpPipeline(rpc listenerrpc.ListenerRPCClient, pipeline *clientpb.Pipeline) (*TCPPipeline, error) {
	tcp := pipeline.GetTcp()

	pp := &TCPPipeline{
		rpc:            rpc,
		Name:           pipeline.Name,
		Port:           uint16(tcp.Port),
		Host:           tcp.Host,
		Enable:         true,
		PipelineConfig: core.FromProtobuf(pipeline),
	}
	var err error
	pp.parser, err = parser.NewParser(pp.Parser)
	if err != nil {
		return nil, err
	}

	return pp, nil
}

type TCPPipeline struct {
	ln     net.Listener
	rpc    listenerrpc.ListenerRPCClient
	Name   string
	Port   uint16
	Host   string
	Enable bool
	parser *parser.MessageParser
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
	pipeline.Enable = false
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
	forward, err := core.NewForward(pipeline.rpc, pipeline)
	if err != nil {
		return err
	}
	forward.ListenerId = pipeline.ListenerID
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
	logs.Log.Infof("[pipeline] starting TCP pipeline on %s:%d, parser: %s, cryptor: %s, tls: %t",
		pipeline.Host, pipeline.Port, pipeline.Parser, pipeline.Encryption.Type, pipeline.Tls.Enable)

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
				if !pipeline.Enable {
					logs.Log.Importantf("%s already disable, break accept", ln.Addr().String())
					return
				} else {
					continue
				}
			}
			logs.Log.Debugf("[pipeline.%s] accept from %s", pipeline.Name, conn.RemoteAddr())
			switch pipeline.Parser {
			case consts.ImplantMalefic:
				go pipeline.handleBeacon(conn)
			case consts.ImplantPulse:
				go pipeline.handlePulse(conn)
			}

		}
	}()
	return ln, nil
}

func (pipeline *TCPPipeline) handlePulse(conn net.Conn) {
	peekConn, err := pipeline.WrapConn(conn)
	if err != nil {
		logs.Log.Debugf("wrap conn error: %s %v", conn.RemoteAddr(), err)
		return
	}
	p := pipeline.parser
	magic, artifactId, err := p.ReadHeader(peekConn)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
	builder, err := pipeline.rpc.GetArtifact(context.Background(), &clientpb.Artifact{
		Id: uint32(artifactId),
	})
	if err != nil {
		logs.Log.Errorf("not found artifact %d ,%s ", artifactId, err.Error())
		return
	} else {
		logs.Log.Infof("send artifact %d %s", builder.Id, builder.Name)
	}
	err = p.WritePacket(peekConn, types.BuildOneSpites(&implantpb.Spite{
		Name: consts.ModuleInit,
		Body: &implantpb.Spite_Init{
			Init: &implantpb.Init{Data: builder.Bin},
		},
	}), magic)
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
}

func (pipeline *TCPPipeline) handleBeacon(conn net.Conn) {
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
		err = connect.Handler(ctx, peekConn)
		if err != nil {
			if !errors.Is(err, io.EOF) {
				logs.Log.Debugf("handler error: %s", err.Error())
			}
			return
		}
	}
}

func (pipeline *TCPPipeline) getConnection(conn *peek.Conn) (*core.Connection, error) {
	p := pipeline.parser
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
