package llm

import (
	"fmt"
	"os"
	"strings"
)

type providerConfig struct {
	baseURL   string
	apiKeyEnv string
}

var presets = map[string]providerConfig{
	"openai":     {baseURL: "https://api.openai.com/v1", apiKeyEnv: "OPENAI_API_KEY"},
	"openrouter": {baseURL: "https://openrouter.ai/api/v1", apiKeyEnv: "OPENROUTER_API_KEY"},
	"deepseek":   {baseURL: "https://api.deepseek.com/v1", apiKeyEnv: "DEEPSEEK_API_KEY"},
	"groq":       {baseURL: "https://api.groq.com/openai/v1", apiKeyEnv: "GROQ_API_KEY"},
	"moonshot":   {baseURL: "https://api.moonshot.cn/v1", apiKeyEnv: "MOONSHOT_API_KEY"},
}

// ProviderOpts holds LLM provider configuration, typically from the client's config ai settings.
type ProviderOpts struct {
	Provider string // "openai", "deepseek", etc.
	APIKey   string // API key from client config ai
	Endpoint string // LLM API base URL from client config ai
}

// resolve determines the baseURL and apiKey to use, following a three-level fallback:
// 1. ProviderOpts fields (from client's config ai)
// 2. Environment variables (BRIDGE_<PROVIDER>_BASE_URL, BRIDGE_<PROVIDER>_API_KEY)
// 3. Provider presets
func resolve(opts ProviderOpts) (baseURL string, apiKey string, err error) {
	provider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if provider == "" {
		provider = "openai"
	}
	envPrefix := strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))

	// Resolve base URL
	baseURL = opts.Endpoint
	if baseURL == "" {
		baseURL = os.Getenv("BRIDGE_" + envPrefix + "_BASE_URL")
	}
	if baseURL == "" {
		if preset, ok := presets[provider]; ok {
			baseURL = preset.baseURL
		}
	}
	if baseURL == "" {
		return "", "", fmt.Errorf("unknown provider %q: configure via 'config ai --endpoint' or set BRIDGE_%s_BASE_URL", provider, envPrefix)
	}

	// Resolve API key
	apiKey = opts.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("BRIDGE_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("BRIDGE_" + envPrefix + "_API_KEY")
	}
	if apiKey == "" {
		if preset, ok := presets[provider]; ok {
			apiKey = os.Getenv(preset.apiKeyEnv)
		}
	}
	if apiKey == "" {
		return "", "", fmt.Errorf("missing API key for provider %q: configure via 'config ai --api-key' or set BRIDGE_%s_API_KEY",
			provider, envPrefix)
	}

	return baseURL, apiKey, nil
}
