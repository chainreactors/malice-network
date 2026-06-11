package ai

import (
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

// Commands returns AI interaction commands (ask, analyze).
// The ai-config command lives under `config ai`.
func Commands(con *core.Console) []*cobra.Command {
	askCmd := &cobra.Command{
		Use:   "ask [question]",
		Short: "Ask the AI assistant a question",
		Long:  "Ask the AI assistant a question with command history context. This is equivalent to using '? <question>' syntax.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return AskCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
// Ask about commands
ask how do I list all sessions

// Ask about current target
ask what commands can I run on this target

// Ask with no history context
ask --no-history how to download a file
~~~`,
	}

	askCmd.Flags().Int("history", 20, "Number of history lines to include as context")
	askCmd.Flags().Bool("no-history", false, "Don't include command history in context")

	questionCmd := &cobra.Command{
		Use:    "? [question]",
		Short:  "Ask the AI assistant (shortcut)",
		Long:   "Ask the AI assistant a question. This is equivalent to using '? <question>' syntax or the 'ask' command.",
		Args:   cobra.MinimumNArgs(1),
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return AskCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}

	analyzeCmd := &cobra.Command{
		Use:   "analyze [error message]",
		Short: "AI-powered error analysis and suggestions",
		Long:  "Analyze an error message using AI and get suggestions for resolution, including possible causes and alternative commands.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return AnalyzeCmd(cmd, con, args)
		},
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
// Analyze an error message
analyze Access denied when trying to read file

// Analyze with more context
analyze "Error: permission denied for /etc/shadow"

// Analyze a command failure
analyze "getsystem failed: UAC is enabled"
~~~`,
	}

	return []*cobra.Command{askCmd, questionCmd, analyzeCmd}
}

// AIConfigCommand returns the ai subcommand for use under `config`.
func AIConfigCommand(con *core.Console) *cobra.Command {
	aiCmd := &cobra.Command{
		Use:   "ai",
		Short: "Show local AI preferences",
		Long: `config ai manages local AI preferences on the client.
Agent chat/skill uses provider/model from this local config, while endpoint/api_key/proxy
are resolved on the server from server/config.yaml -> server.llm.
Legacy local ask/analyze can still use local endpoint/api_key overrides if configured.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return AIShowCmd(con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
// Show current AI configuration
config ai

// Enable local preferences for server-backed agent chat/skill
config ai enable --provider openai --model gpt-5.4

// Switch local provider/model preference
config ai enable --provider claude --model claude-3-5-sonnet

// Disable AI
config ai disable
~~~`,
	}

	enableCmd := &cobra.Command{
		Use:   "enable",
		Short: "Enable local AI preferences",
		Long: `Enable local AI preferences for agent chat/skill.
Provider/model are stored on the client. Endpoint/api_key for the agent pipeline are read
from server/config.yaml -> server.llm. Legacy local ask/analyze can still use local overrides.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return AIEnableCmd(cmd, con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}
	enableCmd.Flags().String("provider", "", "Preferred provider for agent chat/skill: openai or claude")
	enableCmd.Flags().String("api-key", "", "Legacy local API key for direct ask/analyze only")
	enableCmd.Flags().String("endpoint", "", "Legacy local API endpoint for direct ask/analyze only")
	enableCmd.Flags().String("model", "", "Preferred model name for agent chat/skill")
	enableCmd.Flags().Int("max-tokens", 0, "Maximum tokens in response")
	enableCmd.Flags().Int("timeout", 0, "Request timeout in seconds")
	enableCmd.Flags().Int("history-size", 0, "Number of history lines to include as context")
	enableCmd.Flags().Bool("opsec-check", false, "Enable AI OPSEC risk assessment for high-risk commands")

	disableCmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable AI assistant",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AIDisableCmd(con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
	}

	aiCmd.AddCommand(enableCmd, disableCmd)
	return aiCmd
}
