package llm

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

type providerPreset struct {
	baseURL   string
	apiKeyEnv string
}

type ResolvedProvider struct {
	Provider string
	BaseURL  string
	APIKey   string
	ProxyURL string
	Timeout  time.Duration
}

var presets = map[string]providerPreset{
	"openai":     {baseURL: "https://api.openai.com/v1", apiKeyEnv: "OPENAI_API_KEY"},
	"openrouter": {baseURL: "https://openrouter.ai/api/v1", apiKeyEnv: "OPENROUTER_API_KEY"},
	"deepseek":   {baseURL: "https://api.deepseek.com/v1", apiKeyEnv: "DEEPSEEK_API_KEY"},
	"groq":       {baseURL: "https://api.groq.com/openai/v1", apiKeyEnv: "GROQ_API_KEY"},
	"moonshot":   {baseURL: "https://api.moonshot.cn/v1", apiKeyEnv: "MOONSHOT_API_KEY"},
}

// ProviderOpts holds LLM provider configuration, typically selected from client config ai.
// Only provider/model are currently passed over RPC; endpoint/api key remain optional overrides.
type ProviderOpts struct {
	Provider string
	APIKey   string
	Endpoint string
}

func resolve(opts ProviderOpts) (ResolvedProvider, error) {
	llmCfg := configs.GetLLMConfig()

	provider := strings.ToLower(strings.TrimSpace(opts.Provider))
	if provider == "" && llmCfg != nil {
		provider = strings.ToLower(strings.TrimSpace(llmCfg.DefaultProvider))
	}
	if provider == "" {
		provider = "openai"
	}

	envPrefix := strings.ToUpper(strings.ReplaceAll(provider, "-", "_"))
	providerCfg := providerConfigFor(llmCfg, provider)

	baseURL := strings.TrimSpace(opts.Endpoint)
	if baseURL == "" {
		baseURL = strings.TrimSpace(providerCfg.Endpoint)
	}
	if baseURL == "" && llmCfg != nil {
		baseURL = strings.TrimSpace(llmCfg.Endpoint)
	}
	if baseURL == "" {
		baseURL = os.Getenv("BRIDGE_" + envPrefix + "_BASE_URL")
	}
	if baseURL == "" {
		if preset, ok := presets[provider]; ok {
			baseURL = preset.baseURL
		}
	}
	if baseURL == "" {
		return ResolvedProvider{}, fmt.Errorf("unknown provider %q: configure server.llm.providers.%s.endpoint or set BRIDGE_%s_BASE_URL", provider, provider, envPrefix)
	}

	apiKey := strings.TrimSpace(opts.APIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(providerCfg.APIKey)
	}
	if apiKey == "" && llmCfg != nil {
		apiKey = strings.TrimSpace(llmCfg.APIKey)
	}
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
		return ResolvedProvider{}, fmt.Errorf("missing API key for provider %q: configure server.llm.providers.%s.api_key or set BRIDGE_%s_API_KEY", provider, provider, envPrefix)
	}

	proxyURL := strings.TrimSpace(providerCfg.ProxyURL)
	if proxyURL == "" && llmCfg != nil {
		proxyURL = strings.TrimSpace(llmCfg.ProxyURL)
	}

	timeout := 120 * time.Second
	if llmCfg != nil && llmCfg.Timeout > 0 {
		timeout = time.Duration(llmCfg.Timeout) * time.Second
	}
	if providerCfg.Timeout > 0 {
		timeout = time.Duration(providerCfg.Timeout) * time.Second
	}

	return ResolvedProvider{
		Provider: provider,
		BaseURL:  baseURL,
		APIKey:   apiKey,
		ProxyURL: proxyURL,
		Timeout:  timeout,
	}, nil
}

func providerConfigFor(cfg *configs.LLMConfig, provider string) configs.LLMProviderConfig {
	if cfg == nil || cfg.Providers == nil {
		return configs.LLMProviderConfig{}
	}
	if p := cfg.Providers[provider]; p != nil {
		return *p
	}
	for name, p := range cfg.Providers {
		if strings.EqualFold(strings.TrimSpace(name), provider) && p != nil {
			return *p
		}
	}
	return configs.LLMProviderConfig{}
}
