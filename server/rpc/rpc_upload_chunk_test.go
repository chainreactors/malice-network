package rpc

import (
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	types "github.com/chainreactors/IoM-go/types"
)

func useTestUploadStaging(t testing.TB) *uploadStagingManager {
	t.Helper()
	m := newTestUploadStagingManager(t.TempDir())
	old := globalUploadStaging
	globalUploadStaging = m
	t.Cleanup(func() {
		globalUploadStaging = old
	})
	return m
}

func testUploadChunkRequest(uploadID string, totalSize, offset uint64, data []byte) *clientpb.UploadChunkRequest {
	return &clientpb.UploadChunkRequest{
		UploadId:  uploadID,
		Name:      "payload.bin",
		Target:    "/tmp/payload.bin",
		Priv:      0o644,
		TotalSize: totalSize,
		Offset:    offset,
		Data:      data,
		Override:  true,
	}
}

func TestUploadChunkRejectsUnknownSessionBeforeStaging(t *testing.T) {
	newRPCTestEnv(t)
	m := useTestUploadStaging(t)

	_, err := (&Server{}).UploadChunk(
		incomingSessionContext("missing-upload-session"),
		testUploadChunkRequest("unknown-session", 2, 0, []byte("a")),
	)
	if !errors.Is(err, types.ErrNotFoundSession) {
		t.Fatalf("UploadChunk error = %v, want %v", err, types.ErrNotFoundSession)
	}

	m.mu.Lock()
	gotUploads := len(m.uploads)
	m.mu.Unlock()
	if gotUploads != 0 {
		t.Fatalf("staging records = %d, want 0", gotUploads)
	}
	entries, readErr := os.ReadDir(m.tempRoot)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatalf("ReadDir staging root: %v", readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("staging files = %d, want 0", len(entries))
	}
}

func TestUploadChunkConcurrentFinalReplayReturnsSameTask(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "upload-final-session", "upload-final-pipe", true)
	useTestUploadStaging(t)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var sends atomic.Int32
	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(interface{}) error {
			sends.Add(1)
			started <- struct{}{}
			<-release
			return nil
		},
	})
	t.Cleanup(func() { pipelinesCh.Delete(sess.PipelineID) })

	type result struct {
		resp *clientpb.UploadChunkResponse
		err  error
	}
	results := make(chan result, 2)
	req := testUploadChunkRequest("concurrent-final", 3, 0, []byte("abc"))
	go func() {
		resp, err := (&Server{}).UploadChunk(incomingSessionContext(sess.ID), req)
		results <- result{resp: resp, err: err}
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first final chunk did not reach implant dispatch")
	}

	go func() {
		resp, err := (&Server{}).UploadChunk(incomingSessionContext(sess.ID), req)
		results <- result{resp: resp, err: err}
	}()
	time.Sleep(25 * time.Millisecond)
	close(release)

	first := <-results
	second := <-results
	if first.err != nil || second.err != nil {
		t.Fatalf("concurrent final errors = %v / %v", first.err, second.err)
	}
	if first.resp == nil || second.resp == nil || first.resp.Task == nil || second.resp.Task == nil {
		t.Fatalf("concurrent final responses = %#v / %#v", first.resp, second.resp)
	}
	if first.resp.NextOffset != 3 || second.resp.NextOffset != 3 {
		t.Fatalf("next offsets = %d / %d, want 3", first.resp.NextOffset, second.resp.NextOffset)
	}
	if first.resp.Task.TaskId != second.resp.Task.TaskId {
		t.Fatalf("task ids = %d / %d, want identical", first.resp.Task.TaskId, second.resp.Task.TaskId)
	}
	if got := sends.Load(); got != 1 {
		t.Fatalf("implant dispatches = %d, want 1", got)
	}

	deliverTaskResponse(t, sess, first.resp.Task.TaskId, &implantpb.Spite{
		TaskId: first.resp.Task.TaskId,
		Name:   types.MsgAck.String(),
		Body: &implantpb.Spite_Ack{Ack: &implantpb.ACK{
			Success: true,
			End:     true,
		}},
	})
	waitForTaskDone(t, sess.Tasks.Get(first.resp.Task.TaskId), "concurrent upload task")
}

func TestUploadChunkTaskFinishRemovesPartAndKeepsReplayTask(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "upload-cleanup-session", "upload-cleanup-pipe", true)
	m := useTestUploadStaging(t)

	pipelinesCh.Store(sess.PipelineID, &testRPCServerStream{
		sendMsg: func(interface{}) error { return nil },
	})
	t.Cleanup(func() { pipelinesCh.Delete(sess.PipelineID) })

	req := testUploadChunkRequest("cleanup-final", 3, 0, []byte("abc"))
	resp, err := (&Server{}).UploadChunk(incomingSessionContext(sess.ID), req)
	if err != nil {
		t.Fatalf("UploadChunk final: %v", err)
	}
	if resp.Task == nil || resp.NextOffset != 3 {
		t.Fatalf("UploadChunk response = %#v", resp)
	}

	key := m.key("anonymous", sess.ID, req.UploadId)
	m.mu.Lock()
	u := m.uploads[key]
	m.mu.Unlock()
	if u == nil {
		t.Fatal("staging record not found")
	}
	u.mu.Lock()
	stagingPath := u.stagingPath
	u.mu.Unlock()
	if _, err := os.Stat(stagingPath); err != nil {
		t.Fatalf("staging part before task finish: %v", err)
	}

	deliverTaskResponse(t, sess, resp.Task.TaskId, &implantpb.Spite{
		TaskId: resp.Task.TaskId,
		Name:   types.MsgAck.String(),
		Body: &implantpb.Spite_Ack{Ack: &implantpb.ACK{
			Success: true,
			End:     true,
		}},
	})
	waitForTaskDone(t, sess.Tasks.Get(resp.Task.TaskId), "upload task")

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if _, statErr := os.Stat(stagingPath); os.IsNotExist(statErr) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, statErr := os.Stat(stagingPath); !os.IsNotExist(statErr) {
		t.Fatalf("staging part still exists after task finish: %v", statErr)
	}

	replay, err := (&Server{}).UploadChunk(incomingSessionContext(sess.ID), req)
	if err != nil {
		t.Fatalf("final replay: %v", err)
	}
	if replay.Task == nil || replay.Task.TaskId != resp.Task.TaskId || replay.NextOffset != req.TotalSize {
		t.Fatalf("final replay response = %#v, want task %d offset %d", replay, resp.Task.TaskId, req.TotalSize)
	}
}
