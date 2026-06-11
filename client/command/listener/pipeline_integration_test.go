//go:build integration

package listener

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
)

func TestListPipelineCmdIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-live"), true)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-stopped"), false)
	h.SeedPipeline(t, h.NewBindPipeline(t, "bind-live"), true)
	h.SeedPipeline(t, h.NewREMPipeline("rem-live", "tcp://127.0.0.1:19971"), true)
	clientHarness := testsupport.NewClientHarness(t, h)

	listCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineList)
	parseSubcommandArgs(t, listCmd)

	var err error
	output := testsupport.CaptureOutput(func() {
		err = ListPipelineCmd(listCmd, clientHarness.Console)
	})
	if err != nil {
		t.Fatalf("ListPipelineCmd failed: %v", err)
	}

	if !strings.Contains(output, "tcp-live") || !strings.Contains(output, "tcp-stopped") {
		t.Fatalf("pipeline list output missing expected names:\n%s", output)
	}
	if !strings.Contains(output, "bind-live") || !strings.Contains(output, "rem-live") {
		t.Fatalf("pipeline list output missing bind/rem names:\n%s", output)
	}
	if !strings.Contains(output, h.ListenerID()) {
		t.Fatalf("pipeline list output missing listener id:\n%s", output)
	}
	if !strings.Contains(output, consts.BindPipeline) || !strings.Contains(output, consts.RemPipeline) {
		t.Fatalf("pipeline list output missing bind/rem types:\n%s", output)
	}
}

func TestStartPipelineCmdStopsEnabledPipelineBeforeRestart(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-restart"), true)
	clientHarness := testsupport.NewClientHarness(t, h)

	startCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineStart)
	parseSubcommandArgs(t, startCmd, "tcp-restart")
	before := len(h.ControlHistory())

	if err := StartPipelineCmd(startCmd, clientHarness.Console); err != nil {
		t.Fatalf("StartPipelineCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		return len(h.ControlHistory()) >= before+2
	}, "stop-then-start controller history")

	history := h.ControlHistory()[before:]
	if history[0].Ctrl != consts.CtrlPipelineStop || history[1].Ctrl != consts.CtrlPipelineStart {
		t.Fatalf("unexpected ctrl sequence: %s then %s", history[0].Ctrl, history[1].Ctrl)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		model, err := h.GetPipeline("tcp-restart", h.ListenerID())
		return err == nil && model.Enable
	}, "pipeline to remain enabled after restart")
}

func TestStopPipelineCmdUsesDatabaseResolution(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-stop"), true)
	clientHarness := testsupport.NewClientHarness(t, h)

	stopCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineStop)
	parseSubcommandArgs(t, stopCmd, "tcp-stop")

	if err := StopPipelineCmd(stopCmd, clientHarness.Console); err != nil {
		t.Fatalf("StopPipelineCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, ok := clientHarness.Console.Pipelines["tcp-stop"]
		return !ok
	}, "client pipeline cache to remove stopped pipeline")

	model, err := h.GetPipeline("tcp-stop", h.ListenerID())
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}
	if model.Enable {
		t.Fatal("expected pipeline to be disabled")
	}
}

func TestDeletePipelineCmdUsesDatabaseResolution(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-delete"), true)
	clientHarness := testsupport.NewClientHarness(t, h)

	deleteCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineDelete)
	parseSubcommandArgs(t, deleteCmd, "tcp-delete")

	if err := DeletePipelineCmd(deleteCmd, clientHarness.Console); err != nil {
		t.Fatalf("DeletePipelineCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := h.GetPipeline("tcp-delete", h.ListenerID())
		return err != nil
	}, "pipeline record to be removed")
}

func TestStopPipelineCmdIsIdempotentWhenAlreadyStopped(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-already-stopped"), false)
	clientHarness := testsupport.NewClientHarness(t, h)

	stopCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineStop)
	parseSubcommandArgs(t, stopCmd, "tcp-already-stopped")

	if err := StopPipelineCmd(stopCmd, clientHarness.Console); err != nil {
		t.Fatalf("StopPipelineCmd on stopped pipeline failed: %v", err)
	}

	model, err := h.GetPipeline("tcp-already-stopped", h.ListenerID())
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}
	if model.Enable {
		t.Fatal("expected stopped pipeline to remain disabled")
	}
}

