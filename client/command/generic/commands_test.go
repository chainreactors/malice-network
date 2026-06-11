package generic_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
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
	})
}
