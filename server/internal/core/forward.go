package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"sync"
)

var (
	Forwarders = &forwarders{
		forwarders: &sync.Map{},
	}
)

type Message struct {
	proto.Message
	End       bool
	SessionID string
	MessageID string
}

type forwarders struct {
	forwarders *sync.Map
}

func (f *forwarders) Add(fw *Forward) {
	f.forwarders.Store(fw.ID(), fw)
}

func (f *forwarders) Get(id string) *Forward {
	fw, ok := f.forwarders.Load(id)
	if !ok {
		return nil
	}
	return fw.(*Forward)
}

func (f *forwarders) Remove(id string) {
	fw := f.Get(id)
	if fw == nil {
		return
	}
	err := fw.Close()
	if err != nil {
		return
	}
	f.forwarders.Delete(id)
}

func (f *forwarders) Send(id string, msg *Message) {
	fw := f.Get(id)
	if fw == nil {
		return
	}
	fw.Add(msg)
}

func NewForward(conn *grpc.ClientConn, pipeline Pipeline) (*Forward, error) {
	var err error
	forward := &Forward{
		implantC:    make(chan *Message, 255),
		ImplantRpc:  listenerrpc.NewImplantRPCClient(conn),
		ListenerRpc: listenerrpc.NewListenerRPCClient(conn),
		Pipeline:    pipeline,
		ctx:         context.Background(),
	}

	forward.stream, err = forward.ListenerRpc.SpiteStream(metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"listener_id", pipeline.ID()),
	))
	if err != nil {
		return nil, err
	}

	go func() {
		// read message from implant and handler message to server
		forward.Handler()
	}()

	go func() {
		// recv message from server and send to implant
		for {
			msg, err := forward.stream.Recv()
			if err != nil {
				return
			}
			connect := Connections.Get(msg.SessionId)
			if connect == nil {
				logs.Log.Errorf("connection %s not found", msg.SessionId)
				continue
			}
			connect.C <- msg.Spite
		}
	}()
	return forward, nil
}

// Forward is a struct that handles messages from listener and server
type Forward struct {
	ctx   context.Context
	count int
	Pipeline
	stream   listenerrpc.ListenerRPC_SpiteStreamClient
	implantC chan *Message // data from implant

	ImplantRpc  listenerrpc.ImplantRPCClient
	ListenerRpc listenerrpc.ListenerRPCClient
}

func (f *Forward) Add(msg *Message) {
	f.implantC <- msg
	f.count++
}

func (f *Forward) Count() int {
	return f.count
}

// Handler is a loop that handles messages from implant
func (f *Forward) Handler() {
	for msg := range f.implantC {
		spites := msg.Message.(*implantpb.Spites)
		for _, spite := range spites.Spites {
			if size := proto.Size(spite); size <= 1000 {
				logs.Log.Debugf("[listener.%s] receive spite %s, %v", msg.SessionID, spite.Name, spite)
			} else {
				logs.Log.Debugf("[listener.%s] receive spite %s %d bytes", msg.SessionID, spite.Name, size)
			}
			switch spite.Body.(type) {
			case *implantpb.Spite_Empty:
				continue
			case *implantpb.Spite_Register:
				_, err := f.ImplantRpc.Register(f.ctx, &lispb.RegisterSession{
					SessionId:    msg.SessionID,
					ListenerId:   f.ID(),
					RegisterData: spite.GetRegister(),
				})
				if err != nil {
					return
				}
			default:
				spite := spite
				go func() {
					err := f.stream.Send(&lispb.SpiteSession{
						ListenerId: f.ID(),
						SessionId:  msg.SessionID,
						TaskId:     spite.TaskId,
						Spite:      spite,
					})
					if err != nil {
						return
					}
				}()
			}
		}
	}
}
