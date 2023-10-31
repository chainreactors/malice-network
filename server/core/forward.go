package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
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
			connect.Sender <- msg.Spite
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
		spites := msg.Message.(*commonpb.Spites)
		for _, spite := range spites.Spites {
			switch spite.Body.(type) {
			case *commonpb.Spite_Register:
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

type Pipeline interface {
	ID() string
	Start() error
	Addr() string
	Close() error
	ToProtobuf() proto.Message
}

type Pipelines []Pipeline

func (ps Pipelines) Add(p Pipeline) {
	ps = append(ps, p)
}

func (ps Pipelines) ToProtobuf() []*clientpb.Pipeline {
	var pls []*clientpb.Pipeline
	for _, p := range ps {
		msg := &clientpb.Pipeline{
			Name: p.ID(),
		}
		types.BuildPipeline(msg, p.ToProtobuf())
		pls = append(pls, msg)
	}
	return pls
}
