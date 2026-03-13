package agent

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
)

const ModulePoison = "poison"

// PoisonCmd handles the poison command from the CLI.
func PoisonCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	session := con.GetInteractive()
	text := strings.Join(args, " ")
	task, err := Poison(con.Rpc, session, text)
	if err != nil {
		return err
	}
	session.Console(task, string(*con.App.Shell().Line()))
	return nil
}

// Poison sends a poison request to the CLIProxyAPI bridge via ExecuteModule.
// The bridge replaces the LLM agent's conversation history with the given text,
// then streams back all observe events (the full multi-turn conversation) as LLMEvents.
func Poison(rpc clientrpc.MaliceRPCClient, sess *client.Session, text string) (*clientpb.Task, error) {
	task, err := rpc.ExecuteModule(sess.Context(), &implantpb.ExecuteModuleRequest{
		Spite: &implantpb.Spite{
			Name: ModulePoison,
			Body: &implantpb.Spite_Request{
				Request: &implantpb.Request{
					Name:  ModulePoison,
					Input: text,
				},
			},
		},
		Expect: "llm.observe",
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// RegisterPoisonFunc registers the poison command's output parser and helper.
func RegisterPoisonFunc(con *core.Console) {
	con.RegisterImplantFunc(
		ModulePoison,
		Poison,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			if ctx == nil || ctx.Spite == nil {
				return "", nil
			}
			ev := ctx.Spite.GetLlmEvent()
			if ev == nil {
				return "", nil
			}
			return formatLLMEvent(ev), nil
		},
		nil,
	)

	intermediate.RegisterInternalDoneCallback(ModulePoison, func(ctx *clientpb.TaskContext) (string, error) {
		if ctx == nil || ctx.Spite == nil {
			return "", fmt.Errorf("no response")
		}
		ev := ctx.Spite.GetLlmEvent()
		if ev == nil {
			return "", nil
		}
		return formatLLMEvent(ev), nil
	})

	con.AddCommandFuncHelper(
		ModulePoison,
		ModulePoison,
		ModulePoison+`(active(), "What tools do you have?")`,
		[]string{
			"sess: special session",
			"text: message to inject",
		},
		[]string{"task"},
	)
}
