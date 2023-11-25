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
		ch:       make(chan *commonpb.Spites, 255),
		cache:    types.NewSpitesCache(),
	}
	return i
}

type Implant struct {
	Addr     string
	Sid      []byte
	ch       chan *commonpb.Spites
	cache    *types.SpitesCache
	interval time.Duration
}

func (implant *Implant) Connect() (net.Conn, error) {
	conn, err := net.Dial("tcp", implant.Addr)
	if err != nil {
		return conn, err
	}
	return conn, nil
}

func (implant *Implant) MustConnect() net.Conn {
	conn, err := net.Dial("tcp", implant.Addr)
	if err != nil {
		panic(err)
	}
	return conn
}

func (implant *Implant) Register() {
	conn := implant.MustConnect()
	defer conn.Close()
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
	implant.WriteSpite(conn, spite)
}

// request spites
func (implant *Implant) Write(conn net.Conn, msg proto.Message) error {
	err := packet.WritePacket(conn, msg, implant.Sid)
	if err != nil {
		return err
	}
	return nil
}

func (implant *Implant) WriteWithTimeout(conn net.Conn, msg proto.Message) error {
	err := packet.WritePacketWithTimeout(conn, msg, implant.Sid, implant.interval)
	if err != nil {
		return err
	}
	return nil
}

func (implant *Implant) WriteSpite(conn net.Conn, msg proto.Message) error {
	spites := &commonpb.Spites{
		Spites: []*commonpb.Spite{msg.(*commonpb.Spite)},
	}

	return implant.Write(conn, spites)
}

func (implant *Implant) Read(conn net.Conn) (proto.Message, error) {
	_, msg, err := packet.ReadPacket(conn)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (implant *Implant) ReadWithTimeout(conn net.Conn) (proto.Message, error) {
	_, msg, err := packet.ReadPacketWithTimeout(conn, implant.interval)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (implant *Implant) Run() {
	for {
		var wg sync.WaitGroup
		wg.Add(2)
		conn, err := implant.Connect()
		if err != nil {
			return
		}

		go func() {
			defer wg.Done()
			msg, err := implant.ReadWithTimeout(conn)
			if err != nil {
				logs.Log.Error(err.Error())
				return
			}
			logs.Log.Infof("%v", msg)
			go implant.Handler(msg.(*commonpb.Spites))
			return
		}()

		go func() {
			defer wg.Done()
			if implant.cache.Len() == 0 {
				err := implant.WriteEmpty(conn)
				if err != nil {
					logs.Log.Debugf(err.Error())
				}
				return
			}
			select {
			case msg := <-implant.ch:
				err := implant.Write(conn, msg)
				if err != nil {
					logs.Log.Error(err.Error())
					return
				}
				logs.Log.Debugf("send msg %v", msg)
				return
			}
		}()
		wg.Wait()
		conn.Close()
	}
}

func (implant *Implant) Handler(msg *commonpb.Spites) {
	for _, spite := range msg.Spites {
		logs.Log.Debugf("receive spite %v", spite)
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

func (implant *Implant) WriteEmpty(conn net.Conn) error {
	err := implant.Write(conn, types.BuildSpites([]*commonpb.Spite{&commonpb.Spite{Body: &commonpb.Spite_Empty{}}}))
	if err != nil {
		return err
	}
	return nil

}
