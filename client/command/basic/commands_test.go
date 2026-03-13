package basic_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"google.golang.org/grpc/metadata"
)

func TestBasicCommandConformance(t *testing.T) {
	testsupport.RunCases(t, []testsupport.CommandCase{
		{
			Name: "sleep normalizes seconds and reuses session jitter",
			Argv: []string{consts.ModuleSleep, "30"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Timer](t, h, "Sleep")
				if req.Expression != "*/30 * * * * * *" {
					t.Fatalf("sleep expression = %q, want normalized seconds expression", req.Expression)
				}
				if req.Jitter != h.Session.Timer.Jitter {
					t.Fatalf("sleep jitter = %v, want %v", req.Jitter, h.Session.Timer.Jitter)
				}
				assertTaskEvent(t, h, md, consts.ModuleSleep)
			},
		},
		{
			Name:    "sleep rejects invalid cron expression",
			Argv:    []string{consts.ModuleSleep, "not-a-cron"},
			WantErr: "Invalid cron expression",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "keepalive parses enable alias",
			Argv: []string{consts.ModuleKeepalive, "enable"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.CommonBody](t, h, "Keepalive")
				if len(req.BoolArray) != 1 || !req.BoolArray[0] {
					t.Fatalf("keepalive request = %#v, want enable=true", req)
				}
				assertTaskEvent(t, h, md, consts.ModuleKeepalive)
			},
		},
		{
			Name:    "keepalive rejects invalid argument",
			Argv:    []string{consts.ModuleKeepalive, "maybe"},
			WantErr: "invalid argument",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "suicide sends module request",
			Argv: []string{consts.ModuleSuicide},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Request](t, h, "Suicide")
				if req.Name != consts.ModuleSuicide {
					t.Fatalf("suicide request name = %q, want %q", req.Name, consts.ModuleSuicide)
				}
				assertTaskEvent(t, h, md, consts.ModuleSuicide)
			},
		},
		{
			Name: "ping emits a non-zero nonce",
			Argv: []string{consts.ModulePing},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Ping](t, h, "Ping")
				if req.Nonce == 0 {
					t.Fatal("ping nonce = 0, want randomized non-zero nonce")
				}
				assertTaskEvent(t, h, md, consts.ModulePing)
			},
		},
		{
			Name: "wait forwards task id and session id",
			Argv: []string{consts.CommandWait, "42"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*clientpb.Task](t, h, "WaitTaskFinish")
				if req.TaskId != 42 {
					t.Fatalf("wait task id = %d, want 42", req.TaskId)
				}
				if req.SessionId != h.Session.SessionId {
					t.Fatalf("wait session id = %q, want %q", req.SessionId, h.Session.SessionId)
				}
				testsupport.RequireSessionID(t, md, h.Session.SessionId)
				testsupport.RequireCallee(t, md, consts.CalleeCMD)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "wait returns rpc errors instead of dereferencing nil content",
			Argv: []string{consts.CommandWait, "42"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnTaskContext("WaitTaskFinish", func(ctx context.Context, request any) (*clientpb.TaskContext, error) {
					return nil, errors.New("rpc wait failed")
				})
			},
			WantErr: "rpc wait failed",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				_, md := testsupport.MustSingleCall[*clientpb.Task](t, h, "WaitTaskFinish")
				testsupport.RequireSessionID(t, md, h.Session.SessionId)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "polling uses seconds as interval",
			Argv: []string{consts.CommandPolling, "--interval", "2"},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*clientpb.Polling](t, h, "Polling")
				if req.SessionId != h.Session.SessionId {
					t.Fatalf("polling session id = %q, want %q", req.SessionId, h.Session.SessionId)
				}
				if req.Interval != uint64(2*time.Second) {
					t.Fatalf("polling interval = %d, want %d", req.Interval, uint64(2*time.Second))
				}
				if !req.Force {
					t.Fatal("polling force = false, want true")
				}
				testsupport.RequireSessionID(t, md, h.Session.SessionId)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "init bind session forwards raw session bytes",
			Argv: []string{consts.ModuleInit},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Init](t, h, "InitBindSession")
				want := testsupport.SessionRaw(h.Session.RawId)
				if string(req.Data) != string(want) {
					t.Fatalf("init raw bytes = %v, want %v", req.Data, want)
				}
				testsupport.RequireSessionID(t, md, h.Session.SessionId)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "recover refreshes the cached session",
			Argv: []string{consts.CommandRecover},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				updated := testsupport.SessionClone(h.Session)
				updated.Note = "recovered-note"
				h.SetSessionResponse(updated)
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.SessionRequest](t, h, "GetSession")
				if req.SessionId != h.Session.SessionId {
					t.Fatalf("recover session id = %q, want %q", req.SessionId, h.Session.SessionId)
				}
				if h.Session.Note != "recovered-note" {
					t.Fatalf("session note = %q, want recovered-note", h.Session.Note)
				}
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
		{
			Name: "switch combines pipeline and explicit address",
			Argv: []string{consts.ModuleSwitch, "--pipeline", "tcp-a", "--address", "10.0.0.2:9443"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.AddTCPPipeline("tcp-a", "127.0.0.1", 8443)
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, md := testsupport.MustSingleCall[*implantpb.Switch](t, h, "Switch")
				want := []string{"127.0.0.1:8443", "10.0.0.2:9443"}
				if len(req.Urls) != len(want) {
					t.Fatalf("switch urls = %v, want %v", req.Urls, want)
				}
				for i := range want {
					if req.Urls[i] != want[i] {
						t.Fatalf("switch urls = %v, want %v", req.Urls, want)
					}
				}
				assertTaskEvent(t, h, md, consts.ModuleSwitch)
			},
		},
		{
			Name:    "switch rejects unknown pipeline",
			Argv:    []string{consts.ModuleSwitch, "--pipeline", "missing"},
			WantErr: "no such pipeline",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
				testsupport.RequireNoSessionEvents(t, h)
			},
		},
	})
}

func assertTaskEvent(t testing.TB, h *testsupport.Harness, md metadata.MD, wantType string) {
	t.Helper()

	testsupport.RequireSessionID(t, md, h.Session.SessionId)
	testsupport.RequireCallee(t, md, consts.CalleeCMD)

	event, eventMD := testsupport.MustSingleSessionEvent(t, h)
	if event.Op != consts.CtrlSessionTask {
		t.Fatalf("session event op = %q, want %q", event.Op, consts.CtrlSessionTask)
	}
	if event.Task == nil || event.Task.Type != wantType {
		t.Fatalf("session event task = %#v, want task type %q", event.Task, wantType)
	}
	testsupport.RequireSessionID(t, eventMD, h.Session.SessionId)
	testsupport.RequireCallee(t, eventMD, consts.CalleeCMD)
}
