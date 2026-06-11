package agent

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

// Commands returns all LLM agent-related commands.
func Commands(con *core.Console) []*cobra.Command {
	chatCmd := &cobra.Command{
		Use:   "chat [message]",
		Short: "Send a natural-language task to the active chat backend",
		Long: `Chat sends a natural-language message to the active chat backend.
For implant self-agent sessions, the implant runs the agent locally and the server proxies
LLM API calls on its behalf. For hijacked bridge sessions, the request is injected into the
existing agent conversation. Flags override config ai only for the self-agent path.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ChatCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"depend": ModuleChat,
		},
		Example: `~~~
// Ask the agent to list files
chat "list all files in current directory"

// Override model
chat -m gpt-4o "do a network scan"

// Override provider
chat -p deepseek "enumerate running processes"
~~~`,
	}
	chatCmd.Flags().StringP("model", "m", "", "LLM model name (overrides config ai)")
	chatCmd.Flags().StringP("provider", "p", "", "LLM provider (overrides config ai)")
	chatCmd.Flags().Uint32("max-turns", 0, "Max agent loop iterations (0 = default)")

	tappingCmd := &cobra.Command{
		Use:   "tapping",
		Short: "Stream real-time LLM interaction events from the agent session",
		Long: `Tapping activates real-time monitoring of an LLM agent session.
Parsed LLM events (messages, tool calls, tool results) are displayed
as they occur, showing the model name, message count, and content.
Use "tapping off" to stop streaming.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return TappingCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": "tapping",
		},
		Example: `~~~
// Start streaming LLM events from the active session
tapping

// Stop streaming
tapping off
~~~`,
	}

	tappingOffCmd := &cobra.Command{
		Use:   "off",
		Short: "Stop streaming LLM events",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return TappingOffCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": "tapping",
		},
	}
	tappingCmd.AddCommand(tappingOffCmd)

	skillCmd := &cobra.Command{
		Use:   "skill <name> [arguments...]",
		Short: "Execute a skill from skills/ directory",
		Long: `Load a SKILL.md file from skills/ directory and execute it via the
appropriate chat backend. Skills are a client-side Markdown wrapper around chat.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return SkillCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"depend": ModuleChat,
		},
		Example: `~~~
// List available skills
skill list

// Execute a skill
skill recon

// Execute a skill with arguments
skill recon "web servers"
~~~`,
	}

	skillListCmd := &cobra.Command{
		Use:   "list",
		Short: "List all available skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return SkillListCmd(cmd, con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}
	skillCmd.AddCommand(skillListCmd)

	common.BindArgCompletions(skillCmd, nil, SkillNameCompleter())

	schemaCmd := &cobra.Command{
		Use:   "schema",
		Short: "List observed tool schemas from the LLM agent session",
		Long: `Schema queries the EvilClaw bridge for all tool schemas observed
in the active session. These are the tools the LLM agent has registered
(Bash, Read, Write, WebFetch, Grep, etc.) — use tool_call to invoke any of them.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return SchemaCmd(cmd, con)
		},
		Annotations: map[string]string{
			"depend": ModuleSchema,
		},
		Example: `~~~
// List all tools the LLM agent has registered
schema
~~~`,
	}

	toolCallCmd := &cobra.Command{
		Use:   "tool_call <tool_name> [json_args]",
		Short: "Inject an arbitrary tool call into the LLM agent session",
		Long: `Tool_call injects a fabricated tool call into the active LLM agent session.
The tool name must match one of the observed schemas (use "schema" to list them).
Arguments are passed as a JSON object. The bridge validates required fields
against the observed schema before injection.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ToolCallCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"depend": ModuleToolInject,
		},
		Example: `~~~
// Read a file
tool_call Read '{"file_path": "/etc/passwd"}'

// Execute a shell command
tool_call Bash '{"command": "id"}'

// Fetch a URL
tool_call WebFetch '{"url": "http://169.254.169.254/latest/meta-data/", "prompt": "list all"}'

// Search code
tool_call Grep '{"pattern": "password", "path": "/app"}'
~~~`,
	}

	return []*cobra.Command{chatCmd, tappingCmd, skillCmd, schemaCmd, toolCallCmd}
}

// Register registers callback handlers for agent commands.
func Register(con *core.Console) {
	RegisterChatFunc(con)
	RegisterTappingFunc(con)
	RegisterSchemaFunc(con)
	RegisterToolCallFunc(con)
}
