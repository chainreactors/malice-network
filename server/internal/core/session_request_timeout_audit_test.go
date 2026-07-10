//go:build audit

package core

import (
	"errors"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

func TestAuditSessionRequestAndWaitHonorsTimeout(t *testing.T) {
	const taskID uint32 = 4242
	const requestTimeout = 25 * time.Millisecond

	sess := newTestSession("audit-request-timeout")
	defer sess.Cancel()

	sent := make(chan struct{})
	stream := &testServerStream{
		sendMsg: func(interface{}) error {
			close(sent)
			return nil
		},
	}

	type result struct {
		spite *implantpb.Spite
		err   error
	}
	resultCh := make(chan result, 1)
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		spite, err := sess.RequestAndWait(&clientpb.SpiteRequest{
			Task: &clientpb.Task{TaskId: taskID},
		}, stream, requestTimeout)
		resultCh <- result{spite: spite, err: err}
	}()
	defer func() {
		deadline := time.NewTimer(time.Second)
		defer deadline.Stop()
		for {
			select {
			case <-exited:
				sess.RemoveResp(taskID)
				return
			case <-deadline.C:
				sess.RemoveResp(taskID)
				t.Errorf("RequestAndWait goroutine did not exit during test cleanup")
				return
			default:
			}

			if respCh, ok := sess.GetResp(taskID); ok {
				select {
				case respCh <- &implantpb.Spite{Name: "audit-cleanup"}:
				default:
				}
			}
			time.Sleep(time.Millisecond)
		}
	}()

	select {
	case <-sent:
	case <-time.After(time.Second):
		t.Fatal("request was not sent")
	}

	select {
	case got := <-resultCh:
		if !errors.Is(got.err, ErrImplantSendTimeout) {
			t.Fatalf("RequestAndWait error = %v, want %v", got.err, ErrImplantSendTimeout)
		}
		if got.spite != nil {
			t.Fatalf("RequestAndWait response = %#v, want nil", got.spite)
		}
	case <-time.After(10 * requestTimeout):
		t.Fatalf("RequestAndWait did not return within %s; the timeout argument is not honored", 10*requestTimeout)
	}
}
