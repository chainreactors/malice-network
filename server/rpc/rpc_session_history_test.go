package rpc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func TestGetSessionHistorySortsTaskFilesNumerically(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-history-sort", "rpc-history-sort-pipe", false)

	for _, taskID := range []uint32{2, 10} {
		if err := db.AddTask(&clientpb.Task{
			SessionId: sess.ID,
			TaskId:    taskID,
			Type:      "history",
			Cur:       1,
			Total:     1,
		}); err != nil {
			t.Fatalf("AddTask(%d) failed: %v", taskID, err)
		}
	}

	writeHistoryTaskSpite(t, sess.ID, 10, 0, &implantpb.Spite{
		TaskId: 10,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 100}},
	})
	writeHistoryTaskSpite(t, sess.ID, 2, 1, &implantpb.Spite{
		TaskId: 2,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 21}},
	})
	writeHistoryTaskSpite(t, sess.ID, 2, 0, &implantpb.Spite{
		TaskId: 2,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 20}},
	})

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("session_id", sess.ID))
	history, err := (&Server{}).GetSessionHistory(ctx, &clientpb.Int{Limit: 20})
	if err != nil {
		t.Fatalf("GetSessionHistory failed: %v", err)
	}

	if len(history.GetContexts()) != 3 {
		t.Fatalf("history count = %d, want 3", len(history.GetContexts()))
	}
	if got := history.GetContexts()[0].GetTask().GetTaskId(); got != 2 {
		t.Fatalf("history[0] task id = %d, want 2", got)
	}
	if got := history.GetContexts()[0].GetSpite().GetPing().GetNonce(); got != 20 {
		t.Fatalf("history[0] nonce = %d, want 20", got)
	}
	if got := history.GetContexts()[1].GetTask().GetTaskId(); got != 2 {
		t.Fatalf("history[1] task id = %d, want 2", got)
	}
	if got := history.GetContexts()[1].GetSpite().GetPing().GetNonce(); got != 21 {
		t.Fatalf("history[1] nonce = %d, want 21", got)
	}
	if got := history.GetContexts()[2].GetTask().GetTaskId(); got != 10 {
		t.Fatalf("history[2] task id = %d, want 10", got)
	}
	if got := history.GetContexts()[2].GetSpite().GetPing().GetNonce(); got != 100 {
		t.Fatalf("history[2] nonce = %d, want 100", got)
	}
}

func TestGetSessionHistoryReturnsEmptyWhenTaskDirectoryIsMissing(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-history-missing-dir", "rpc-history-missing-dir-pipe", false)

	taskDir := filepath.Join(configs.ContextPath, sess.ID, consts.TaskPath)
	if err := os.RemoveAll(taskDir); err != nil {
		t.Fatalf("RemoveAll(%q) failed: %v", taskDir, err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("session_id", sess.ID))
	history, err := (&Server{}).GetSessionHistory(ctx, &clientpb.Int{Limit: 10})
	if err != nil {
		t.Fatalf("GetSessionHistory failed: %v", err)
	}
	if len(history.GetContexts()) != 0 {
		t.Fatalf("history contexts = %#v, want empty result", history.GetContexts())
	}
}

func writeHistoryTaskSpite(t testing.TB, sessionID string, taskID uint32, index int, spite *implantpb.Spite) {
	t.Helper()

	content, err := proto.Marshal(spite)
	if err != nil {
		t.Fatalf("Marshal(spite) failed: %v", err)
	}

	taskPath := filepath.Join(configs.ContextPath, sessionID, consts.TaskPath, taskHistoryFileName(taskID, index))
	if err := os.MkdirAll(filepath.Dir(taskPath), 0o700); err != nil {
		t.Fatalf("MkdirAll(%q) failed: %v", filepath.Dir(taskPath), err)
	}
	if err := os.WriteFile(taskPath, content, 0o600); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", taskPath, err)
	}
}

func taskHistoryFileName(taskID uint32, index int) string {
	return fmt.Sprintf("%d_%d", taskID, index)
}
