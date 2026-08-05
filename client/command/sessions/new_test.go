package sessions_test

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	sessionscmd "github.com/chainreactors/malice-network/client/command/sessions"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestNewBindSessionRegistersResolvedPipelineRoute(t *testing.T) {
	for _, pipelineID := range []string{"bind-main", "listener-a:bind-main"} {
		t.Run(pipelineID, func(t *testing.T) {
			h := testsupport.NewClientHarness(t)
			h.Console.Pipelines["listener-a:bind-main"] = bindPipeline("bind-main", "listener-a")
			prepareRegisterResponse(h)

			sess, err := sessionscmd.NewBindSession(h.Console, pipelineID, "10.0.0.8:5001", "bind-01")
			if err != nil {
				t.Fatalf("NewBindSession failed: %v", err)
			}
			t.Cleanup(func() { _ = sess.Close() })

			req := recordedRegisterRequest(t, h)
			if req.GetPipelineId() != "bind-main" || req.GetListenerId() != "listener-a" {
				t.Fatalf("register route = %q/%q, want %q/%q", req.GetListenerId(), req.GetPipelineId(), "listener-a", "bind-main")
			}
		})
	}
}

func TestNewBindSessionCommandUsesPositionalName(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-a:bind-main"] = bindPipeline("bind-main", "listener-a")
	prepareRegisterResponse(h)

	err := h.ExecuteClient(
		consts.CommandSession,
		consts.CommandNewBindSession,
		"bind-01",
		"--target", "10.0.0.8:5001",
		"--pipeline", "listener-a:bind-main",
	)
	if err != nil {
		t.Fatalf("session newbind failed: %v", err)
	}

	req := recordedRegisterRequest(t, h)
	if req.GetRegisterData().GetName() != "bind-01" {
		t.Fatalf("register name = %q, want %q", req.GetRegisterData().GetName(), "bind-01")
	}
}

func TestNewBindSessionRejectsUnresolvablePipeline(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testsupport.Harness)
		pipeline string
		wantErr  string
	}{
		{
			name:     "missing",
			pipeline: "missing",
			wantErr:  "missing",
		},
		{
			name: "ambiguous",
			setup: func(h *testsupport.Harness) {
				h.Console.Pipelines["listener-a:shared"] = bindPipeline("shared", "listener-a")
				h.Console.Pipelines["listener-b:shared"] = bindPipeline("shared", "listener-b")
			},
			pipeline: "shared",
			wantErr:  "ambiguous",
		},
		{
			name: "not bind",
			setup: func(h *testsupport.Harness) {
				h.Console.Pipelines["listener-a:http-main"] = &clientpb.Pipeline{
					Name:       "http-main",
					ListenerId: "listener-a",
					Type:       consts.HTTPPipeline,
				}
			},
			pipeline: "listener-a:http-main",
			wantErr:  "not a bind pipeline",
		},
		{
			name: "listener missing",
			setup: func(h *testsupport.Harness) {
				h.Console.Pipelines["legacy-bind"] = bindPipeline("legacy-bind", "")
			},
			pipeline: "legacy-bind",
			wantErr:  "has no listener",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := testsupport.NewClientHarness(t)
			if tt.setup != nil {
				tt.setup(h)
			}

			_, err := sessionscmd.NewBindSession(h.Console, tt.pipeline, "10.0.0.8:5001", "bind-01")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("NewBindSession error = %v, want substring %q", err, tt.wantErr)
			}
			testsupport.RequireNoPrimaryCalls(t, h)
		})
	}
}

func bindPipeline(name, listenerID string) *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Type:       consts.BindPipeline,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{Name: name},
		},
	}
}

func prepareRegisterResponse(h *testsupport.Harness) {
	h.Recorder.OnEmpty("Register", func(_ context.Context, request any) (*clientpb.Empty, error) {
		req := request.(*clientpb.RegisterSession)
		h.SetSessionResponse(&clientpb.Session{
			SessionId:  req.GetSessionId(),
			RawId:      req.GetRawId(),
			PipelineId: req.GetPipelineId(),
			ListenerId: req.GetListenerId(),
			Target:     req.GetTarget(),
			Type:       req.GetType(),
			Timer:      req.GetRegisterData().GetTimer(),
			Data:       "null",
		})
		return &clientpb.Empty{}, nil
	})
}

func recordedRegisterRequest(t testing.TB, h *testsupport.Harness) *clientpb.RegisterSession {
	t.Helper()
	for _, call := range h.Recorder.Calls() {
		if call.Method == "Register" {
			req, ok := call.Request.(*clientpb.RegisterSession)
			if !ok {
				t.Fatalf("register request type = %T, want *clientpb.RegisterSession", call.Request)
			}
			return req
		}
	}
	t.Fatal("Register was not called")
	return nil
}
