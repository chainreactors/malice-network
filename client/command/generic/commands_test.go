package generic_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/chainreactors/malice-network/helper/utils/output"
)

func TestGenericCommandConformance(t *testing.T) {
	testsupport.RunClientCases(t, []testsupport.CommandCase{
		{
			Name: "version requests basic info",
			Argv: []string{consts.CommandVersion},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.MustSingleCall[*clientpb.Empty](t, h, "GetBasic")
			},
		},
		{
			Name: "version propagates server error",
			Argv: []string{consts.CommandVersion},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnBasic("GetBasic", func(_ context.Context, _ any) (*clientpb.Basic, error) {
					return nil, errors.New("basic unavailable")
				})
			},
			WantErr: "basic unavailable",
		},
		{
			Name: "broadcast sends broadcast event",
			Argv: []string{consts.CommandBroadcast, "hello", "operators"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Event](t, h, "Broadcast")
				if req.Type != consts.EventBroadcast || string(req.Message) != "hello operators" {
					t.Fatalf("broadcast event = %#v", req)
				}
			},
		},
		{
			Name: "broadcast notify sends notify event",
			Argv: []string{consts.CommandBroadcast, "--notify", "hello", "operators"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Event](t, h, "Notify")
				if req.Type != consts.EventNotify || string(req.Message) != "hello operators" {
					t.Fatalf("notify event = %#v", req)
				}
			},
		},
		{
			Name: "broadcast propagates rpc errors",
			Argv: []string{consts.CommandBroadcast, "hello"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnEmpty("Broadcast", func(_ context.Context, _ any) (*clientpb.Empty, error) {
					return nil, errors.New("broadcast failed")
				})
			},
			WantErr: "broadcast failed",
		},
		{
			Name: "license requests server license",
			Argv: []string{consts.CommandLicense},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.MustSingleCall[*clientpb.Empty](t, h, "GetLicenseInfo")
			},
		},
		{
			Name: "pivot filters contexts by pivot type",
			Argv: []string{consts.CommandPivot, "--all"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContexts("GetContexts", func(_ context.Context, _ any) (*clientpb.Contexts, error) {
					return &clientpb.Contexts{
						Contexts: []*clientpb.Context{
							{
								Type:  consts.ContextPivoting,
								Value: (&output.PivotingContext{Enable: true, Listener: "listener-1", Pipeline: "pipe-1", RemAgentID: "agent-1", LocalURL: "tcp://127.0.0.1:8080", RemoteURL: "tcp://10.0.0.2:8080", InboundSide: "local"}).Marshal(),
							},
						},
					}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Context](t, h, "GetContexts")
				if req.Type != consts.ContextPivoting {
					t.Fatalf("pivot context filter = %#v, want type %q", req, consts.ContextPivoting)
				}
			},
		},
		{
			Name: "pivot list filters contexts by pivot type",
			Argv: []string{consts.CommandPivot, "list", "--all"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContexts("GetContexts", func(_ context.Context, _ any) (*clientpb.Contexts, error) {
					return &clientpb.Contexts{}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.Context](t, h, "GetContexts")
				if req.Type != consts.ContextPivoting {
					t.Fatalf("pivot context filter = %#v, want type %q", req, consts.ContextPivoting)
				}
			},
		},
		{
			Name: "pivot status sends rem_dial status to owning session",
			Argv: []string{consts.CommandPivot, "status", "agent-1"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContexts("GetContexts", func(_ context.Context, _ any) (*clientpb.Contexts, error) {
					return &clientpb.Contexts{
						Contexts: []*clientpb.Context{
							pivotContextFixture("session-owner", "pipe-1", "agent-1", true),
						},
					}, nil
				})
				h.AddSession(t, "session-owner")
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				assertPivotRemDial(t, h, "status", "agent-1", "listener-1:pipe-1", "session-owner")
			},
		},
		{
			Name: "pivot stop sends rem_dial stop to owning session",
			Argv: []string{consts.CommandPivot, "stop", "agent-1"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContexts("GetContexts", func(_ context.Context, _ any) (*clientpb.Contexts, error) {
					return &clientpb.Contexts{
						Contexts: []*clientpb.Context{
							pivotContextFixture("session-owner", "pipe-1", "agent-1", true),
						},
					}, nil
				})
				h.AddSession(t, "session-owner")
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				assertPivotRemDial(t, h, "stop", "agent-1", "listener-1:pipe-1", "session-owner")
			},
		},
		{
			Name: "pivot log reads rem agent log",
			Argv: []string{consts.CommandPivot, "log", "agent-1"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContexts("GetContexts", func(_ context.Context, _ any) (*clientpb.Contexts, error) {
					return &clientpb.Contexts{
						Contexts: []*clientpb.Context{
							pivotContextFixture("session-owner", "pipe-1", "agent-1", true),
						},
					}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				calls := h.Recorder.Calls()
				if len(calls) != 2 {
					t.Fatalf("primary call count = %d, want 2", len(calls))
				}
				if calls[1].Method != "RemAgentLog" {
					t.Fatalf("second method = %s, want RemAgentLog", calls[1].Method)
				}
				req, ok := calls[1].Request.(*clientpb.REMAgent)
				if !ok {
					t.Fatalf("RemAgentLog request type = %T", calls[1].Request)
				}
				if req.Id != "agent-1" || req.PipelineId != "listener-1:pipe-1" {
					t.Fatalf("RemAgentLog request = %#v", req)
				}
			},
		},
		{
			Name:    "pivot status requires unambiguous agent",
			Argv:    []string{consts.CommandPivot, "status", "agent-1"},
			WantErr: "multiple pivots match agent agent-1",
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnContexts("GetContexts", func(_ context.Context, _ any) (*clientpb.Contexts, error) {
					return &clientpb.Contexts{
						Contexts: []*clientpb.Context{
							pivotContextFixture("session-owner", "pipe-1", "agent-1", true),
							pivotContextFixture("session-owner", "pipe-2", "agent-1", true),
						},
					}, nil
				})
			},
		},
	})
}

