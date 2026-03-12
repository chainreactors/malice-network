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
		Short: "Show and update AI assistant configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return AIConfigCmd(cmd, con)
		},
		Annotations: map[string]string{
			"static": "true",
		},
		Example: `~~~
// Enable AI with OpenAI
config ai --enable --provider openai --api-key "sk-xxx" --model gpt-4

// Enable AI with Claude
config ai --enable --provider claude --api-key "sk-ant-xxx" --model claude-3-opus-20240229

// Show current configuration
config ai --show

// Disable AI
config ai --disable
~~~`,
	}

	aiCmd.Flags().Bool("enable", false, "Enable AI assistant")
	aiCmd.Flags().Bool("disable", false, "Disable AI assistant")
	aiCmd.Flags().Bool("show", false, "Show current AI configuration")
	aiCmd.Flags().String("provider", "", "AI provider: openai or claude")
	aiCmd.Flags().String("api-key", "", "API key for the AI provider")
	aiCmd.Flags().String("endpoint", "", "API endpoint URL")
	aiCmd.Flags().String("model", "", "Model name (e.g., gpt-4, claude-3-opus-20240229)")
	aiCmd.Flags().Int("max-tokens", 0, "Maximum tokens in response")
	aiCmd.Flags().Int("timeout", 0, "Request timeout in seconds")
	aiCmd.Flags().Int("history-size", 0, "Number of history lines to include as context")
	aiCmd.Flags().Bool("opsec-check", false, "Enable AI OPSEC risk assessment for high-risk commands")

	return aiCmd
}

