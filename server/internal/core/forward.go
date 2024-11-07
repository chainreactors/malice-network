package core

import (
	"context"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"

	"github.com/chainreactors/malice-network/helper/proto/services/listenerrpc"
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
	Spites     *implantpb.Spites
	RawID      uint32
	SessionID  string
	RemoteAddr string
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
		logs.Log.Errorf("forwarder %s not found", id)
		return
	}
	fw.Add(msg)
}

func NewForward(conn *grpc.ClientConn, pipeline Pipeline) (*Forward, error) {
	var err error
	forward := &Forward{
		implantC:    make(chan *Message, 255),
		ListenerRpc: listenerrpc.NewListenerRPCClient(conn),
		Pipeline:    pipeline,
		ctx:         context.Background(),
	}

	forward.Stream, err = forward.ListenerRpc.SpiteStream(metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"pipeline_id", pipeline.ID()),
	))
	if err != nil {
		return nil, err
	}

	go forward.Handler()

	return forward, nil
}

// Forward is a struct that handles messages from listener and server
type Forward struct {
	ctx   context.Context
	count int
	Pipeline
	Stream   listenerrpc.ListenerRPC_SpiteStreamClient
	implantC chan *Message // data from implant

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
		_, err := f.ListenerRpc.Checkin(metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"session_id", msg.SessionID),
		), &implantpb.Ping{})
		if err != nil {
			logs.Log.Error(err)
			continue
		}
		for _, spite := range msg.Spites.Spites {
			switch spite.Body.(type) {
			case *implantpb.Spite_Register:
				_, err := f.ListenerRpc.Register(f.ctx, &clientpb.RegisterSession{
					SessionId:    msg.SessionID,
					PipelineId:   f.ID(),
					RegisterData: spite.GetRegister(),
					Target:       msg.RemoteAddr,
					RawId:        msg.RawID,
				})
				if err != nil {
					logs.Log.Errorf("register err %s", err.Error())
					continue
				}
			default:
				if size := proto.Size(spite); size <= 1000 {
					logs.Log.Debugf("[listener.%s] receive spite %s, %v", msg.SessionID, spite.Name, spite)
				} else {
					logs.Log.Debugf("[listener.%s] receive spite %s %d bytes", msg.SessionID, spite.Name, size)
				}
				spite := spite
				go func() {
					err := f.Stream.Send(&clientpb.SpiteResponse{
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
