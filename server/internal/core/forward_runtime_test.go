package core

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/grpc"
)

type testForwardRPC struct{}

func (testForwardRPC) Checkin(context.Context, *implantpb.Ping, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

func (testForwardRPC) Register(context.Context, *clientpb.RegisterSession, ...grpc.CallOption) (*clientpb.Empty, error) {
	return &clientpb.Empty{}, nil
}

type testForwardStream struct {
	sendErr error
}

func (s testForwardStream) Send(*clientpb.SpiteResponse) error {
	return s.sendErr
}

func (testForwardStream) Recv() (*clientpb.SpiteRequest, error) {
	return nil, errors.New("not used")
}

type testPipeline struct {
	id       string
	closeErr error
}

func (p testPipeline) ID() string { return p.id }

func (testPipeline) Start() error { return nil }

func (p testPipeline) Close() error { return p.closeErr }

func (p testPipeline) ToProtobuf() *clientpb.Pipeline {
	return &clientpb.Pipeline{Name: p.id}
}

func TestForwardHandlerReturnsStreamSendError(t *testing.T) {
	want := errors.New("stream send failed")
	forward := &Forward{
		ctx:         context.Background(),
		Pipeline:    testPipeline{id: "pipe-a"},
		ListenerRpc: testForwardRPC{},
		Stream:      testForwardStream{sendErr: want},
		implantC:    make(chan *Message, 1),
	}

	forward.implantC <- &Message{
		SessionID:  "session-a",
		Spites:     &implantpb.Spites{Spites: []*implantpb.Spite{{Name: "exec"}}},
		RemoteAddr: "127.0.0.1:9000",
	}
	close(forward.implantC)

	err := forward.Handler()
	if !errors.Is(err, want) {
		t.Fatalf("Forward.Handler error = %v, want %v", err, want)
	}
}

func TestForwardersRemoveDeletesOnCloseError(t *testing.T) {
	want := errors.New("close failed")
	store := &forwarders{forwarders: &sync.Map{}}
	forward := &Forward{
		Pipeline: testPipeline{id: "pipe-remove", closeErr: want},
		Stream:   testForwardStream{},
	}
	store.Add(forward)

	err := store.Remove(forward.ID())
	if !errors.Is(err, want) {
		t.Fatalf("Remove error = %v, want %v", err, want)
	}
	if got := store.Get(forward.ID()); got != nil {
		t.Fatalf("expected forwarder to be deleted, got %#v", got)
	}
}
