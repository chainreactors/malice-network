//go:build integration

package pipeline

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
)

func TestNewHTTPPipelineCmdIntegrationPreservesErrorPageContent(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	errorPagePath, err := h.WriteTempFile("error.html", []byte("<h1>boom</h1>"))
	if err != nil {
		t.Fatalf("WriteTempFile failed: %v", err)
	}

	httpCmd := mustCommand(t, Commands(clientHarness.Console), consts.HTTPPipeline)
	parseArgs(t, httpCmd, "http-integration-cmd", "--listener", h.ListenerID(), "--host", "127.0.0.1", "--headers", "X-Test=ok", "--error-page", errorPagePath)

	if err := NewHttpPipelineCmd(httpCmd, clientHarness.Console); err != nil {
		t.Fatalf("NewHttpPipelineCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, ok := clientHarness.Console.Pipelines["http-integration-cmd"]
		return ok
	}, "client pipeline cache to include started http pipeline")

	model, err := h.GetPipeline("http-integration-cmd", h.ListenerID())
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}
	if !model.Enable {
		t.Fatal("expected http pipeline to be enabled")
	}

	params, err := implanttypes.UnmarshalPipelineParams(model.GetHttp().GetParams())
	if err != nil {
		t.Fatalf("UnmarshalPipelineParams failed: %v", err)
	}
	if params.ErrorPage != "<h1>boom</h1>" {
		t.Fatalf("error page = %q, want %q", params.ErrorPage, "<h1>boom</h1>")
	}
	if got := params.Headers["X-Test"]; len(got) != 1 || got[0] != "ok" {
		t.Fatalf("headers = %#v, want X-Test=ok", params.Headers)
	}
}

func TestNewBindPipelineCmdGeneratesNameWhenOmitted(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	bindCmd := mustCommand(t, Commands(clientHarness.Console), consts.CommandPipelineBind)
	parseArgs(t, bindCmd, "--listener", h.ListenerID())

	if err := NewBindPipelineCmd(bindCmd, clientHarness.Console); err != nil {
		t.Fatalf("NewBindPipelineCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return len(h.ControlHistory()) > 0
	}, "bind pipeline start control")

	last := h.ControlHistory()[len(h.ControlHistory())-1]
	if last.Ctrl != consts.CtrlPipelineStart {
		t.Fatalf("last ctrl = %s, want %s", last.Ctrl, consts.CtrlPipelineStart)
	}
	name := last.GetJob().GetPipeline().GetName()
	if name == "" || !strings.HasPrefix(name, "bind_"+h.ListenerID()+"_") {
		t.Fatalf("generated bind name = %q", name)
	}

	model, err := h.GetPipeline(name, h.ListenerID())
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}
	if !model.Enable || model.Type != consts.BindPipeline {
		t.Fatalf("bind pipeline = %#v, want enabled bind pipeline", model)
	}
}

func TestListRemCmdIntegrationShowsEnabledRemsOnly(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewREMPipeline("rem-enabled", "tcp://127.0.0.1:19966"), true)
	h.SeedPipeline(t, h.NewREMPipeline("rem-disabled", "tcp://127.0.0.1:19967"), false)
	clientHarness := testsupport.NewClientHarness(t, h)

	listCmd := mustPipelineSubcommand(t, mustCommand(t, Commands(clientHarness.Console), consts.CommandRem), consts.CommandListRem)
	parseArgs(t, listCmd, h.ListenerID())

	var err error
	output := testsupport.CaptureOutput(func() {
		err = ListRemCmd(listCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("ListRemCmd failed: %v", err)
	}
	if !strings.Contains(output, "rem-enabled") {
		t.Fatalf("enabled rem missing from output:\n%s", output)
	}
	if strings.Contains(output, "rem-disabled") {
		t.Fatalf("disabled rem should not be listed:\n%s", output)
	}
}

func TestNewRemCmdGeneratesNameWhenOmitted(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	newCmd := mustPipelineSubcommand(t, mustCommand(t, Commands(clientHarness.Console), consts.CommandRem), consts.CommandRemNew)
	parseArgs(t, newCmd, "--listener", h.ListenerID(), "--console", "tcp://127.0.0.1:19966")

	if err := NewRemCmd(newCmd, clientHarness.Console); err != nil {
		t.Fatalf("NewRemCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return len(h.ControlHistory()) > 0
	}, "rem start control")

	last := h.ControlHistory()[len(h.ControlHistory())-1]
	if last.Ctrl != consts.CtrlRemStart {
		t.Fatalf("last ctrl = %s, want %s", last.Ctrl, consts.CtrlRemStart)
	}
	name := last.GetJob().GetPipeline().GetName()
	if name == "" || !strings.HasPrefix(name, "rem_"+h.ListenerID()+"_") {
		t.Fatalf("generated rem name = %q", name)
	}

	model, err := h.GetPipeline(name, h.ListenerID())
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}
	if !model.Enable || model.Type != consts.RemPipeline {
		t.Fatalf("rem pipeline = %#v, want enabled rem pipeline", model)
	}
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		pipe, ok := clientHarness.Console.Pipelines[name]
		return ok && pipe.Enable
	}, "client pipeline cache to include started rem pipeline")
}

