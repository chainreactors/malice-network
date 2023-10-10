package listener

import (
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/core"
	"github.com/chainreactors/malice-network/utils/packet"
	"net"
)

type TCPListener struct {
	Name     string `config:"id"`
	Port     uint16 `config:"port"`
	Host     string `config:"host"`
	Enable   bool   `config:"enable"`
	Protocol string `config:"protocol"`
}

func (l *TCPListener) ID() string {
	return fmt.Sprintf("%s_%s_%s_%d", l.Name, l.Protocol, l.Host, l.Port)
}

func (l *TCPListener) Start() (*core.Job, error) {
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

	return job, nil
}

func (l *TCPListener) handler() (net.Listener, error) {
	logs.Log.Infof("Starting TCP listener on %s:%d", l.Host, l.Port)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", l.Host, l.Port))
	if err != nil {
		logs.Log.Error(err.Error())
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
			go l.handleConnection(conn)
		}
	}()
	return ln, nil
}

func (l *TCPListener) handleConnection(conn net.Conn) {
	//done := make(chan bool)
	//defer func() {
	//	done <- true
	//}()

	for {
		msg, err := packet.ReadMessage(conn)
		if err != nil {
			logs.Log.Errorf("Error reading packet: %v", err)
			return
		}
		core.Forwarders.Get(l.ID()).Add(&core.Message{
			Message:    msg,
			RemoteAddr: conn.RemoteAddr().String(),
		})
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
