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
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

const ModuleChat = "chat"

const bridgeChatTargetPrefix = "llm-agent://"

type chatBackend int

const (
	chatBackendDedicated chatBackend = iota
	chatBackendBridge
)

type ChatOptions struct {
	Text     string
	Model    string
	Provider string
	MaxTurns uint32
}

func hasModule(sess *client.Session, name string) bool {
	if sess == nil {
		return false
	}
	for _, mod := range sess.Modules {
		if mod == name {
			return true
		}
	}
	return false
}

func chatBackendForSession(sess *client.Session) chatBackend {
	if sess != nil && strings.HasPrefix(strings.ToLower(sess.Target), bridgeChatTargetPrefix) {
		return chatBackendBridge
	}
	return chatBackendDedicated
}

func loadChatSettings() (*assets.AISettings, error) {
	settings, err := assets.LoadSettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load settings: %w", err)
	}
	if settings == nil || settings.AI == nil || !settings.AI.Enable {
		return nil, fmt.Errorf("AI is not enabled. Use 'config ai enable' to enable it")
	}
	return settings.AI, nil
}

func ChatCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	session := con.GetInteractive()
	opts, err := chatOptionsFromCommand(cmd, session, strings.Join(args, " "))
	if err != nil {
		return err
	}
	task, err := DispatchChat(con.Rpc, session, opts)
	if err != nil {
		return err
	}
	session.Console(task, "chat")
	return nil
}

func Chat(rpc clientrpc.MaliceRPCClient, sess *client.Session, text string) (*clientpb.Task, error) {
	opts, err := defaultChatOptions(sess, text)
	if err != nil {
		return nil, err
	}
	return DispatchChat(rpc, sess, opts)
}

func DispatchChat(rpc clientrpc.MaliceRPCClient, sess *client.Session, opts ChatOptions) (*clientpb.Task, error) {
	if chatBackendForSession(sess) == chatBackendBridge {
		return ExecuteBridgeChat(rpc, sess, opts.Text)
	}
	return BridgeAgentChat(rpc, sess, opts.Text, opts.Model, opts.Provider, opts.MaxTurns)
}

func defaultChatOptions(sess *client.Session, text string) (ChatOptions, error) {
	opts := ChatOptions{Text: text}
	if chatBackendForSession(sess) == chatBackendBridge {
		return opts, nil
	}
	aiSettings, err := loadChatSettings()
	if err != nil {
		return ChatOptions{}, err
	}
	opts.Model = aiSettings.Model
	opts.Provider = aiSettings.Provider
	return opts, nil
}

func chatOptionsFromCommand(cmd *cobra.Command, sess *client.Session, text string) (ChatOptions, error) {
	opts, err := defaultChatOptions(sess, text)
	if err != nil {
		return ChatOptions{}, err
	}
	if chatBackendForSession(sess) == chatBackendDedicated {
		if v, _ := cmd.Flags().GetString("model"); v != "" {
			opts.Model = v
		}
		if v, _ := cmd.Flags().GetString("provider"); v != "" {
			opts.Provider = v
		}
		opts.MaxTurns, _ = cmd.Flags().GetUint32("max-turns")
	}
	return opts, nil
}

func RegisterChatFunc(con *core.Console) {
	con.RegisterImplantFunc(
		ModuleChat,
		Chat,
		"",
		nil,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			return formatChatTask(ctx)
		},
		nil,
	)

	_ = intermediate.RegisterInternalDoneCallback(ModuleChat, formatChatTask)

	_ = con.AddCommandFuncHelper(
		ModuleChat,
		ModuleChat,
		ModuleChat+`(active(), "List files in the current directory")`,
		[]string{
			"sess: special session",
			"text: message to send",
		},
		[]string{"task"},
	)
}

func formatChatTask(ctx *clientpb.TaskContext) (string, error) {
	if ctx == nil || ctx.Spite == nil {
		return "", nil
	}
	if resp := ctx.Spite.GetBridgeAgentResponse(); resp != nil {
		return formatBridgeAgentResponse(resp), nil
	}
	if ev := ctx.Spite.GetLlmEvent(); ev != nil {
		return formatLLMEvent(ev), nil
	}
	if ctx.Spite.GetExecResponse() != nil {
		execOut, err := output.ParseExecResponse(ctx)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%v", execOut), nil
	}
	return "", nil
}

func formatBridgeAgentResponse(resp *implantpb.BridgeAgentResponse) string {
	var sb strings.Builder
	if resp.Error != "" {
		sb.WriteString("ERROR: " + resp.Error + "\n")
		return sb.String()
	}
	sb.WriteString(resp.Text + "\n")
	sb.WriteString(fmt.Sprintf("--- %d iterations, %d tool calls ---\n", resp.Iterations, resp.ToolCallsMade))
	if len(resp.AvailableTools) > 0 {
		names := make([]string, len(resp.AvailableTools))
		for i, t := range resp.AvailableTools {
			names[i] = t.Name
		}
		sb.WriteString("tools: " + strings.Join(names, ", ") + "\n")
	}
	return sb.String()
}
