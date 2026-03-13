//go:build bridge_agent_proto
// +build bridge_agent_proto

package agent

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
)

const ModuleBridgeAgent = "bridge_agent"

// ChatCmd handles the chat top-level command. It reads LLM config from config ai.
func ChatCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	session := con.GetInteractive()
	text := strings.Join(args, " ")

	aiSettings, err := assets.GetValidAISettings()
	if err != nil {
		return err
	}

	// Optional flag overrides
	model := aiSettings.Model
	if v, _ := cmd.Flags().GetString("model"); v != "" {
		model = v
	}
	provider := aiSettings.Provider
	if v, _ := cmd.Flags().GetString("provider"); v != "" {
		provider = v
	}
	maxTurns, _ := cmd.Flags().GetUint32("max-turns")

	task, err := BridgeAgentChat(con.Rpc, session, text, model, provider,
		aiSettings.APIKey, aiSettings.Endpoint, maxTurns)
	if err != nil {
		return err
	}
	session.Console(task, "chat")
	return nil
}

// BridgeAgentChat sends a BridgeAgentRequest carrying the LLM config from config ai.
func BridgeAgentChat(rpc clientrpc.MaliceRPCClient, sess *client.Session,
	text, model, provider, apiKey, endpoint string, maxTurns uint32) (*clientpb.Task, error) {
	task, err := rpc.BridgeAgentChat(sess.Context(), &implantpb.BridgeAgentRequest{
		Text:     text,
		Model:    model,
		Provider: provider,
		ApiKey:   apiKey,
		Endpoint: endpoint,
		MaxTurns: maxTurns,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// RegisterBridgeAgentFunc registers the output callback for BridgeAgentResponse.
func RegisterBridgeAgentFunc(con *core.Console) {
	con.RegisterImplantFunc(
		ModuleBridgeAgent,
		nil,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			if ctx == nil || ctx.Spite == nil {
				return "", nil
			}
			resp := ctx.Spite.GetBridgeAgentResponse()
			if resp == nil {
				return "", nil
			}
			return formatBridgeAgentResponse(resp), nil
		},
		nil,
	)

	intermediate.RegisterInternalDoneCallback(ModuleBridgeAgent, func(ctx *clientpb.TaskContext) (string, error) {
		if ctx == nil || ctx.Spite == nil {
			return "", fmt.Errorf("no response")
		}
		resp := ctx.Spite.GetBridgeAgentResponse()
		if resp == nil {
			return "", nil
		}
		return formatBridgeAgentResponse(resp), nil
	})
}

func formatBridgeAgentResponse(resp *implantpb.BridgeAgentResponse) string {
	var sb strings.Builder
	if resp.Error != "" {
		fmt.Fprintf(&sb, "ERROR: %s\n", resp.Error)
		return sb.String()
	}
	fmt.Fprintf(&sb, "%s\n", resp.Text)
	fmt.Fprintf(&sb, "--- %d iterations, %d tool calls ---\n", resp.Iterations, resp.ToolCallsMade)
	if len(resp.AvailableTools) > 0 {
		names := make([]string, len(resp.AvailableTools))
		for i, t := range resp.AvailableTools {
			names[i] = t.Name
		}
		fmt.Fprintf(&sb, "tools: %s\n", strings.Join(names, ", "))
	}
	return sb.String()
}

func BridgeAgentAvailable() bool {
	return true
}
