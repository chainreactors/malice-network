package pipeline_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	pipelinecmd "github.com/chainreactors/malice-network/client/command/pipeline"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/spf13/cobra"
)

func TestNewWebShellCmdStoresParamsInCustomPipeline(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	cmd := newWebShellTestCommand(t, "--listener", "listener-a", "--suo5", "suo5://target/bridge.php", "--token", "secret123", "ws-a")
	if err := pipelinecmd.NewWebShellCmd(cmd, h.Console); err != nil {
		t.Fatalf("NewWebShellCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(calls))
	}
	if calls[0].Method != "RegisterPipeline" {
		t.Fatalf("first method = %s, want RegisterPipeline", calls[0].Method)
	}

	req, ok := calls[0].Request.(*clientpb.Pipeline)
	if !ok {
		t.Fatalf("register request type = %T, want *clientpb.Pipeline", calls[0].Request)
	}
	if req.Type != "webshell" {
		t.Fatalf("pipeline type = %q, want %q", req.Type, "webshell")
	}
	custom, ok := req.Body.(*clientpb.Pipeline_Custom)
	if !ok {
		t.Fatalf("register pipeline body = %T, want *clientpb.Pipeline_Custom", req.Body)
	}

	var params struct {
		Suo5URL    string `json:"suo5_url"`
		StageToken string `json:"stage_token"`
	}
	if err := json.Unmarshal([]byte(custom.Custom.Params), &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}
	if params.Suo5URL != "suo5://target/bridge.php" {
		t.Fatalf("suo5_url = %q, want %q", params.Suo5URL, "suo5://target/bridge.php")
	}
	if params.StageToken != "secret123" {
		t.Fatalf("stage_token = %q, want %q", params.StageToken, "secret123")
	}
}

func TestNewWebShellCmdRequiresSuo5Flag(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	cmd := newWebShellTestCommand(t, "--listener", "listener-b", "ws-b")
	err := pipelinecmd.NewWebShellCmd(cmd, h.Console)
	if err == nil {
		t.Fatal("NewWebShellCmd error = nil, want error")
	}
	if !strings.Contains(err.Error(), "--suo5") {
		t.Fatalf("error = %q, want suo5 requirement", err)
	}
}

func TestNewWebShellCmdWrapsRegisterError(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Recorder.OnEmpty("RegisterPipeline", func(_ context.Context, _ any) (*clientpb.Empty, error) {
		return nil, errors.New("listener not found")
	})

	cmd := newWebShellTestCommand(t, "--listener", "listener-c", "--suo5", "suo5://target/x.php", "--token", "secret", "ws-c")
	err := pipelinecmd.NewWebShellCmd(cmd, h.Console)
	if err == nil {
		t.Fatal("NewWebShellCmd error = nil, want error")
	}
	if !strings.Contains(err.Error(), "register webshell pipeline") {
		t.Fatalf("error = %q, want register error", err)
	}
}

func TestStartWebShellCmdRejectsNonWebShellPipeline(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["tcp-a"] = &clientpb.Pipeline{
		Name:       "tcp-a",
		ListenerId: "listener-a",
		Type:       "tcp",
	}

	cmd := newWebShellTestCommand(t, "tcp-a")
	err := pipelinecmd.StartWebShellCmd(cmd, h.Console)
	if err == nil {
		t.Fatal("StartWebShellCmd error = nil, want error")
	}
	if !strings.Contains(err.Error(), "pipeline tcp-a is type tcp, not webshell") {
		t.Fatalf("error = %q, want pipeline type validation", err)
	}
	if calls := h.Recorder.Calls(); len(calls) != 0 {
		t.Fatalf("call count = %d, want 0", len(calls))
	}
}

func TestStopWebShellCmdResolvesListenerAndStopsMatchingPipeline(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Recorder.OnPipelines("ListPipelines", func(_ context.Context, in any) (*clientpb.Pipelines, error) {
		listener, ok := in.(*clientpb.Listener)
		if !ok {
			t.Fatalf("request type = %T, want *clientpb.Listener", in)
		}
		if listener.GetId() != "listener-z" {
			t.Fatalf("listener id = %q, want %q", listener.GetId(), "listener-z")
		}
		return &clientpb.Pipelines{
			Pipelines: []*clientpb.Pipeline{{
				Name:       "ws-z",
				ListenerId: "listener-z",
				Type:       "webshell",
			}},
		}, nil
	})

	cmd := newWebShellTestCommand(t, "--listener", "listener-z", "ws-z")
	if err := pipelinecmd.StopWebShellCmd(cmd, h.Console); err != nil {
		t.Fatalf("StopWebShellCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(calls))
	}
	if calls[1].Method != "StopPipeline" {
		t.Fatalf("second method = %s, want StopPipeline", calls[1].Method)
	}
	req, ok := calls[1].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("stop request type = %T, want *clientpb.CtrlPipeline", calls[1].Request)
	}
	if req.GetListenerId() != "listener-z" {
		t.Fatalf("stop listener_id = %q, want %q", req.GetListenerId(), "listener-z")
	}
}

func newWebShellTestCommand(t *testing.T, args ...string) *cobra.Command {
	t.Helper()

	cmd := &cobra.Command{}
	cmd.Flags().StringP("listener", "l", "", "listener id")
	cmd.Flags().String("suo5", "", "suo5 URL")
	cmd.Flags().String("token", "", "stage token")
	cmd.Flags().String("dll", "", "DLL path")
	cmd.Flags().String("deps", "", "deps directory")
	if err := cmd.Flags().Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return cmd
}
