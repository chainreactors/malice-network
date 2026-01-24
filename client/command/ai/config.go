package ai

import (
	"fmt"
	"strings"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
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

// AIConfigCmd handles the ai-config command
func AIConfigCmd(cmd *cobra.Command, con *core.Console) error {
	showConfig, _ := cmd.Flags().GetBool("show")
	enableAI, _ := cmd.Flags().GetBool("enable")
	disableAI, _ := cmd.Flags().GetBool("disable")

	// Load current settings
	settings, err := assets.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Initialize AI settings if nil
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

	// Show current config
	if showConfig {
		printAIConfig(settings.AI)
		return nil
	}

	// If no flags provided, show help
	if !enableAI && !disableAI && !cmd.Flags().Changed("provider") &&
		!cmd.Flags().Changed("api-key") && !cmd.Flags().Changed("endpoint") &&
		!cmd.Flags().Changed("model") && !cmd.Flags().Changed("max-tokens") &&
		!cmd.Flags().Changed("timeout") && !cmd.Flags().Changed("history-size") {
		printAIConfig(settings.AI)
		fmt.Println("\nUse --help to see available options")
		return nil
	}

	// Update settings based on flags
	if enableAI {
		settings.AI.Enable = true
	}
	if disableAI {
		settings.AI.Enable = false
	}

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

	// Validate configuration if enabling
	if settings.AI.Enable && settings.AI.APIKey == "" {
		fmt.Println("Warning: AI is enabled but API key is not set. Use --api-key to set it.")
	}

	// Save settings
	if err := assets.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	fmt.Println("AI configuration updated successfully")
	printAIConfig(settings.AI)

	return nil
}

func boolToYesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}

func printAIConfig(ai *assets.AISettings) {
	fmt.Println("\nAI Configuration:")
	fmt.Println("─────────────────────────────────────")

	fmt.Printf("  Enabled:      %s\n", boolToYesNo(ai.Enable))
	fmt.Printf("  Provider:     %s\n", ai.Provider)
	fmt.Printf("  Endpoint:     %s\n", ai.Endpoint)
	fmt.Printf("  Model:        %s\n", ai.Model)

	// Mask API key
	apiKeyDisplay := "(not set)"
	if ai.APIKey != "" {
		if len(ai.APIKey) > 8 {
			apiKeyDisplay = ai.APIKey[:4] + "..." + ai.APIKey[len(ai.APIKey)-4:]
		} else {
			apiKeyDisplay = "****"
		}
	}
	fmt.Printf("  API Key:      %s\n", apiKeyDisplay)

	fmt.Printf("  Max Tokens:   %d\n", ai.MaxTokens)
	fmt.Printf("  Timeout:      %ds\n", ai.Timeout)
	fmt.Printf("  History Size: %d lines\n", ai.HistorySize)

	fmt.Printf("  OPSEC Check:  %s\n", boolToYesNo(ai.OpsecCheck))
	fmt.Println()
}
