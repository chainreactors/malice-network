//go:build integration

package pipeline

import (
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
)

func TestNewTcpPipelineCmdIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	tcpCmd := mustCommand(t, Commands(clientHarness.Console), consts.CommandPipelineTcp)
	parseArgs(t, tcpCmd, "tcp-integration-cmd", "--listener", h.ListenerID(), "--host", "127.0.0.1")

	if err := NewTcpPipelineCmd(tcpCmd, clientHarness.Console); err != nil {
		t.Fatalf("NewTcpPipelineCmd failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		_, ok := clientHarness.Console.Pipelines["tcp-integration-cmd"]
		return ok
	}, "client pipeline cache to include started tcp pipeline")

	model, err := h.GetPipeline("tcp-integration-cmd", h.ListenerID())
	if err != nil {
		t.Fatalf("GetPipeline failed: %v", err)
	}
	if !model.Enable {
		t.Fatalf("expected pipeline to be enabled in db")
	}

	history := h.ControlHistory()
	if len(history) == 0 {
		t.Fatal("expected controller history to contain start control")
	}
	last := history[len(history)-1]
	if last.Ctrl != consts.CtrlPipelineStart {
		t.Fatalf("last ctrl = %s, want %s", last.Ctrl, consts.CtrlPipelineStart)
	}
	if last.GetJob().GetPipeline().GetListenerId() != h.ListenerID() {
		t.Fatalf("start ctrl listener_id = %s, want %s", last.GetJob().GetPipeline().GetListenerId(), h.ListenerID())
	}
}

func TestTcpCommandSmokeIntegration(t *testing.T) {
	h := testsupport.NewControlPlaneHarness(t)
	clientHarness := testsupport.NewClientHarness(t, h)

	tcpCmd := mustCommand(t, Commands(clientHarness.Console), consts.CommandPipelineTcp)
	tcpCmd.SetArgs([]string{"tcp-smoke", "--listener", h.ListenerID(), "--host", "127.0.0.1"})

	if err := tcpCmd.Execute(); err != nil {
		t.Fatalf("tcp command execute failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		model, err := h.GetPipeline("tcp-smoke", h.ListenerID())
		return err == nil && model.Enable
	}, "smoke tcp pipeline to be enabled")
}

func mustCommand(t testing.TB, commands []*cobra.Command, name string) *cobra.Command {
	t.Helper()

	for _, cmd := range commands {
		if cmd.Name() == name || strings.Split(cmd.Use, " ")[0] == name {
			return cmd
		}
	}
	t.Fatalf("command %q not found", name)
	return nil
}

func parseArgs(t testing.TB, cmd *cobra.Command, args ...string) {
	t.Helper()

	if err := cmd.ParseFlags(args); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}
}
