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

// AnalyzeCmd handles the analyze command - analyzes errors and provides suggestions
func AnalyzeCmd(cmd *cobra.Command, con *core.Console, args []string) error {
	aiSettings, err := assets.GetValidAISettings()
	if err != nil {
		return err
	}

	// Get the error to analyze
	var errorText string
	if len(args) > 0 {
		errorText = strings.Join(args, " ")
	}

	if errorText == "" {
		return fmt.Errorf("please provide an error message to analyze. Usage: analyze <error message>")
	}

	// Get context
	historySize := aiSettings.HistorySize
	if historySize <= 0 {
		historySize = 20
	}
	history := con.GetRecentHistory(historySize)

	// Build session context if available
	sessionContext := buildSessionContext(con)

	// Build the analysis prompt
	prompt := buildAnalysisPrompt(errorText, history, sessionContext)

	aiClient := core.NewAIClient(aiSettings)

	timeout := aiSettings.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	fmt.Println("\nAnalyzing error...")
	fmt.Println()

	// Use streaming for real-time output
	response, err := aiClient.AskStream(ctx, prompt, nil, func(chunk string) {
		fmt.Print(chunk)
	})
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	fmt.Println()

	// Parse command suggestions
	commands := core.ParseCommandSuggestions(response)
	if len(commands) > 0 {
		fmt.Println("\nSuggested commands:")
		for i, cmd := range commands {
			fmt.Printf("  [%d] %s\n", i+1, cmd.Command)
		}
	}

	fmt.Println()
	return nil
}

func buildSessionContext(con *core.Console) string {
	var sb strings.Builder

	session := con.GetInteractive()
	if session != nil {
		sb.WriteString(fmt.Sprintf("Current session: %s\n", session.SessionId))
		if session.Os != nil {
			sb.WriteString(fmt.Sprintf("OS: %s %s\n", session.Os.Name, session.Os.Arch))
		}
		if session.Process != nil {
			sb.WriteString(fmt.Sprintf("Process: %s (PID: %d)\n", session.Process.Name, session.Process.Pid))
			sb.WriteString(fmt.Sprintf("User: %s\n", session.Process.Owner))
		}
	} else {
		sb.WriteString("No active session\n")
	}

	return sb.String()
}

func buildAnalysisPrompt(errorText string, history []string, sessionContext string) string {
	var sb strings.Builder

	sb.WriteString("Analyze the following error and provide:\n")
	sb.WriteString("1. Possible causes of the error\n")
	sb.WriteString("2. Suggested solutions or workarounds\n")
	sb.WriteString("3. Alternative commands that might work\n\n")

	sb.WriteString("Error message:\n")
	sb.WriteString(errorText)
	sb.WriteString("\n\n")

	if sessionContext != "" {
		sb.WriteString("Session context:\n")
		sb.WriteString(sessionContext)
		sb.WriteString("\n")
	}

	if len(history) > 0 {
		sb.WriteString("Recent command history:\n")
		for _, cmd := range history {
			sb.WriteString(fmt.Sprintf("- %s\n", cmd))
		}
	}

	sb.WriteString("\nProvide a concise analysis. Wrap any command suggestions in backticks like `command`.")

	return sb.String()
}