func pivotContextFixture(sessionID, pipelineID, agentID string, enabled bool) *clientpb.Context {
	return &clientpb.Context{
		Type: consts.ContextPivoting,
		Session: &clientpb.Session{
			SessionId: sessionID,
		},
		Value: (&output.PivotingContext{
			Enable:      enabled,
			Listener:    "listener-1",
			Pipeline:    pipelineID,
			RemAgentID:  agentID,
			LocalURL:    "tcp://127.0.0.1:8080",
			RemoteURL:   "tcp://10.0.0.2:8080",
			InboundSide: "local",
		}).Marshal(),
	}
}

func assertPivotRemDial(t testing.TB, h *testsupport.Harness, action, agentID, pipelineID, sessionID string) {
	t.Helper()

	calls := h.Recorder.Calls()
	if len(calls) != 2 {
		t.Fatalf("primary call count = %d, want 2", len(calls))
	}
	if calls[0].Method != "GetContexts" || calls[1].Method != "RemDial" {
		t.Fatalf("calls = %s, %s; want GetContexts, RemDial", calls[0].Method, calls[1].Method)
	}
	req, ok := calls[1].Request.(*implantpb.Request)
	if !ok {
		t.Fatalf("RemDial request type = %T", calls[1].Request)
	}
	if req.Name != consts.ModuleRemDial {
		t.Fatalf("RemDial name = %q, want %q", req.Name, consts.ModuleRemDial)
	}
	if strings.Join(req.Args, " ") != action+" "+agentID {
		t.Fatalf("RemDial args = %v, want [%s %s]", req.Args, action, agentID)
	}
	if req.Params["pipeline_id"] != pipelineID {
		t.Fatalf("RemDial pipeline = %q, want %q", req.Params["pipeline_id"], pipelineID)
	}
	if got := calls[1].Metadata.Get("session_id"); len(got) != 1 || got[0] != sessionID {
		t.Fatalf("RemDial metadata session_id = %v, want %s", got, sessionID)
	}
}
