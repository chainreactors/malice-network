package agent

import (
	"fmt"
	"strings"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/utils/output"
	"github.com/spf13/cobra"
)

const (
	ModuleTapping    = "tapping"
	ModuleTappingOff = "tapping_off"
)

// TappingCmd handles the tapping command from the CLI.
func TappingCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	task, err := Tapping(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(task, "tapping")
	return nil
}

// TappingOffCmd handles the "tapping off" command from the CLI.
func TappingOffCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	task, err := TappingOff(con.Rpc, session)
	if err != nil {
		return err
	}
	session.Console(task, "tapping off")
	return nil
}

// Tapping sends a tapping request to the CLIProxyAPI bridge via ExecuteModule.
// The bridge acknowledges the module; observe events are continuously forwarded
// and displayed via the DoneCallback.
func Tapping(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
	task, err := rpc.ExecuteModule(sess.Context(), &implantpb.ExecuteModuleRequest{
		Spite: &implantpb.Spite{
			Name: ModuleTapping,
			Body: &implantpb.Spite_Request{
				Request: &implantpb.Request{Name: ModuleTapping},
			},
		},
		Expect: "llm.observe",
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// TappingOff sends a tapping_off request to stop observe event forwarding.
func TappingOff(rpc clientrpc.MaliceRPCClient, sess *client.Session) (*clientpb.Task, error) {
	task, err := rpc.ExecuteModule(sess.Context(), &implantpb.ExecuteModuleRequest{
		Spite: &implantpb.Spite{
			Name: ModuleTappingOff,
			Body: &implantpb.Spite_Request{
				Request: &implantpb.Request{Name: ModuleTappingOff},
			},
		},
		Expect: consts.ModuleExecute,
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

// RegisterTappingFunc registers the tapping command's DoneCallback for parsing
// LLMEvent spites and formatting them as readable output.
func RegisterTappingFunc(con *core.Console) {
	con.RegisterImplantFunc(
		ModuleTapping,
		Tapping,
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

	intermediate.RegisterInternalDoneCallback(ModuleTapping, func(ctx *clientpb.TaskContext) (string, error) {
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
		ModuleTapping,
		ModuleTapping,
		ModuleTapping+`(active())`,
		[]string{
			"sess: special session",
		},
		[]string{"task"},
	)

	con.RegisterImplantFunc(
		ModuleTappingOff,
		TappingOff,
		"",
		nil,
		output.ParseExecResponse,
		nil,
	)
}

const (
	maxContentRunes = 120
	maxArgsRunes    = 80
)

// truncateText collapses whitespace and truncates to maxRunes.
func truncateText(s string, maxRunes int) string {
	s = strings.Join(strings.Fields(s), " ")
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes]) + "..."
	}
	return s
}

// eventSummary builds a compact type summary for the header line.
// Response: "text", "⚡bash", "text ⚡bash ⚡Read", etc.
// Request:  "user", "↩result", "user ↩result", etc.
func eventSummary(ev *implantpb.LLMEvent) string {
	var parts []string

	if ev.Type == "response" {
		for _, msg := range ev.Messages {
			if msg.Role == "assistant" && strings.TrimSpace(msg.Content) != "" {
				parts = append(parts, "text")
				break
			}
		}
		for _, tc := range ev.ToolCalls {
			parts = append(parts, "⚡"+tc.Name)
		}
	} else {
		for _, msg := range ev.Messages {
			if msg.Role == "user" {
				parts = append(parts, "user")
				break
			}
		}
		if len(ev.ToolResults) > 0 {
			parts = append(parts, "↩result")
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return " | " + strings.Join(parts, " ")
}

// formatLLMEvent renders a structured LLMEvent as a concise human-readable string.
func formatLLMEvent(ev *implantpb.LLMEvent) string {
	var s strings.Builder
	summary := eventSummary(ev)

	switch ev.Type {
	case "request":
		s.WriteString(fmt.Sprintf("◀ REQ %s [%d msgs]%s\n", ev.Model, ev.MessageCount, summary))
	case "response":
		s.WriteString(fmt.Sprintf("▶ RSP %s%s\n", ev.Model, summary))
	default:
		s.WriteString(fmt.Sprintf("● %s %s%s\n", ev.Type, ev.Model, summary))
	}

	// Track which tool_call IDs already appear as messages to avoid duplicates
	toolResultShown := make(map[string]bool)

	for _, msg := range ev.Messages {
		if msg.Role == "system" {
			continue
		}
		content := truncateText(msg.Content, maxContentRunes)
		if content == "" {
			continue
		}
		if msg.Role == "tool" {
			// Will be shown via ToolResults below
			continue
		}
		if msg.Role == "assistant" && ev.Type == "response" {
			s.WriteString(fmt.Sprintf("  %s\n", content))
		} else {
			s.WriteString(fmt.Sprintf("  %s: %s\n", msg.Role, content))
		}
	}

	for _, tc := range ev.ToolCalls {
		args := truncateText(tc.Arguments, maxArgsRunes)
		s.WriteString(fmt.Sprintf("  ⚡ %s(%s)\n", tc.Name, args))
	}

	for _, tr := range ev.ToolResults {
		if toolResultShown[tr.CallId] {
			continue
		}
		toolResultShown[tr.CallId] = true
		content := truncateText(tr.Content, maxContentRunes)
		if content == "" {
			continue
		}
		s.WriteString(fmt.Sprintf("  ↩ %s\n", content))
	}

	return strings.TrimRight(s.String(), "\n")
}
