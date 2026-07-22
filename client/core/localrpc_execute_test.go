package core_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	localrpcpb "github.com/chainreactors/IoM-go/proto/services/localrpc"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/server/testsupport"
)

func newLocalRPCExecutionConsole(t *testing.T) (*core.Console, string) {
	t.Helper()

	h := testsupport.NewControlPlaneHarness(t)
	h.SeedPipeline(t, h.NewTCPPipeline(t, "rpc-pipe"), true)
	sess := h.SeedSession(t, "rpc123", "rpc-pipe", true)
	clientHarness := testsupport.NewClientHarness(t, h)

	con := clientHarness.Console
	con.NewConsole()
	con.App.Menu(consts.ClientMenu).Command = command.BindClientsCommands(con)()
	con.App.Menu(consts.ImplantMenu).Command = command.BindImplantCommands(con)()
	con.App.SwitchMenu(consts.ClientMenu)

	return con, sess.ID
}

func TestLocalRPCExecuteCommandReturnsStaticOutputWithoutSession(t *testing.T) {
	con, _ := newLocalRPCExecutionConsole(t)
	server := core.NewLocalRPCServer(con)

	resp, err := server.ExecuteCommand(context.Background(), &localrpcpb.ExecuteCommandRequest{
		Command: "session --all",
	})
	if err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("ExecuteCommand failed: %s", resp.Error)
	}
	if !strings.Contains(resp.Output, "rpc123") {
		t.Fatalf("session output = %q, want it to contain %q", resp.Output, "rpc123")
	}
}

func TestLocalRPCExecuteCommandReturnsHelpAndPreservesConsoleOutput(t *testing.T) {
	con, _ := newLocalRPCExecutionConsole(t)
	server := core.NewLocalRPCServer(con)

	root := con.App.ActiveMenu().Command
	var consoleOutput bytes.Buffer
	originalOut := root.OutOrStdout()
	originalErr := root.ErrOrStderr()
	root.SetOut(&consoleOutput)
	root.SetErr(&consoleOutput)
	t.Cleanup(func() {
		root.SetOut(originalOut)
		root.SetErr(originalErr)
	})

	resp, err := server.ExecuteCommand(context.Background(), &localrpcpb.ExecuteCommandRequest{
		Command: "session --help",
	})
	if err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("ExecuteCommand failed: %s", resp.Error)
	}

	for _, want := range []string{"session", "Usage", "--all"} {
		if !strings.Contains(resp.Output, want) {
			t.Errorf("RPC output = %q, want it to contain %q", resp.Output, want)
		}
		if !strings.Contains(consoleOutput.String(), want) {
			t.Errorf("console output = %q, want it to contain %q", consoleOutput.String(), want)
		}
	}
	if strings.Count(resp.Output, "\n") < 2 {
		t.Errorf("RPC output = %q, want multiline help", resp.Output)
	}
	if root.OutOrStdout() != &consoleOutput || root.ErrOrStderr() != &consoleOutput {
		t.Fatal("ExecuteCommand did not restore the command output writers")
	}
}

func TestLocalRPCExecuteCommandReturnsClientOutputWhenNoTaskIsCreated(t *testing.T) {
	con, sessionID := newLocalRPCExecutionConsole(t)
	server := core.NewLocalRPCServer(con)

	resp, err := server.ExecuteCommand(context.Background(), &localrpcpb.ExecuteCommandRequest{
		Command:   "listener",
		SessionId: sessionID,
	})
	if err != nil {
		t.Fatalf("ExecuteCommand returned error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("ExecuteCommand failed: %s", resp.Error)
	}
	if !strings.Contains(resp.Output, "fixture-listener") {
		t.Fatalf("listener output = %q, want it to contain %q", resp.Output, "fixture-listener")
	}
}
