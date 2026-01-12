package common

import (
	"context"
	"strings"
	"time"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
)

// AIQuestionCompleter provides AI-powered completion for questions starting with '?'
// When users type '? <question>' and press Tab, this completer calls the AI
// and returns suggestions based on the AI's response.
func AIQuestionCompleter(con *core.Console) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		// Build the question from args and current value (works for '?' command or '? <question>' style).
		parts := make([]string, 0, len(c.Args)+1)
		parts = append(parts, c.Args...)
		if c.Value != "" {
			parts = append(parts, c.Value)
		}
		question := strings.TrimSpace(strings.Join(parts, " "))
		question = strings.TrimSpace(strings.TrimPrefix(question, "?"))

		// If no question yet, show hint
		if question == "" {
			return carapace.ActionMessage("Type your question after '?', then press Tab for AI suggestions")
		}

		// If question is too short, don't call AI
		if len(question) < 3 {
			return carapace.ActionMessage("Enter a longer question for AI suggestions")
		}

		// Load settings
		settings, err := assets.GetSetting()
		if err != nil || settings == nil || settings.AI == nil || !settings.AI.Enable {
			return carapace.ActionMessage("AI not enabled. Use 'ai-config --enable --api-key <key>' to enable")
		}

		if settings.AI.APIKey == "" {
			return carapace.ActionMessage("AI API key not configured. Use 'ai-config --api-key <key>'")
		}

		// Get command history for context
		historySize := 20
		if settings.AI.HistorySize > 0 {
			historySize = settings.AI.HistorySize
		}
		history := con.GetRecentHistory(historySize)

		// Create AI client with a shorter timeout for completion
		aiClient := core.NewAIClient(settings.AI)
		timeout := 15 // Shorter timeout for completion
		if settings.AI.Timeout > 0 && settings.AI.Timeout < timeout {
			timeout = settings.AI.Timeout
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()

		// Ask AI
		response, err := aiClient.Ask(ctx, question, history)
		if err != nil {
			return carapace.ActionMessage("AI Error: " + err.Error())
		}

		// Parse command suggestions from response
		commands := core.ParseCommandSuggestions(response)

		if len(commands) == 0 {
			// No specific commands found, show a truncated response
			lines := strings.Split(response, "\n")
			results := make([]string, 0)
			shown := 0
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if shown >= 5 { // Limit to first 5 non-empty lines
					break
				}
				shown++

				// Truncate long lines
				if len(line) > 80 {
					line = line[:77] + "..."
				}
				results = append(results, line, "")
			}
			if len(results) > 0 {
				return carapace.ActionValuesDescribed(results...).Tag("AI Response")
			}
			return carapace.ActionMessage("No suggestions from AI")
		}

		// Build completion results from commands
		results := make([]string, 0, len(commands)*2)
		for _, cmd := range commands {
			description := cmd.Description
			if description == "" {
				description = "AI suggested command"
			}
			results = append(results, cmd.Command, description)
		}

		return carapace.ActionValuesDescribed(results...).Tag("AI Suggestions")
	})
}

// RegisterAICompleter registers the AI question completer for the '?' prefix
// This should be called during command registration
func RegisterAICompleter(con *core.Console) {
	// The AI completer is invoked through PreCmdRunLineHooks for '?' prefix
	// Tab completion is handled by the carapace integration
	// This function can be used for additional registration if needed
}
