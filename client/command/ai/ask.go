package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

// AskCmd handles the ask command
func AskCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	writer := cmd.OutOrStdout()
	question := strings.Join(args, " ")
	if question == "" {
		return fmt.Errorf("please provide a question")
	}

	// Validate AI settings
	aiSettings, err := assets.GetValidAISettings()
	if err != nil {
		return err
	}

	// Get history settings
	historySize, _ := cmd.Flags().GetInt("history")
	noHistory, _ := cmd.Flags().GetBool("no-history")

	var history []string
	if !noHistory {
		history = con.GetRecentHistory(historySize)
	}

	// Create AI client
	aiClient := core.NewAIClient(aiSettings)

	// Create context with timeout
	timeout := aiSettings.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	fmt.Fprintln(writer, "Thinking...")

	// Ask the AI
	response, err := aiClient.Ask(ctx, question, history)
	if err != nil {
		return fmt.Errorf("AI error: %w", err)
	}

	// Parse command suggestions
	commands := core.ParseCommandSuggestions(response)

	// Display response
	fmt.Fprintf(writer, "\n%s\n", response)

	// If there are command suggestions, list them
	if len(commands) > 0 {
		fmt.Fprintln(writer, "\nSuggested commands:")
		for i, suggestion := range commands {
			fmt.Fprintf(writer, "  [%d] %s\n", i+1, suggestion.Command)
		}
	}

	fmt.Fprintln(writer)

	return nil
}
