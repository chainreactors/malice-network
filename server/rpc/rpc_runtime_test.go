package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc/metadata"
)

type testJobStreamServer struct {
	ctx  context.Context
	send func(*clientpb.JobCtrl) error
	recv func() (*clientpb.JobStatus, error)
}

func (s *testJobStreamServer) SetHeader(metadata.MD) error  { return nil }
func (s *testJobStreamServer) SendHeader(metadata.MD) error { return nil }
func (s *testJobStreamServer) SetTrailer(metadata.MD)       {}
func (s *testJobStreamServer) Context() context.Context     { return s.ctx }
func (s *testJobStreamServer) SendMsg(interface{}) error    { return nil }
func (s *testJobStreamServer) RecvMsg(interface{}) error    { return nil }
func (s *testJobStreamServer) Send(msg *clientpb.JobCtrl) error {
	return s.send(msg)
}
func (s *testJobStreamServer) Recv() (*clientpb.JobStatus, error) {
	return s.recv()
}

func TestDeliverSpiteResponseReturnsErrorOnClosedOrFullChannel(t *testing.T) {
	fullCh := make(chan *implantpb.Spite, 1)
	fullCh <- &implantpb.Spite{Name: "full"}
	if err := deliverSpiteResponse(fullCh, &implantpb.Spite{Name: "next"}); err == nil {
		t.Fatal("expected full channel error")
	}

	closedCh := make(chan *implantpb.Spite)
	close(closedCh)
	if err := deliverSpiteResponse(closedCh, &implantpb.Spite{Name: "panic"}); err == nil {
		t.Fatal("expected closed channel error")
	}
}

func TestJobStreamReturnsSendErrorAndCleansCtrlState(t *testing.T) {
	lns := core.NewListener("listener-job-stream", "127.0.0.1")
	core.Listeners.Store(lns.Name, lns)
	defer core.Listeners.Delete(lns.Name)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("listener_id", lns.Name))
	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()
	streamErr := errors.New("send failed")
	stream := &testJobStreamServer{
		ctx: streamCtx,
		send: func(*clientpb.JobCtrl) error {
			return streamErr
		},
		recv: func() (*clientpb.JobStatus, error) {
			<-streamCtx.Done()
			return nil, io.EOF
		},
	}

	done := make(chan error, 1)
	go func() {
		done <- (&Server{}).JobStream(stream)
	}()

	job := &clientpb.JobCtrl{Id: 42, Job: &clientpb.Job{Name: "test-job"}}
	select {
	case lns.Ctrl <- job:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out sending job ctrl")
	}

	select {
	case err := <-done:
		if !errors.Is(err, streamErr) {
			t.Fatalf("JobStream error = %v, want %v", err, streamErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for JobStream to return")
	}

	if _, ok := lns.CtrlJob.Load(job.Id); ok {
		t.Fatal("CtrlJob entry should be removed after send failure")
	}
}

func TestPollingPublishesSessionErrorAndClearsMarker(t *testing.T) {
	oldBroker := core.EventBroker
	oldSessions := core.Sessions
	oldTicker := core.GlobalTicker
	defer func() {
		core.EventBroker = oldBroker
		core.Sessions = oldSessions
		core.GlobalTicker = oldTicker
	}()

	testTicker := core.NewTicker()
	defer testTicker.RemoveAll()
	core.GlobalTicker = testTicker

	broker := core.NewBroker()
	defer broker.Stop()
	waitEventBrokerReady(t, broker)
	sub := broker.Subscribe()
	defer broker.Unsubscribe(sub)
	core.EventBroker = broker

	core.NewSessions()
	sess := &core.Session{
		ID:         "polling-session",
		PipelineID: "missing-pipeline",
		Tasks:      core.NewTasks(),
		SessionContext: &client.SessionContext{
			SessionInfo: &client.SessionInfo{},
			Any:         map[string]interface{}{},
		},
	}
	core.Sessions.Add(sess)

	_, err := (&Server{}).Polling(context.Background(), &clientpb.Polling{
		SessionId: sess.ID,
		Id:        "polling-1",
		Interval:  1,
		Force:     true,
	})
	if err != nil {
		t.Fatalf("Polling returned error: %v", err)
	}

	deadline := time.After(2 * time.Second)
	for {
		select {
		case evt := <-sub:
			if evt.Op != consts.CtrlSessionError {
				continue
			}
			if evt.Session == nil || evt.Session.SessionId != sess.ID {
				t.Fatalf("unexpected session error event: %#v", evt)
			}
			if _, ok := sess.GetAny("polling-1"); ok {
				t.Fatal("polling marker should be cleared after failure")
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for polling session error event")
		}
	}
}

func TestPollingUsesConcurrentSafeAnyStorage(t *testing.T) {
	sess := &core.Session{
		SessionContext: &client.SessionContext{
			SessionInfo: &client.SessionInfo{},
			Any:         map[string]interface{}{},
		},
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("key-%d", i)
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sess.SetAny(id, true)
			if _, ok := sess.GetAny(id); !ok {
				t.Errorf("missing key %s", id)
			}
			sess.DeleteAny(id)
		}(id)
	}
	wg.Wait()
}
