package rpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

func TestWaitTaskContentReturnsWhenTaskProgressArrives(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-wait-content", "rpc-wait-content-pipe", true)
	task := sess.NewTask("wait-content", 1)
	t.Cleanup(task.Close)

	resultCh := make(chan struct {
		ctx *clientpb.TaskContext
		err error
	}, 1)
	go func() {
		ctx, err := (&Server{}).WaitTaskContent(context.Background(), &clientpb.Task{
			SessionId: sess.ID,
			TaskId:    task.Id,
			Need:      0,
		})
		resultCh <- struct {
			ctx *clientpb.TaskContext
			err error
		}{ctx: ctx, err: err}
	}()

	time.Sleep(50 * time.Millisecond)
	spite := &implantpb.Spite{
		TaskId: task.Id,
		Name:   task.Type,
		Body:   &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
	}
	sess.AddMessage(spite, 0)
	task.Done(spite, "ready")

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("WaitTaskContent returned error: %v", result.err)
		}
		if result.ctx == nil || result.ctx.Spite == nil || result.ctx.Spite.TaskId != task.Id {
			t.Fatalf("WaitTaskContent result = %#v, want spite for task %d", result.ctx, task.Id)
		}
	case <-time.After(500 * time.Millisecond):
		task.Close()
		t.Fatal("WaitTaskContent did not return after task progress arrived")
	}
}

func TestWaitTaskContentRejectsIndexEqualToTotal(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-wait-index", "rpc-wait-index-pipe", true)
	task := sess.NewTask("wait-index", 1)
	t.Cleanup(task.Close)

	_, err := (&Server{}).WaitTaskContent(context.Background(), &clientpb.Task{
		SessionId: sess.ID,
		TaskId:    task.Id,
		Need:      1,
	})
	if !errors.Is(err, types.ErrTaskIndexExceed) {
		t.Fatalf("WaitTaskContent error = %v, want %v", err, types.ErrTaskIndexExceed)
	}
}

func TestWaitTaskContentReturnsWhenCallerContextCancels(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-wait-content-cancel", "rpc-wait-content-cancel-pipe", true)
	task := sess.NewTask("wait-content-cancel", 1)
	t.Cleanup(task.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := (&Server{}).WaitTaskContent(ctx, &clientpb.Task{
			SessionId: sess.ID,
			TaskId:    task.Id,
			Need:      0,
		})
		resultCh <- err
	}()

	select {
	case err := <-resultCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("WaitTaskContent cancel error = %v, want %v", err, context.DeadlineExceeded)
		}
	case <-time.After(500 * time.Millisecond):
		task.Close()
		t.Fatal("WaitTaskContent did not return when caller context canceled")
	}
}

func TestWaitTaskFinishReturnsWhenCallerContextCancels(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-wait-finish-cancel", "rpc-wait-finish-cancel-pipe", true)
	task := sess.NewTask("wait-finish-cancel", 1)
	t.Cleanup(task.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := (&Server{}).WaitTaskFinish(ctx, &clientpb.Task{
			SessionId: sess.ID,
			TaskId:    task.Id,
		})
		resultCh <- err
	}()

	select {
	case err := <-resultCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("WaitTaskFinish cancel error = %v, want %v", err, context.DeadlineExceeded)
		}
	case <-time.After(500 * time.Millisecond):
		task.Close()
		t.Fatal("WaitTaskFinish did not return when caller context canceled")
	}
}
