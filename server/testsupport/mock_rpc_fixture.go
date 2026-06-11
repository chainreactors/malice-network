//go:build mockimplant

package testsupport

import (
	"context"
	stdpath "path"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"google.golang.org/grpc/metadata"
)

type MockRPCFixture struct {
	H       *ControlPlaneHarness
	Mock    *MockImplant
	Lib     *MockScenarioLibrary
	RPC     clientrpc.MaliceRPCClient
	Session context.Context
}

func NewMockRPCFixture(t testing.TB, pipelineName string) *MockRPCFixture {
	t.Helper()

	if strings.TrimSpace(pipelineName) == "" {
		pipelineName = "mock-implant-pipe"
	}

	h := NewControlPlaneHarness(t)
	mock := NewMockImplant(t, h, h.NewTCPPipeline(t, pipelineName))
	lib := NewMockScenarioLibrary()
	lib.Install(mock)
	if err := mock.Start(); err != nil {
		t.Fatalf("mock implant start failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := h.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return &MockRPCFixture{
		H:    h,
		Mock: mock,
		Lib:  lib,
		RPC:  clientrpc.NewMaliceRPCClient(conn),
		Session: metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"session_id", mock.SessionID,
			"callee", consts.CalleeCMD,
		)),
	}
}

func WaitTaskFinish(t testing.TB, rpc clientrpc.MaliceRPCClient, sessionID string, taskID uint32) *clientpb.TaskContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	content, err := rpc.WaitTaskFinish(ctx, &clientpb.Task{
		SessionId: sessionID,
		TaskId:    taskID,
	})
	if err != nil {
		t.Fatalf("WaitTaskFinish(%d) failed: %v", taskID, err)
	}
	if content == nil || content.Task == nil || content.Spite == nil {
		t.Fatalf("WaitTaskFinish(%d) returned incomplete content: %#v", taskID, content)
	}
	return content
}

func WaitModuleRequest(t testing.TB, mock *MockImplant, module string, before int) *clientpb.SpiteRequest {
	t.Helper()

	WaitForCondition(t, 5*time.Second, func() bool {
		return len(mock.RequestsByName(module)) >= before+1
	}, "mock implant request "+module)

	requests := mock.RequestsByName(module)
	if len(requests) <= before {
		t.Fatalf("no new request for %s after wait", module)
	}
	return requests[before]
}

func NormalizeWindowsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	path = strings.ReplaceAll(path, `\`, "/")
	path = stdpath.Clean(path)
	path = strings.ReplaceAll(path, "/", `\`)
	if len(path) == 2 && strings.HasSuffix(path, ":") {
		path += `\`
	}
	return strings.ToLower(path)
}
