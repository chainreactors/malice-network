package ai

import (
	"fmt"
	"strings"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

// Provider constants
const (
	ProviderOpenAI    = "openai"
	ProviderClaude    = "claude"
	ProviderAnthropic = "anthropic"

	EndpointOpenAI    = "https://api.openai.com/v1"
	EndpointAnthropic = "https://api.anthropic.com/v1"

	DefaultModel     = "gpt-4"
	DefaultMaxTokens = 1024
	DefaultTimeout   = 30
	DefaultHistory   = 20
)

func initAISettings(settings *assets.Settings) {
	if settings.AI == nil {
		settings.AI = &assets.AISettings{
			Enable:      false,
			Provider:    ProviderOpenAI,
			Endpoint:    EndpointOpenAI,
			Model:       DefaultModel,
			MaxTokens:   DefaultMaxTokens,
			Timeout:     DefaultTimeout,
			HistorySize: DefaultHistory,
		}
	}
}

// AIShowCmd displays the current AI configuration as a KV table.
func AIShowCmd(con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	initAISettings(settings)
	printAIStatus(con, settings.AI)
	return nil
}

// AIEnableCmd enables AI and updates configuration flags.
func AIEnableCmd(cmd *cobra.Command, con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	initAISettings(settings)

	settings.AI.Enable = true

	if provider, _ := cmd.Flags().GetString("provider"); provider != "" {
		provider = strings.ToLower(provider)
		if provider == ProviderAnthropic {
			provider = ProviderClaude
		}
		if provider != ProviderOpenAI && provider != ProviderClaude {
			return fmt.Errorf("invalid provider: %s. Must be '%s' or '%s'", provider, ProviderOpenAI, ProviderClaude)
		}
		settings.AI.Provider = provider

		// Set default endpoint based on provider
		if !cmd.Flags().Changed("endpoint") {
			if provider == ProviderClaude {
				settings.AI.Endpoint = EndpointAnthropic
			} else {
				settings.AI.Endpoint = EndpointOpenAI
			}
		}
	}

	if apiKey, _ := cmd.Flags().GetString("api-key"); apiKey != "" {
		settings.AI.APIKey = apiKey
	}

	if endpoint, _ := cmd.Flags().GetString("endpoint"); endpoint != "" {
		settings.AI.Endpoint = endpoint
	}

	if model, _ := cmd.Flags().GetString("model"); model != "" {
		settings.AI.Model = model
	}

	if maxTokens, _ := cmd.Flags().GetInt("max-tokens"); maxTokens > 0 {
		settings.AI.MaxTokens = maxTokens
	}

	if timeout, _ := cmd.Flags().GetInt("timeout"); timeout > 0 {
		settings.AI.Timeout = timeout
	}

	if historySize, _ := cmd.Flags().GetInt("history-size"); historySize > 0 {
		settings.AI.HistorySize = historySize
	}

	if cmd.Flags().Changed("opsec-check") {
		opsecCheck, _ := cmd.Flags().GetBool("opsec-check")
		settings.AI.OpsecCheck = opsecCheck
	}

	if err := assets.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	con.Log.Importantf("AI preferences enabled\n")
	printAIStatus(con, settings.AI)
	return nil
}

// AIDisableCmd disables the AI assistant.
func AIDisableCmd(con *core.Console) error {
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}
	initAISettings(settings)

	settings.AI.Enable = false
	if err := assets.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	con.Log.Importantf("AI preferences disabled\n")
	return nil
}

func maskAPIKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) > 8 {
		return key[:4] + "..." + key[len(key)-4:]
	}
	return "****"
}

func printAIStatus(con *core.Console, ai *assets.AISettings) {
	enabled := tui.RedFg.Render("No")
	if ai.Enable {
		enabled = tui.GreenFg.Render("Yes")
	}

	opsec := tui.RedFg.Render("No")
	if ai.OpsecCheck {
		opsec = tui.GreenFg.Render("Yes")
	}

	values := map[string]string{
		"Enabled":                enabled,
		"Provider":               ai.Provider,
		"Model":                  ai.Model,
		"Max Tokens":             fmt.Sprintf("%d", ai.MaxTokens),
		"Timeout":                fmt.Sprintf("%ds", ai.Timeout),
		"History Size":           fmt.Sprintf("%d lines", ai.HistorySize),
		"OPSEC Check":            opsec,
		"Agent LLM Pipeline":     "server/config.yaml -> server.llm",
		"Legacy Local Endpoint":  legacyValue(ai.Endpoint),
		"Legacy Local API Key":   maskAPIKey(ai.APIKey),
	}
	keys := []string{
		"Enabled",
		"Provider",
		"Model",
		"Max Tokens",
		"Timeout",
		"History Size",
		"OPSEC Check",
		"Agent LLM Pipeline",
		"Legacy Local Endpoint",
		"Legacy Local API Key",
	}
	con.Log.Console(common.NewKVTable("AI", keys, values).View() + "\n")
}

func legacyValue(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(not set)"
	}
	return v
}