func TestDeletePipelineCmdDeletesStoppedPipelineWithoutRuntime(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-delete-stopped"), false)
	clientHarness := testsupport.NewClientHarness(t, h)

	deleteCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineDelete)
	parseSubcommandArgs(t, deleteCmd, "tcp-delete-stopped")

	if err := DeletePipelineCmd(deleteCmd, clientHarness.Console); err != nil {
		t.Fatalf("DeletePipelineCmd on stopped pipeline failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, err := h.GetPipeline("tcp-delete-stopped", h.ListenerID())
		return err != nil
	}, "stopped pipeline to be deleted")
}

func TestStopPipelineCmdPropagatesListenerFailure(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-stop-fail"), true)
	clientHarness := testsupport.NewClientHarness(t, h)
	h.FailNextCtrl(consts.CtrlPipelineStop, "tcp-stop-fail", errors.New("listener stop failed"))

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, ok := clientHarness.Console.Pipelines["tcp-stop-fail"]
		return ok
	}, "client cache to load pipeline before stop failure")

	stopCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineStop)
	parseSubcommandArgs(t, stopCmd, "tcp-stop-fail")

	err := StopPipelineCmd(stopCmd, clientHarness.Console)
	if err == nil || !strings.Contains(err.Error(), "listener stop failed") {
		t.Fatalf("StopPipelineCmd error = %v, want listener failure", err)
	}

	model, getErr := h.GetPipeline("tcp-stop-fail", h.ListenerID())
	if getErr != nil {
		t.Fatalf("GetPipeline failed: %v", getErr)
	}
	if !model.Enable {
		t.Fatal("expected pipeline to remain enabled after failed stop")
	}
	if !h.JobExists("tcp-stop-fail", h.ListenerID()) {
		t.Fatal("expected runtime job to remain after failed stop")
	}
}

func TestDeletePipelineCmdPropagatesListenerFailure(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "tcp-delete-fail"), true)
	clientHarness := testsupport.NewClientHarness(t, h)
	h.FailNextCtrl(consts.CtrlPipelineStop, "tcp-delete-fail", errors.New("listener delete stop failed"))

	deleteCmd := mustSubcommand(t, mustRootCommand(t, Commands(clientHarness.Console), consts.CommandPipeline), consts.CommandPipelineDelete)
	parseSubcommandArgs(t, deleteCmd, "tcp-delete-fail")

	err := DeletePipelineCmd(deleteCmd, clientHarness.Console)
	if err == nil || !strings.Contains(err.Error(), "listener delete stop failed") {
		t.Fatalf("DeletePipelineCmd error = %v, want listener failure", err)
	}

	model, getErr := h.GetPipeline("tcp-delete-fail", h.ListenerID())
	if getErr != nil {
		t.Fatalf("GetPipeline failed: %v", getErr)
	}
	if !model.Enable {
		t.Fatal("expected pipeline to remain enabled after failed delete")
	}
	if !h.JobExists("tcp-delete-fail", h.ListenerID()) {
		t.Fatal("expected runtime job to remain after failed delete")
	}
}

func mustRootCommand(t testing.TB, commands []*cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range commands {
		if cmd.Name() == name || strings.Split(cmd.Use, " ")[0] == name {
			return cmd
		}
	}
	t.Fatalf("root command %q not found", name)
	return nil
}

func mustSubcommand(t testing.TB, root *cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range root.Commands() {
		if cmd.Name() == name || strings.Split(cmd.Use, " ")[0] == name {
			return cmd
		}
	}
	t.Fatalf("subcommand %q not found under %q", name, root.Name())
	return nil
}

func parseSubcommandArgs(t testing.TB, cmd *cobra.Command, args ...string) {
	t.Helper()

	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
}
