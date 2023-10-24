package pipeline

import (
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/server/configs"
	"github.com/chainreactors/malice-network/server/core"
	"google.golang.org/protobuf/proto"
	"net"
)

type TCPPipeline struct {
	done      chan bool
	forwarder *core.Forward
	Name      string `config:"id"`
	Port      uint16 `config:"port"`
	Host      string `config:"host"`
	Enable    bool   `config:"enable"`
	Protocol  string `config:"protocol"`
}

func (l *TCPPipeline) ID() string {
	return fmt.Sprintf("%s_%s_%s_%d", l.Name, l.Protocol, l.Host, l.Port)
}

func (l *TCPPipeline) Start() (*core.Job, error) {
	if !l.Enable {
		return nil, nil
	}
	ln, err := l.handler()
	if err != nil {
		return nil, err
	}

	job := &core.Job{
		ID:          core.NextJobID(),
		Name:        "TCP",
		Description: "Raw TCP listener (stager only)",
		Protocol:    "tcp",
		Host:        l.Host,
		Port:        l.Port,
		JobCtrl:     make(chan bool),
	}

	go func() {
		<-job.JobCtrl
		logs.Log.Infof("Stopping TCP listener (%d) ...", job.ID)
		ln.Close() // Kills listener GoRoutines in startMutualTLSListener() but NOT connections

		core.Jobs.Remove(job)

		//core.EventBroker.Publish(core.Event{
		//	Job:       job,
		//	EventType: consts.JobStoppedEvent,
		//})
	}()
	l.forwarder, err = core.NewForward(configs.GetServerConfig().String(), l)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (l *TCPPipeline) handler() (net.Listener, error) {
	logs.Log.Infof("Starting TCP listener on %s:%d", l.Host, l.Port)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", l.Host, l.Port))
	if err != nil {
		return nil, err
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
			go l.handleRead(conn)
		}
	}()
	return ln, nil
}

func (l *TCPPipeline) handleRead(conn net.Conn) {
	defer func() {
		l.done <- true
	}()
	var err error
	var connect *core.Connection
	for {
		var rawID string
		var msg proto.Message
		if connect == nil {
			var length int
			rawID, length, err = packet.ReadHeader(conn)
			if err != nil {
				logs.Log.Errorf("Error reading header: %v", err)
				return
			}
			sid := hash.Md5Hash([]byte(rawID))
			connect = core.Connections.Get(sid)
			if connect == nil {
				connect = core.NewConnection(rawID)
			}
			go l.handleWrite(conn, connect)
			msg, err = packet.ReadMessage(conn, length)
			if err != nil {
				core.Connections.Remove(sid)
				logs.Log.Errorf("Error reading message: %v", err)
				return
			}
		} else {
			rawID, msg, err = packet.ReadPacket(conn)
			if err != nil {
				core.Connections.Remove(hash.Md5Hash([]byte(rawID)))
				logs.Log.Errorf("Error reading packet: %v", err)
				return
			}
		}
		l.forwarder.Add(&core.Message{
			Message:   msg,
			SessionID: hash.Md5Hash([]byte(rawID)),
			//RemoteAddr: conn.RemoteAddr().String(),
		})
	}

}

func (l *TCPPipeline) handleWrite(conn net.Conn, connect *core.Connection) {
	msg := &commonpb.Spites{Spites: []*commonpb.Spite{}}

	for {
		select {
		case spite := <-connect.Sender:
			msg.Spites = append(msg.Spites, spite.(*commonpb.Spite))
		case <-l.done:
			return
		default:
			if len(msg.Spites) > 0 {
				err := packet.WritePacket(conn, msg, connect.RawID)
				if err != nil {
					logs.Log.Errorf("Error writing packet: %v", err)
					return
				}
				msg.Spites = []*commonpb.Spite{}
			}
		}
	}

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
