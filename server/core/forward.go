package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"google.golang.org/grpc"
)

type Listener interface {
	ID() string
	Start() (*Job, error)
}

func NewForward(rpcAddr string, listener Listener) (*Forward, error) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	logs.Log.Importantf("Forwarder connected to %s", rpcAddr)
	forward := &Forward{
		implantC:    make(chan *Message, 255),
		ImplantRpc:  listenerrpc.NewImplantRPCClient(conn),
		ListenerRpc: listenerrpc.NewListenerRPCClient(conn),
		Listener:    listener,
		ctx:         context.Background(),
	}
	forward.stream, err = forward.ListenerRpc.SpiteStream(context.Background())
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
			connect.Sender <- msg.Spite
		}
	}()
	return forward, nil
}

// Forward is a struct that handles messages from listener and server
type Forward struct {
	ctx   context.Context
	count int
	Listener
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
		spites := msg.Message.(*commonpb.Spites)
		for _, spite := range spites.Spites {
			switch spite.Body.(type) {
			case *commonpb.Spite_Register:
				_, err := f.ImplantRpc.Register(f.ctx, &lispb.RegisterSession{
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
						TaskId:     msg.TaskID,
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

func (f *Forward) handlerSpite(spite *commonpb.Spite) {
	switch spite.GetBody().(type) {
	//case *commonpb.
	}
}
