package core

import (
	"context"
	"fmt"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	types "github.com/chainreactors/IoM-go/types"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
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

type forwardRPCClient interface {
	Checkin(ctx context.Context, in *implantpb.Ping, opts ...grpc.CallOption) (*clientpb.Empty, error)
	Register(ctx context.Context, in *clientpb.RegisterSession, opts ...grpc.CallOption) (*clientpb.Empty, error)
}

type forwardClient interface {
	forwardRPCClient
	SpiteStream(ctx context.Context, opts ...grpc.CallOption) (listenerrpc.ListenerRPC_SpiteStreamClient, error)
}

type forwardStream interface {
	Send(*clientpb.SpiteResponse) error
	Recv() (*clientpb.SpiteRequest, error)
}

var openForwardStream = func(rpc forwardClient, ctx context.Context) (forwardStream, error) {
	return rpc.SpiteStream(ctx)
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

func (f *forwarders) Remove(id string) error {
	fw := f.Get(id)
	if fw == nil {
		return nil
	}
	f.forwarders.Delete(id)
	err := fw.Close()
	if err != nil {
		return err
	}
	return nil
}

func (f *forwarders) Send(id string, msg *Message) {
	fw := f.Get(id)
	if fw == nil {
		logs.Log.Errorf("forwarder %s not found", id)
		return
	}
	fw.Add(msg)
}

func NewForward(rpc forwardClient, pipeline Pipeline) (*Forward, error) {
	var err error
	forward := &Forward{
		implantC:    make(chan *Message, 255),
		ListenerRpc: rpc,
		Pipeline:    pipeline,
		ctx:         context.Background(),
	}

	forward.Stream, err = openForwardStream(rpc, metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
		"pipeline_id", pipeline.ID()),
	))
	if err != nil {
		return nil, err
	}

	GoGuarded("forward:"+pipeline.ID(), forward.Handler, forward.handleRuntimeError())

	return forward, nil
}

// Forward is a struct that handles messages from listener and server
type Forward struct {
	ctx   context.Context
	count int
	Pipeline
	ListenerId string
	Stream     forwardStream
	implantC   chan *Message // data from implant

	ListenerRpc forwardRPCClient
}

func (f *Forward) Add(msg *Message) {
	f.implantC <- msg
	f.count++
}

func (f *Forward) Count() int {
	return f.count
}

func (f *Forward) Context(sid string) context.Context {
	return metadata.NewOutgoingContext(f.ctx, metadata.Pairs(
		"session_id", sid,
		"listener_id", f.ListenerId,
		"timestamp", strconv.FormatInt(time.Now().Unix(), 10),
	))
}

// Handler is a loop that handles messages from implant
func (f *Forward) Handler() error {
	for msg := range f.implantC {
		for _, spite := range msg.Spites.Spites {
			_, err := f.ListenerRpc.Checkin(f.Context(msg.SessionID), &implantpb.Ping{})
			if err != nil {
				logs.Log.Warnf("forward %s checkin failed for session %s: %v", f.ID(), msg.SessionID, err)
				spite, _ := types.BuildSpite(
					&implantpb.Spite{
						Name: types.MsgInit.String(),
					},
					&implantpb.Init{Data: (*[4]byte)(unsafe.Pointer(&msg.RawID))[:]})
				err = Connections.Push(msg.SessionID, &clientpb.SpiteRequest{
					Spite: spite,
				})
				if err != nil {
					logs.Log.Errorf("forward %s init spite push failed for session %s: %v", f.ID(), msg.SessionID, err)
				}
			}
			switch spite.Body.(type) {
			case *implantpb.Spite_Register:
				_, err := f.ListenerRpc.Register(f.Context(msg.SessionID), &clientpb.RegisterSession{
					SessionId:    msg.SessionID,
					PipelineId:   f.ID(),
					ListenerId:   f.ListenerId,
					RegisterData: spite.GetRegister(),
					Target:       msg.RemoteAddr,
					RawId:        msg.RawID,
				})
				if err != nil {
					logs.Log.Errorf("register err %s", err.Error())
					continue
				}
			case *implantpb.Spite_Ping:
				continue
			default:
				if size := proto.Size(spite); size <= 1000 {
					logs.Log.Debugf("[listener.%s] receive spite %s, %v", msg.SessionID, spite.Name, spite)
				} else {
					logs.Log.Debugf("[listener.%s] receive spite %s %d bytes", msg.SessionID, spite.Name, size)
				}
				if err := f.Stream.Send(&clientpb.SpiteResponse{
					ListenerId: f.ID(),
					SessionId:  msg.SessionID,
					TaskId:     spite.TaskId,
					Spite:      spite,
				}); err != nil {
					return fmt.Errorf("forward %s send spite response: %w", f.ID(), err)
				}
			}
		}
	}
	return nil
}

func (f *Forward) handleRuntimeError() GoErrorHandler {
	label := "forward:" + f.ID()
	return CombineErrorHandlers(
		LogGuardedError(label),
		func(err error) {
			logs.Log.Errorf("[%s] runtime failure: %s", label, ErrorText(err))
		},
	)
}
