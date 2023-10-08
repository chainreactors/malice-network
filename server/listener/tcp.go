package listener

import (
	"encoding/binary"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/core"
	"net"
)

type TCPListener struct {
	Port     uint16 `config:"port"`
	Host     string `config:"host"`
	Enable   bool   `config:"enable"`
	Protocol string `config:"protocol"`
}

func (l *TCPListener) Name() string {
	return l.Protocol
}
func (l *TCPListener) Start() (*core.Job, error) {
	if !l.Enable {
		return nil, nil
	}
	ln, err := StartTCPListener(l.Host, l.Port, nil)
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

// StartTCPListener - Start a TCP listener
func StartTCPListener(bindIface string, port uint16, data []byte) (net.Listener, error) {
	logs.Log.Infof("Starting TCP listener on %s:%d", bindIface, port)
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", bindIface, port))
	if err != nil {
		logs.Log.Error(err.Error())
		return nil, err
	}
	go acceptConnections(ln, data)
	return ln, nil
}

func acceptConnections(ln net.Listener, data []byte) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if errType, ok := err.(*net.OpError); ok && errType.Op == "accept" {
				break
			}
			logs.Log.Errorf("Accept failed: %v", err)
			continue
		}
		go handleConnection(conn, data)
	}
}

func handleConnection(conn net.Conn, data []byte) {
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
