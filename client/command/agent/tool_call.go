package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

const ModuleToolInject = "tool_inject"

func ToolCallCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	session := con.GetInteractive()
	toolName := args[0]
	argsJSON := "{}"
	if len(args) > 1 {
		argsJSON = strings.Join(args[1:], " ")
	}

	if !json.Valid([]byte(argsJSON)) {
		return fmt.Errorf("invalid JSON arguments: %s", argsJSON)
	}

	task, err := ToolCall(con.Rpc, session, toolName, argsJSON)
	if err != nil {
		return err
	}
	session.Console(task, "tool_call "+toolName)
	return nil
}

func ToolCall(rpc clientrpc.MaliceRPCClient, sess *client.Session,
	toolName, argsJSON string) (*clientpb.Task, error) {
	task, err := rpc.ExecuteModule(sess.Context(), &implantpb.ExecuteModuleRequest{
		Spite: &implantpb.Spite{
			Name: ModuleToolInject,
			Body: &implantpb.Spite_Request{
				Request: &implantpb.Request{
					Name:  ModuleToolInject,
					Input: toolName,
					Args:  []string{argsJSON},
				},
			},
		},
		Expect: "response",
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func RegisterToolCallFunc(con *core.Console) {
	con.RegisterImplantFunc(
		ModuleToolInject,
		func(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
			return nil, fmt.Errorf("tool_call requires tool name and arguments")
		},
		"",
		nil,
		output.ParseResponse,
		nil,
	)

	_ = intermediate.RegisterInternalDoneCallback(ModuleToolInject, func(ctx *clientpb.TaskContext) (string, error) {
		if ctx == nil || ctx.Spite == nil {
			return "", fmt.Errorf("no response")
		}
		resp := ctx.Spite.GetResponse()
		if resp == nil {
			return "", fmt.Errorf("no response")
		}
		return resp.GetOutput(), nil
	})

	_ = con.AddCommandFuncHelper(
		ModuleToolInject,
		ModuleToolInject,
		ModuleToolInject+`(active(), "Bash", '{"command":"id"}')`,
		[]string{
			"sess: special session",
			"tool_name: name of the tool to invoke",
			"args_json: JSON arguments for the tool",
		},
		[]string{"task"},
	)
}
