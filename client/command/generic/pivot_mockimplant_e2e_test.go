//go:build mockimplant

package generic_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/command/generic"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/chainreactors/malice-network/server/testsupport"
	"github.com/spf13/cobra"
)

func TestPivotManagementCommandsWithMockImplant(t *testing.T) {
	fixture := testsupport.NewMockRPCFixture(t, "pivot-e2e-tcp")
	fixture.Mock.On(consts.ModuleRemDial, func(_ context.Context, req *clientpb.SpiteRequest, send func(*implantpb.Spite) error) error {
		action := ""
		if args := req.GetSpite().GetRequest().GetArgs(); len(args) > 0 {
			action = args[0]
		}
		outputText := "running"
		if action == "stop" {
			outputText = "stopped"
		}
		return send(&implantpb.Spite{
			Body: &implantpb.Spite_Response{
				Response: &implantpb.Response{Output: outputText},
			},
		})
	})

	remPipe := fixture.H.SeedPipeline(t, fixture.H.NewREMPipeline("pivot-e2e-rem", "tcp://127.0.0.1:19980"), true)
	agentID := "agent-pivot-e2e"
	addPivotContext(t, fixture, remPipe, agentID)

	clientHarness := testsupport.NewClientHarness(t, fixture.H)
	clientHarness.Console.NewConsole()
	clientHarness.Console.App.SwitchMenu(consts.ClientMenu)

	assertPivotCommand := func(action, wantOutput string) {
		t.Helper()

		before := len(fixture.Mock.RequestsByName(consts.ModuleRemDial))
		cmd := newPivotCommandForTest(clientHarness.Console, action+" "+agentID+" -p "+fixture.H.ListenerID()+":"+remPipe.Name)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("pivot %s failed: %v", action, err)
		}

		req := testsupport.WaitModuleRequest(t, fixture.Mock, consts.ModuleRemDial, before)
		request := req.GetSpite().GetRequest()
		if request == nil {
			t.Fatalf("pivot %s delivered nil rem_dial request", action)
		}
		if got, want := request.GetArgs(), []string{action, agentID}; !reflect.DeepEqual(got, want) {
			t.Fatalf("pivot %s args = %v, want %v", action, got, want)
		}
		if len(request.GetParams()) != 0 {
			t.Fatalf("pivot %s implant request params = %v, want empty params", action, request.GetParams())
		}

		taskContext := testsupport.WaitTaskFinish(t, fixture.RPC, fixture.Mock.SessionID, req.GetTask().GetTaskId())
		if got := taskContext.GetSpite().GetResponse().GetOutput(); got != wantOutput {
			t.Fatalf("pivot %s output = %q, want %q", action, got, wantOutput)
		}
	}

	assertPivotCommand("status", "running")
	assertPivotCommand("stop", "stopped")

	contexts, err := fixture.RPC.GetContexts(context.Background(), &clientpb.Context{Type: consts.ContextPivoting})
	if err != nil {
		t.Fatalf("GetContexts failed: %v", err)
	}
	if len(contexts.GetContexts()) != 1 {
		t.Fatalf("pivot context count = %d, want 1", len(contexts.GetContexts()))
	}
}

func addPivotContext(t testing.TB, fixture *testsupport.MockRPCFixture, remPipe *clientpb.Pipeline, agentID string) {
	t.Helper()

	_, err := fixture.RPC.AddContext(context.Background(), &clientpb.Context{
		Type:     consts.ContextPivoting,
		Session:  &clientpb.Session{SessionId: fixture.Mock.SessionID},
		Pipeline: remPipe,
		Value: (&output.PivotingContext{
			Enable:      true,
			Listener:    fixture.H.ListenerID(),
			Pipeline:    remPipe.GetName(),
			RemAgentID:  agentID,
			LocalURL:    "socks5://127.0.0.1:1080",
			RemoteURL:   "raw://10.0.0.8:8080",
			InboundSide: "remote",
		}).Marshal(),
	})
	if err != nil {
		t.Fatalf("AddContext failed: %v", err)
	}

	testsupport.WaitForCondition(t, 5*time.Second, func() bool {
		contexts, err := fixture.RPC.GetContexts(context.Background(), &clientpb.Context{Type: consts.ContextPivoting})
		return err == nil && len(contexts.GetContexts()) == 1
	}, "pivot context to be visible")
}

func newPivotCommandForTest(con *core.Console, argv string) *cobra.Command {
	cmd := generic.PivotCommand(con)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(strings.Fields(argv))
	con.App.Shell().Line().Set([]rune("pivot " + argv)...)
	return cmd
}
