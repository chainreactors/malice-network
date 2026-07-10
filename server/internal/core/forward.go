package core

import (
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	types "github.com/chainreactors/IoM-go/types"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func forwardRawIDBytes(rawID uint32) []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, rawID)
	return data
}

var (
	Forwarders = &forwarders{
		forwarders: &sync.Map{},
	}
)

func PipelineRuntimeKey(listenerID, pipelineID string) string {
	if listenerID == "" || pipelineID == "" {
		return pipelineID
	}
	return listenerID + ":" + pipelineID
}

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

type ForwardClient interface {
	forwardRPCClient
	OpenForwardStream(ctx context.Context, pipeline Pipeline) (ForwardStream, error)
}

type ForwardStream interface {
	Send(*clientpb.SpiteResponse) error
	Recv() (*clientpb.SpiteRequest, error)
}

type reverseForwardClient struct {
	client listenerrpc.ListenerRPCClient
}

func NewReverseForwardClient(client listenerrpc.ListenerRPCClient) ForwardClient {
	return &reverseForwardClient{client: client}
}

func (c *reverseForwardClient) Checkin(ctx context.Context, in *implantpb.Ping, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return c.client.Checkin(ctx, in, opts...)
}

func (c *reverseForwardClient) Register(ctx context.Context, in *clientpb.RegisterSession, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	return c.client.Register(ctx, in, opts...)
}

func (c *reverseForwardClient) OpenForwardStream(ctx context.Context, pipeline Pipeline) (ForwardStream, error) {
	listenerID := ""
	if pb := pipeline.ToProtobuf(); pb != nil {
		listenerID = pb.ListenerId
	}
	return c.client.SpiteStream(metadata.NewOutgoingContext(ctx, metadata.Pairs(
		"pipeline_id", pipeline.ID(),
		"listener_id", listenerID),
	))
}

type forwarders struct {
	forwarders *sync.Map
}

func (f *forwarders) Add(fw *Forward) {
	f.forwarders.Store(fw.RuntimeKey(), fw)
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
	fw.shutdown()
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

func NewForward(rpc ForwardClient, pipeline Pipeline) (*Forward, error) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())
	listenerID := ""
	if pb := pipeline.ToProtobuf(); pb != nil {
		listenerID = pb.ListenerId
	}
	forward := &Forward{
		implantC:    make(chan *Message, 255),
		ListenerRpc: rpc,
		Pipeline:    pipeline,
		ListenerId:  listenerID,
		ctx:         ctx,
		cancel:      cancel,
		done:        make(chan struct{}),
		handlerDone: make(chan struct{}),
	}
	forward.alive.Store(true)

	forward.Stream, err = rpc.OpenForwardStream(ctx, pipeline)
	if err != nil {
		cancel()
		return nil, err
	}

	GoGuarded("forward:"+pipeline.ID(), func() error {
		defer close(forward.handlerDone)
		return forward.Handler()
	}, forward.handleRuntimeError(), forward.shutdown)

	return forward, nil
}

// Forward is a struct that handles messages from listener and server
type Forward struct {
	ctx    context.Context
	cancel context.CancelFunc
	count  int
	Pipeline
	ListenerId string
	Stream     ForwardStream
	implantC   chan *Message // data from implant

	ListenerRpc forwardRPCClient

	alive       atomic.Bool
	done        chan struct{}
	closeOnce   sync.Once
	handlerDone chan struct{}
	abortOnce   sync.Once
	abortErr    error
}

func (f *Forward) RuntimeKey() string {
	if f == nil {
		return ""
	}
	return PipelineRuntimeKey(f.ListenerId, f.ID())
}

func (f *Forward) Add(msg *Message) {
	if !f.alive.Load() {
		logs.Log.Warnf("forward %s is not alive, dropping message from %s", f.ID(), msg.SessionID)
		return
	}
	select {
	case f.implantC <- msg:
		f.count++
	case <-f.done:
		logs.Log.Warnf("forward %s closed, dropping message from %s", f.ID(), msg.SessionID)
	}
}

func (f *Forward) shutdown() {
	f.alive.Store(false)
	f.closeOnce.Do(func() {
		if f.cancel != nil {
			f.cancel()
		}
		close(f.done)
	})
}

// Abort closes an uncommitted forward without touching the global registry or
// closing its pipeline. It is safe to call more than once.
func (f *Forward) Abort() error {
	if f == nil {
		return nil
	}
	f.abortOnce.Do(func() {
		f.shutdown()
		if stream, ok := f.Stream.(interface{ CloseSend() error }); ok {
			f.abortErr = stream.CloseSend()
		}
		if f.handlerDone != nil {
			<-f.handlerDone
		}
	})
	return f.abortErr
}

func (f *Forward) Count() int {
	return f.count
}

func (f *Forward) Context(sid string) context.Context {
	return metadata.NewOutgoingContext(f.ctx, metadata.Pairs(
		"session_id", sid,
		"listener_id", f.ListenerId,
		"pipeline_id", f.ID(),
		"timestamp", strconv.FormatInt(time.Now().Unix(), 10),
	))
}

// Handler is a loop that handles messages from implant
func (f *Forward) Handler() error {
	for {
		var msg *Message
		select {
		case <-f.done:
			return nil
		case received, ok := <-f.implantC:
			if !ok {
				return nil
			}
			msg = received
		}
		_, err := f.ListenerRpc.Checkin(f.Context(msg.SessionID), &implantpb.Ping{})
		if err != nil {
			logs.Log.Warnf("forward %s checkin failed for session %s: %v", f.ID(), msg.SessionID, err)
			initSpite, _ := types.BuildSpite(
				&implantpb.Spite{
					Name: types.MsgInit.String(),
				},
				&implantpb.Init{Data: forwardRawIDBytes(msg.RawID)})
			if pushErr := Connections.Push(msg.SessionID, &clientpb.SpiteRequest{
				Spite: initSpite,
			}); pushErr != nil {
				logs.Log.Errorf("forward %s init spite push failed for session %s: %v", f.ID(), msg.SessionID, pushErr)
			}
		}
		for _, spite := range msg.Spites.Spites {
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
					logs.Log.Debugf("listener.%s - receive_spite session=%s name=%s spite=%v", msg.SessionID, msg.SessionID, spite.Name, spite)
				} else {
					logs.Log.Debugf("listener.%s - receive_spite session=%s name=%s bytes=%d", msg.SessionID, msg.SessionID, spite.Name, size)
				}
				if err := f.Stream.Send(&clientpb.SpiteResponse{
					ListenerId: f.ListenerId,
					SessionId:  msg.SessionID,
					TaskId:     spite.TaskId,
					Spite:      spite,
				}); err != nil {
					return fmt.Errorf("forward %s send spite response: %w", f.ID(), err)
				}
			}
		}
	}
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
