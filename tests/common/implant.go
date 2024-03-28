package common

import (
	"crypto/tls"
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/server/listener/encryption"
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
		ch:       make(chan *implantpb.Spites, 255),
		cache:    types.NewSpitesCache(),
	}
	return i
}

type Implant struct {
	Addr     string
	Sid      []byte
	ch       chan *implantpb.Spites
	cache    *types.SpitesCache
	interval time.Duration
	Enc      bool
	Tls      bool
}

func (implant *Implant) Connect() (net.Conn, error) {
	conn, err := net.Dial("tcp", implant.Addr)
	if err != nil {
		return nil, err
	}

	if implant.Tls {
		conn = tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if implant.Enc {
		conn, err = encryption.WrapWithEncryption(conn, []byte("maliceofinternal"))
		if err != nil {
			return nil, err
		}
	}

	return conn, nil
}

func (implant *Implant) MustConnect() net.Conn {
	conn, err := net.Dial("tcp", implant.Addr)
	if err != nil {
		panic(err)
	}
	if implant.Tls {
		conn = tls.Client(conn, &tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if implant.Enc {
		conn, err = encryption.WrapWithEncryption(conn, []byte("maliceofinternal"))
		if err != nil {
			panic(err)
		}
	}

	return conn
}

func (implant *Implant) Register() {
	conn := implant.MustConnect()
	defer conn.Close()
	spite := &implantpb.Spite{
		TaskId: 0,
	}
	body := &implantpb.Register{
		Os: &implantpb.Os{
			Name: "windows",
		},
		Process: &implantpb.Process{
			Name: "test",
			Pid:  123,
		},
		Timer: &implantpb.Timer{
			Interval: 10,
		},
	}
	types.BuildSpite(spite, body)
	implant.WriteSpite(conn, spite)
}

// request spites
func (implant *Implant) Write(conn net.Conn, msg *implantpb.Spites) error {
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

func (implant *Implant) WriteSpite(conn net.Conn, msg *implantpb.Spite) error {
	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{msg},
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
			go implant.Handler(msg.(*implantpb.Spites))
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

func (implant *Implant) Handler(msg *implantpb.Spites) {
	for _, spite := range msg.Spites {
		logs.Log.Debugf("receive spite %v", spite)
		implant.cache.Append(implant.HandlerSpite(spite))
	}
	implant.ch <- implant.cache.Build()
}

func (implant *Implant) HandlerSpite(msg *implantpb.Spite) *implantpb.Spite {
	spite := &implantpb.Spite{
		TaskId: msg.TaskId,
		Status: &implantpb.Status{TaskId: msg.TaskId, Status: 0},
	}
	var resp proto.Message
	switch msg.Body.(type) {
	case *implantpb.Spite_ExecRequest:
		resp = &implantpb.ExecResponse{
			Stdout:     []byte("admin"),
			Pid:        999,
			StatusCode: 0,
		}
	}
	types.BuildSpite(spite, resp)
	return spite
}

func (implant *Implant) Request(req *implantpb.Spite) (proto.Message, error) {
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

func (implant *Implant) Expect(req *implantpb.Spite, m types.MsgName) (proto.Message, error) {
	resp, err := implant.Request(req)
	if err != nil {
		return nil, err
	}

	if m != types.MessageType(resp.(*implantpb.Spites).Spites[0]) {
		return resp, errors.New("unexpect response type ")
	}
	return resp, nil
}

func (implant *Implant) BuildTaskSpite(msg proto.Message, taskid uint32) (*implantpb.Spite, error) {
	spite := &implantpb.Spite{
		TaskId: uint32(taskid),
	}
	_, err := types.BuildSpite(spite, msg)
	if err != nil {
		return nil, err
	}
	return spite, nil
}

func (implant *Implant) BuildCommonSpite(t string, taskId uint32) *implantpb.Spite {
	switch t {
	case EmptySpite:
		return &implantpb.Spite{Body: &implantpb.Spite_Empty{}}
	//case StatusSpite:
	//	return & implantpb.Spite{TaskId: taskId, Body: & implantpb.Spite_AsyncStatus{
	//		AsyncStatus: & implantpb.AsyncStatus{
	//			TaskId: taskId,
	//			Status: 0,
	//		},
	//	}}
	case AckSpite:
		return &implantpb.Spite{TaskId: taskId, Body: &implantpb.Spite_AsyncAck{
			AsyncAck: &implantpb.AsyncACK{
				Success: true,
			},
		}}
	default:
		return nil
	}
}

func (implant *Implant) WriteEmpty(conn net.Conn) error {
	err := implant.Write(conn, types.BuildSpites([]*implantpb.Spite{&implantpb.Spite{Body: &implantpb.Spite_Empty{}}}))
	if err != nil {
		return err
	}
	return nil
}
