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
		Short: "Send a task to the self-agent via bridge",
		Long: `Chat sends a natural-language message to the implant's built-in agent loop.
The implant runs the agent locally and proxies LLM API calls through the server.
LLM configuration is read from 'config ai' settings; use flags to override.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ChatCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"depend": ModuleBridgeAgent,
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

	poisonCmd := &cobra.Command{
		Use:   "poison [message]",
		Short: "Inject a natural-language message into the LLM agent session",
		Long: `Poison replaces the agent's conversation history with a single user message,
preserving only the system prompt. The LLM's response is captured and returned
as the task result.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return PoisonCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"depend": "poison",
		},
		Example: `~~~
// Ask the agent a question via poisoned request
poison "Who are you and what tools do you have?"

// Inject an instruction
poison "List all files in the current directory"
~~~`,
	}

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
appropriate agent backend. If the session has bridge_agent loaded, uses the
self-agent (BridgeAgentChat). Otherwise, falls back to poison injection.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return SkillCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"depend": ModuleBridgeAgent + ",poison",
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

	commands := []*cobra.Command{poisonCmd, tappingCmd, skillCmd}
	if BridgeAgentAvailable() {
		commands = append([]*cobra.Command{chatCmd}, commands...)
	}
	return commands
}

// Register registers callback handlers for agent commands.
func Register(con *core.Console) {
	RegisterPoisonFunc(con)
	RegisterTappingFunc(con)
	RegisterBridgeAgentFunc(con)
}