func TestStartRemCmdUsesDatabaseResolution(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewREMPipeline("rem-start", "tcp://127.0.0.1:19968"), false)
	clientHarness := testsupport.NewClientHarness(t, h)

	startCmd := mustPipelineSubcommand(t, mustCommand(t, Commands(clientHarness.Console), consts.CommandRem), consts.CommandRemStart)
	parseArgs(t, startCmd, "rem-start")

	if err := StartRemCmd(startCmd, clientHarness.Console); err != nil {
		t.Fatalf("StartRemCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		model, err := h.GetPipeline("rem-start", h.ListenerID())
		return err == nil && model.Enable
	}, "rem pipeline to be enabled")
	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		pipe, ok := clientHarness.Console.Pipelines["rem-start"]
		return ok && pipe.Enable
	}, "client cache to mark rem pipeline enabled")
}

func TestStopRemCmdPropagatesListenerFailure(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewREMPipeline("rem-stop-fail", "tcp://127.0.0.1:19969"), true)
	clientHarness := testsupport.NewClientHarness(t, h)
	h.FailNextCtrl(consts.CtrlRemStop, "rem-stop-fail", errors.New("rem stop failed"))

	stopCmd := mustPipelineSubcommand(t, mustCommand(t, Commands(clientHarness.Console), consts.CommandRem), consts.CommandRemStop)
	parseArgs(t, stopCmd, "rem-stop-fail")

	err := StopRemCmd(stopCmd, clientHarness.Console)
	if err == nil || !strings.Contains(err.Error(), "rem stop failed") {
		t.Fatalf("StopRemCmd error = %v, want listener failure", err)
	}

	model, getErr := h.GetPipeline("rem-stop-fail", h.ListenerID())
	if getErr != nil {
		t.Fatalf("GetPipeline failed: %v", getErr)
	}
	if !model.Enable {
		t.Fatal("expected rem pipeline to remain enabled after failed stop")
	}
	if !h.JobExists("rem-stop-fail", h.ListenerID()) {
		t.Fatal("expected runtime rem job to remain after failed stop")
	}
}

func TestDeleteRemCmdPropagatesListenerFailure(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewREMPipeline("rem-delete-fail", "tcp://127.0.0.1:19970"), true)
	clientHarness := testsupport.NewClientHarness(t, h)
	h.FailNextCtrl(consts.CtrlRemStop, "rem-delete-fail", errors.New("rem delete failed"))

	deleteCmd := mustPipelineSubcommand(t, mustCommand(t, Commands(clientHarness.Console), consts.CommandRem), consts.CommandPipelineDelete)
	parseArgs(t, deleteCmd, "rem-delete-fail")

	err := DeleteRemCmd(deleteCmd, clientHarness.Console)
	if err == nil || !strings.Contains(err.Error(), "rem delete failed") {
		t.Fatalf("DeleteRemCmd error = %v, want listener failure", err)
	}

	model, getErr := h.GetPipeline("rem-delete-fail", h.ListenerID())
	if getErr != nil {
		t.Fatalf("GetPipeline failed: %v", getErr)
	}
	if !model.Enable {
		t.Fatal("expected rem pipeline to remain enabled after failed delete")
	}
	if !h.JobExists("rem-delete-fail", h.ListenerID()) {
		t.Fatal("expected runtime rem job to remain after failed delete")
	}
}

func mustPipelineSubcommand(t testing.TB, root *cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range root.Commands() {
		if cmd.Name() == name || strings.Split(cmd.Use, " ")[0] == name {
			return cmd
		}
	}
	t.Fatalf("subcommand %q not found under %q", name, root.Name())
	return nil
}
