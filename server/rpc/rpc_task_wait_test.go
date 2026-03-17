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

func TestWaitTaskContentWaitsForNextDiskBackedCallbackIndex(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-wait-disk-index", "rpc-wait-disk-index-pipe", true)
	task := sess.NewTask("wait-disk-index", 2)
	t.Cleanup(task.Close)

	first := &implantpb.Spite{
		TaskId: task.Id,
		Name:   task.Type,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 11}},
	}
	sess.AddMessage(first, 0)
	task.Done(first, "first")
	if err := sess.TaskLog(task, first); err != nil {
		t.Fatalf("TaskLog(first) failed: %v", err)
	}

	resultCh := make(chan struct {
		ctx *clientpb.TaskContext
		err error
	}, 1)
	go func() {
		ctx, err := (&Server{}).WaitTaskContent(context.Background(), &clientpb.Task{
			SessionId: sess.ID,
			TaskId:    task.Id,
			Need:      1,
		})
		resultCh <- struct {
			ctx *clientpb.TaskContext
			err error
		}{ctx: ctx, err: err}
	}()

	select {
	case result := <-resultCh:
		t.Fatalf("WaitTaskContent returned early with %#v / %v, want to wait for callback index 1", result.ctx, result.err)
	case <-time.After(100 * time.Millisecond):
	}

	second := &implantpb.Spite{
		TaskId: task.Id,
		Name:   task.Type,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 22}},
	}
	sess.AddMessage(second, 1)
	task.Done(second, "second")
	if err := sess.TaskLog(task, second); err != nil {
		t.Fatalf("TaskLog(second) failed: %v", err)
	}

	select {
	case result := <-resultCh:
		if result.err != nil {
			t.Fatalf("WaitTaskContent returned error: %v", result.err)
		}
		if result.ctx == nil || result.ctx.Spite == nil {
			t.Fatalf("WaitTaskContent result = %#v, want second task content", result.ctx)
		}
		if got := result.ctx.Spite.GetPing().GetNonce(); got != 22 {
			t.Fatalf("WaitTaskContent nonce = %d, want 22", got)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("WaitTaskContent did not return after callback index 1 arrived")
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

func TestTaskContentHandlersRejectNilOrMissingSessionRequests(t *testing.T) {
	server := &Server{}

	if _, err := server.GetTasks(context.Background(), nil); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("GetTasks(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.GetTaskContent(context.Background(), nil); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("GetTaskContent(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.WaitTaskContent(context.Background(), nil); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("WaitTaskContent(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.WaitTaskFinish(context.Background(), nil); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("WaitTaskFinish(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := server.GetAllTaskContent(context.Background(), nil); !errors.Is(err, types.ErrMissingRequestField) {
		t.Fatalf("GetAllTaskContent(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}

	emptyTask := &clientpb.Task{}
	if _, err := server.GetTaskContent(context.Background(), emptyTask); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("GetTaskContent(empty session id) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
	if _, err := server.WaitTaskContent(context.Background(), emptyTask); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("WaitTaskContent(empty session id) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
	if _, err := server.WaitTaskFinish(context.Background(), emptyTask); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("WaitTaskFinish(empty session id) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
	if _, err := server.GetAllTaskContent(context.Background(), emptyTask); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("GetAllTaskContent(empty session id) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
	if _, err := server.GetTasks(context.Background(), &clientpb.TaskRequest{}); !errors.Is(err, types.ErrInvalidSessionID) {
		t.Fatalf("GetTasks(empty session id) error = %v, want %v", err, types.ErrInvalidSessionID)
	}
}
