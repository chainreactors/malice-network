package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func writeTaskRequestToDisk(t testing.TB, sessionID string, taskID uint32, spite *implantpb.Spite) []byte {
	t.Helper()

	requestPath, err := taskRequestPathFor(sessionID, taskID)
	if err != nil {
		t.Fatalf("taskRequestPathFor: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(requestPath), 0o700); err != nil {
		t.Fatalf("MkdirAll request dir: %v", err)
	}
	data, err := proto.Marshal(spite)
	if err != nil {
		t.Fatalf("Marshal request: %v", err)
	}
	if err := os.WriteFile(requestPath, data, 0o600); err != nil {
		t.Fatalf("WriteFile request: %v", err)
	}
	return data
}

func TestQueryTasksUsesStoredMetadata(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "qt-meta-sess", "qt-meta-pipe", true)
	request := &implantpb.Spite{
		Name:   consts.ModuleExecute,
		TaskId: 1,
		Body: &implantpb.Spite_ExecRequest{
			ExecRequest: &implantpb.ExecRequest{
				Path: "whoami",
				Args: []string{"/all"},
			},
		},
	}
	requestBytes := writeTaskRequestToDisk(t, sess.ID, 1, request)
	requestHash := sha256.Sum256(requestBytes)
	requestSummary, err := core.BuildTaskRequestSummaryJSON(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AddTask(&clientpb.Task{
		SessionId:      sess.ID,
		TaskId:         1,
		Type:           consts.ModuleExecute,
		CommandSummary: core.BuildTaskCommandSummary(request),
		RequestSummary: requestSummary,
		RequestSize:    int64(len(requestBytes)),
		RequestSha256:  hex.EncodeToString(requestHash[:]),
		HasRequest:     true,
	}); err != nil {
		t.Fatalf("AddTask: %v", err)
	}

	resp, err := (&Server{}).QueryTasks(context.Background(), &clientpb.TaskQuery{
		SessionId:             sess.ID,
		PageSize:              1,
		IncludeRequestSummary: true,
		IncludeTotalCount:     true,
	})
	if err != nil {
		t.Fatalf("QueryTasks: %v", err)
	}
	if resp.GetTotalCount() != 1 {
		t.Fatalf("total_count = %d, want 1", resp.GetTotalCount())
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("tasks = %d, want 1", len(resp.GetTasks()))
	}
	task := resp.GetTasks()[0].GetTask()
	if task.GetCommandSummary() != "exec whoami -- /all" {
		t.Fatalf("command summary = %q", task.GetCommandSummary())
	}
	if task.GetRequestSummary() == "" {
		t.Fatal("request summary should be included")
	}
	if task.GetRequestSize() != int64(len(requestBytes)) {
		t.Fatalf("request size = %d, want %d", task.GetRequestSize(), len(requestBytes))
	}
	if resp.GetTasks()[0].GetRawRequest() != nil {
		t.Fatal("raw request should not be returned unless requested")
	}
}

func TestQueryTasksBuildsMissingSummaryFromRequestCache(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "qt-cache-sess", "qt-cache-pipe", true)
	request := &implantpb.Spite{
		Name:   consts.ModulePing,
		TaskId: 3,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 99}},
	}
	writeTaskRequestToDisk(t, sess.ID, 3, request)
	if err := db.AddTask(&clientpb.Task{
		SessionId: sess.ID,
		TaskId:    3,
		Type:      consts.ModulePing,
	}); err != nil {
		t.Fatalf("AddTask: %v", err)
	}

	resp, err := (&Server{}).QueryTasks(context.Background(), &clientpb.TaskQuery{
		SessionId:             sess.ID,
		TaskIds:               []uint32{3},
		PageSize:              1,
		IncludeRequestSummary: true,
		IncludeRawRequest:     true,
	})
	if err != nil {
		t.Fatalf("QueryTasks: %v", err)
	}
	detail := resp.GetTasks()[0]
	if detail.GetTask().GetRequestSummary() == "" {
		t.Fatal("request summary should be built from request cache")
	}
	if detail.GetTask().GetCommandSummary() != consts.ModulePing {
		t.Fatalf("command summary = %q, want %q", detail.GetTask().GetCommandSummary(), consts.ModulePing)
	}
	if detail.GetRawRequest().GetPing().GetNonce() != 99 {
		t.Fatalf("raw ping nonce = %d, want 99", detail.GetRawRequest().GetPing().GetNonce())
	}
}

