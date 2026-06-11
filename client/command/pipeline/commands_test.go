package pipeline

import (
	"context"
	"testing"

	iomclient "github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/IoM-go/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func TestCommandsExposeExpectedPipelineRoots(t *testing.T) {
	cmds := Commands(&core.Console{})
	if len(cmds) != 4 {
		t.Fatalf("pipeline command roots = %d, want 4", len(cmds))
	}

	want := map[string]bool{
		consts.CommandPipelineTcp:  true,
		consts.HTTPPipeline:        true,
		consts.CommandPipelineBind: true,
		consts.CommandRem:          true,
	}
	for _, cmd := range cmds {
		delete(want, cmd.Name())
	}
	if len(want) != 0 {
		t.Fatalf("missing pipeline roots: %#v", want)
	}
}

func TestCommandsExposeRemUpdateIntervalSubcommand(t *testing.T) {
	var remCmdName string
	for _, cmd := range Commands(&core.Console{}) {
		if cmd.Name() == consts.CommandRem {
			remCmdName = cmd.Name()
			updateCmd, _, err := cmd.Find([]string{"update", "interval"})
			if err != nil {
				t.Fatalf("expected rem update interval command under %s: %v", remCmdName, err)
			}
			if updateCmd == nil || updateCmd.Name() != "interval" {
				t.Fatalf("unexpected rem update interval command: %#v", updateCmd)
			}
			return
		}
	}
	t.Fatalf("rem command %q not found", consts.CommandRem)
}

func TestRemUpdateIntervalCmdUsesScopedSessionPipeline(t *testing.T) {
	rpc := &remCommandTestRPC{}
	state := &iomclient.ServerState{
		Rpc: &iomclient.Rpc{
			MaliceRPCClient:   rpc,
			ListenerRPCClient: rpc,
		},
		Client:    &clientpb.Client{Name: "tester", ID: 1},
		Pipelines: map[string]*clientpb.Pipeline{},
		Sessions:  map[string]*iomclient.Session{},
	}
	con := &core.Console{
		Server: &core.Server{ServerState: state},
		Log:    iomclient.Log,
	}
	session := &iomclient.Session{Session: &clientpb.Session{
		SessionId:  "session-a",
		PipelineId: "rem-a",
		ListenerId: "listener-a",
	}}
	con.Sessions[session.SessionId] = session
	con.Pipelines["listener-a:rem-a"] = &clientpb.Pipeline{
		Name:       "rem-a",
		ListenerId: "listener-a",
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name: "rem-a",
				Agents: map[string]*clientpb.REMAgent{
					"agent-a": {Id: "agent-a"},
				},
			},
		},
	}

	cmd := &cobra.Command{}
	cmd.Flags().String("session-id", "", "")
	cmd.Flags().String("pipeline-id", "", "")
	cmd.Flags().String("agent-id", "", "")
	if err := cmd.Flags().Parse([]string{"--session-id", session.SessionId, "750"}); err != nil {
		t.Fatalf("Parse flags failed: %v", err)
	}
	if err := RemUpdateIntervalCmd(cmd, con); err != nil {
		t.Fatalf("RemUpdateIntervalCmd failed: %v", err)
	}
	if rpc.request == nil || rpc.request.PipelineId != "listener-a:rem-a" || rpc.request.Id != "agent-a" {
		t.Fatalf("rem request = %#v, want listener-a:rem-a agent-a", rpc.request)
	}
}

type remCommandTestRPC struct {
	clientrpc.MaliceRPCClient
	listenerrpc.ListenerRPCClient

	request *clientpb.REMAgent
}

func (r *remCommandTestRPC) RemAgentCtrl(ctx context.Context, in *clientpb.REMAgent, opts ...grpc.CallOption) (*clientpb.Empty, error) {
	r.request = in
	return &clientpb.Empty{}, nil
}
