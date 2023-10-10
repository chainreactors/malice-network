package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"google.golang.org/grpc"
	"sync"
)

type Listener interface {
	ID() string
	Start() (*Job, error)
}

var Forwarders = forwarders{
	forwarders: &sync.Map{},
}

func NewForward(rpcAddr string, listener Listener) *Forward {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logs.Log.Errorf("Failed to connect: %v", err)
	}
	logs.Log.Importantf("Forwarder connected to %s", rpcAddr)
	forward := &Forward{
		C:        make(chan *Message, 255),
		Rpc:      listenerrpc.NewImplantRPCClient(conn),
		Listener: listener,
		ctx:      context.Background(),
	}

	go func() {
		defer func() {
			close(forward.C)
		}()
		forward.Handler()
	}()
	return forward
}

type Forward struct {
	ctx   context.Context
	count int
	Listener

	C   chan *Message
	Rpc listenerrpc.ImplantRPCClient
}

func (f *Forward) Add(msg *Message) {
	f.C <- msg
	f.count++
}

func (f *Forward) Count() int {
	return f.count
}

func (f *Forward) Handler() {
	for msg := range f.C {
		switch msg.Message.(type) {
		case *commonpb.Spite:
			f.handlerSpite(msg.Message.(*commonpb.Spite))
		case *commonpb.Promise:
			f.handlerPromise(msg.Message.(*commonpb.Promise))
		}
	}
}

func (f *Forward) handlerPromise(promise *commonpb.Promise) {
	switch promise.GetBody().(type) {
	case *commonpb.Promise_Register:
		_, err := f.Rpc.Register(f.ctx, &lispb.RegisterSession{
			ListenerId:   f.ID(),
			RegisterData: promise.GetRegister(),
		})
		if err != nil {
			return
		}
	case *commonpb.Promise_Ping:

	}
}

func (f *Forward) handlerSpite(spite *commonpb.Spite) {
	switch spite.GetBody().(type) {
	case *commonpb.Spite_ExecRequest:
	case *commonpb.Spite_ExecResponse:

	}
}

type forwarders struct {
	forwarders *sync.Map // map[uint32]*Session
}

func (f *forwarders) Add(forwarder *Forward) {
	f.forwarders.Store(forwarder.ID(), forwarder)
}

func (f *forwarders) Get(listenerID string) *Forward {
	if forwarder, ok := f.forwarders.Load(listenerID); ok {
		return forwarder.(*Forward)
	}
	return nil
}

func (f *forwarders) Remove(listenerID string) {
	f.forwarders.Delete(listenerID)
}
