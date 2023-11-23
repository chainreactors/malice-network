package common

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"google.golang.org/protobuf/proto"
	"net"
	"sync"
	"time"
)

func NewImplant(addr string, sid []byte) *Implant {
	i := &Implant{
		Addr:     addr,
		Sid:      sid,
		interval: 10 * time.Second,
	}
	err := i.Connect()
	if err != nil {
		panic(err)
	}
	return i
}

type Implant struct {
	Addr     string
	conn     net.Conn
	Sid      []byte
	ch       chan *commonpb.Spites
	cache    types.SpitesCache
	interval time.Duration
}

func (implant *Implant) Connect() error {
	conn, err := net.Dial("tcp", implant.Addr)
	if err != nil {
		return err
	}
	implant.conn = conn
	return nil
}

func (implant *Implant) Register() {
	spite := &commonpb.Spite{
		TaskId: 1,
	}
	body := &commonpb.Register{
		Os: &commonpb.Os{
			Name: "windows",
		},
		Process: &commonpb.Process{
			Name: "test",
			Pid:  123,
			Uid:  "admin",
			Gid:  "root",
		},
		Timer: &commonpb.Timer{
			Interval: 10,
		},
	}
	types.BuildSpite(spite, body)
	implant.WriteSpite(spite)
}

// request spites
func (implant *Implant) Write(msg proto.Message) error {
	err := packet.WritePacket(implant.conn, msg, implant.Sid)
	if err != nil {
		return err
	}
	return nil
}

func (implant *Implant) WriteWithTimeout(msg proto.Message) error {
	err := packet.WritePacketWithTimeout(implant.conn, msg, implant.Sid, implant.interval)
	if err != nil {
		return err
	}
	return nil
}

func (implant *Implant) WriteSpite(msg proto.Message) error {
	spites := &commonpb.Spites{
		Spites: []*commonpb.Spite{msg.(*commonpb.Spite)},
	}

	return implant.Write(spites)
}

func (implant *Implant) Read() (proto.Message, error) {
	_, msg, err := packet.ReadPacket(implant.conn)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (implant *Implant) ReadWithTimeout() (proto.Message, error) {
	_, msg, err := packet.ReadPacketWithTimeout(implant.conn, implant.interval)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (implant *Implant) Close() {
	implant.conn.Close()
}

func (implant *Implant) Run() {
	implant.Register()
	for {
		var wg sync.WaitGroup
		wg.Add(2)
		implant.Connect()
		done := make(chan bool) // 创建一个缓冲通道
		go func() {
			defer wg.Done()
			msg, err := implant.ReadWithTimeout()
			if err != nil {
				done <- true
				logs.Log.Error(err.Error())
				return
			}
			implant.Handler(msg.(*commonpb.Spites))
			done <- true
			return
		}()

		go func() {
			defer wg.Done()
			select {
			case msg := <-implant.ch:
				err := implant.Write(msg)
				if err != nil {
					logs.Log.Error(err.Error())
					return
				}
				return
			case <-done:
				logs.Log.Error("auto close")
				return
			}
		}()
		wg.Wait()
	}
}

func (implant *Implant) Handler(msg *commonpb.Spites) {
	for _, spite := range msg.Spites {
		implant.cache.Append(implant.HandlerSpite(spite))
	}
	implant.ch <- implant.cache.Build()
}

func (implant *Implant) HandlerSpite(msg *commonpb.Spite) *commonpb.Spite {
	spite := &commonpb.Spite{
		TaskId: msg.TaskId,
		End:    true,
	}
	var resp proto.Message
	switch msg.Body.(type) {
	case *commonpb.Spite_ExecRequest:
		resp = &pluginpb.ExecResponse{
			Stdout:     []byte("admin"),
			Pid:        999,
			StatusCode: 0,
		}
	}
	types.BuildSpite(spite, resp)
	return spite
}