func TestQueryTasksPaginationAndTotalCount(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "qt-page-sess", "qt-page-pipe", true)
	for i := uint32(1); i <= 3; i++ {
		if err := db.AddTask(&clientpb.Task{
			SessionId: sess.ID,
			TaskId:    i,
			Type:      consts.ModulePing,
		}); err != nil {
			t.Fatalf("AddTask(%d): %v", i, err)
		}
	}

	resp, err := (&Server{}).QueryTasks(context.Background(), &clientpb.TaskQuery{
		SessionId:         sess.ID,
		PageSize:          2,
		IncludeTotalCount: true,
	})
	if err != nil {
		t.Fatalf("QueryTasks: %v", err)
	}
	if len(resp.GetTasks()) != 2 {
		t.Fatalf("tasks = %d, want 2", len(resp.GetTasks()))
	}
	if resp.GetNextPageToken() != "2" {
		t.Fatalf("next_page_token = %q, want 2", resp.GetNextPageToken())
	}
	if resp.GetTotalCount() != 3 {
		t.Fatalf("total_count = %d, want 3", resp.GetTotalCount())
	}
}

func TestQueryTasksWithoutSessionReturnsRecentTasksAcrossSessions(t *testing.T) {
	env := newRPCTestEnv(t)
	first := env.seedSession(t, "qt-global-first", "qt-global-pipe-1", true)
	second := env.seedSession(t, "qt-global-second", "qt-global-pipe-2", true)
	for _, task := range []*clientpb.Task{
		{SessionId: first.ID, TaskId: 1, Type: consts.ModulePing},
		{SessionId: second.ID, TaskId: 2, Type: consts.ModuleExecute},
	} {
		if err := db.AddTask(task); err != nil {
			t.Fatalf("AddTask(%s): %v", task.GetSessionId(), err)
		}
	}

	resp, err := (&Server{}).QueryTasks(context.Background(), &clientpb.TaskQuery{
		PageSize:          10,
		IncludeTotalCount: true,
	})
	if err != nil {
		t.Fatalf("QueryTasks: %v", err)
	}
	if resp.GetTotalCount() != 2 {
		t.Fatalf("total_count = %d, want 2", resp.GetTotalCount())
	}
	seen := make(map[string]bool)
	for _, detail := range resp.GetTasks() {
		seen[detail.GetTask().GetSessionId()] = true
	}
	if !seen[first.ID] || !seen[second.ID] {
		t.Fatalf("global task query sessions = %#v", seen)
	}
}

func TestQueryTasksIncludesResults(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "qt-results-sess", "qt-results-pipe", true)
	if err := db.AddTask(&clientpb.Task{
		SessionId: sess.ID,
		TaskId:    4,
		Type:      consts.ModulePing,
	}); err != nil {
		t.Fatalf("AddTask: %v", err)
	}
	writeTaskSpiteToDisk(t, sess.ID, 4, 1, &implantpb.Spite{
		TaskId: 4,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 2}},
	})
	writeTaskSpiteToDisk(t, sess.ID, 4, 0, &implantpb.Spite{
		TaskId: 4,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 1}},
	})

	resp, err := (&Server{}).QueryTasks(context.Background(), &clientpb.TaskQuery{
		SessionId:      sess.ID,
		TaskIds:        []uint32{4},
		PageSize:       1,
		IncludeResults: true,
	})
	if err != nil {
		t.Fatalf("QueryTasks: %v", err)
	}
	results := resp.GetTasks()[0].GetResults()
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].GetPing().GetNonce() != 1 || results[1].GetPing().GetNonce() != 2 {
		t.Fatalf("results are not sorted by index: %#v", results)
	}
}

func TestQueryTasksRejectsLargeExpandedList(t *testing.T) {
	_ = newRPCTestEnv(t)
	_, err := (&Server{}).QueryTasks(context.Background(), &clientpb.TaskQuery{
		SessionId:         "qt-invalid-sess",
		PageSize:          11,
		IncludeRawRequest: true,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("QueryTasks error = %v, want InvalidArgument", err)
	}
}
