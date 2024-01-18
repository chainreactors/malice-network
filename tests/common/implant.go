package common

import (
	"errors"
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

const (
	EmptySpite  = "empty"
	StatusSpite = "status"
	AckSpite    = "ack"
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
		TaskId: 0,
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
func (implant *Implant) Write(conn net.Conn, msg *commonpb.Spites) error {
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

func (implant *Implant) WriteSpite(conn net.Conn, msg *commonpb.Spite) error {
	spites := &commonpb.Spites{
		Spites: []*commonpb.Spite{msg},
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

		// read
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

		// write
		go func() {
			defer wg.Done()
			if len(implant.ch) == 0 {
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
					implant.ch <- msg
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
		Status: &commonpb.AsyncStatus{TaskId: msg.TaskId, Status: 0},
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

func (implant *Implant) Request(req *commonpb.Spite) (proto.Message, error) {
	conn := implant.MustConnect()
	conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	defer conn.Close()
	if req != nil {
		err := implant.WriteSpite(conn, req)
		if err != nil {
			return nil, err
		}
		resp, err := implant.Read(conn)
		if err != nil {
			return nil, err
		}
		return resp, nil
	} else {
		err := implant.WriteEmpty(conn)
		if err != nil {
			return nil, err
		}
		resp, err := implant.Read(conn)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}

func (implant *Implant) Expect(req *commonpb.Spite, m types.MsgName) (proto.Message, error) {
	resp, err := implant.Request(req)
	if err != nil {
		return nil, err
	}

	if m != types.MessageType(resp.(*commonpb.Spites).Spites[0]) {
		return resp, errors.New("unexpect response type ")
	}
	return resp, nil
}

func (implant *Implant) BuildTaskSpite(msg proto.Message, taskid uint32) (*commonpb.Spite, error) {
	spite := &commonpb.Spite{
		TaskId: uint32(taskid),
	}
	_, err := types.BuildSpite(spite, msg)
	if err != nil {
		return nil, err
	}
	return spite, nil
}

func (implant *Implant) BuildCommonSpite(t string, taskId uint32) *commonpb.Spite {
	switch t {
	case EmptySpite:
		return &commonpb.Spite{Body: &commonpb.Spite_Empty{}}
	//case StatusSpite:
	//	return &commonpb.Spite{TaskId: taskId, Body: &commonpb.Spite_AsyncStatus{
	//		AsyncStatus: &commonpb.AsyncStatus{
	//			TaskId: taskId,
	//			Status: 0,
	//		},
	//	}}
	case AckSpite:
		return &commonpb.Spite{TaskId: taskId, Body: &commonpb.Spite_AsyncAck{
			AsyncAck: &commonpb.AsyncACK{
				Success: true,
			},
		}}
	default:
		return nil
	}
}

func (implant *Implant) WriteEmpty(conn net.Conn) error {
	err := implant.Write(conn, types.BuildSpites([]*commonpb.Spite{&commonpb.Spite{Body: &commonpb.Spite_Empty{}}}))
	if err != nil {
		return err
	}
	return nil
}
