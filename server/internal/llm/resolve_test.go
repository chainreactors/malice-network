package llm

import (
	"strings"
	"testing"
	"time"

	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name          string
		opts          ProviderOpts
		setupConfig   func()
		envs          map[string]string
		wantProvider  string
		wantBaseURL   string
		wantAPIKey    string
		wantProxyURL  string
		wantTimeout   time.Duration
		wantErrSubstr string
	}{
		{
			name: "request opts take priority over config and env",
			opts: ProviderOpts{
				Provider: "openai",
				APIKey:   "key-from-opts",
				Endpoint: "https://custom.endpoint/v1",
			},
			setupConfig: func() {
				config.Set("server.llm.providers.openai.endpoint", "https://cfg.endpoint/v1")
				config.Set("server.llm.providers.openai.api_key", "cfg-key")
			},
			envs: map[string]string{
				"BRIDGE_OPENAI_BASE_URL": "https://env.endpoint/v1",
				"BRIDGE_API_KEY":         "env-key",
			},
			wantProvider: "openai",
			wantBaseURL:  "https://custom.endpoint/v1",
			wantAPIKey:   "key-from-opts",
			wantTimeout:  120 * time.Second,
		},
		{
			name: "server config provider fallback is used before env",
			opts: ProviderOpts{Provider: "deepseek"},
			setupConfig: func() {
				config.Set("server.llm.providers.deepseek.endpoint", "https://cfg.deepseek/v1")
				config.Set("server.llm.providers.deepseek.api_key", "cfg-deepseek-key")
				config.Set("server.llm.providers.deepseek.proxy_url", "socks5://127.0.0.1:1080")
				config.Set("server.llm.providers.deepseek.timeout", 45)
			},
			envs: map[string]string{
				"BRIDGE_DEEPSEEK_BASE_URL": "https://env.deepseek/v1",
				"BRIDGE_DEEPSEEK_API_KEY":  "env-deepseek-key",
			},
			wantProvider: "deepseek",
			wantBaseURL:  "https://cfg.deepseek/v1",
			wantAPIKey:   "cfg-deepseek-key",
			wantProxyURL: "socks5://127.0.0.1:1080",
			wantTimeout:  45 * time.Second,
		},
		{
			name: "global server config fallback works for default provider",
			opts: ProviderOpts{},
			setupConfig: func() {
				config.Set("server.llm.default_provider", "openrouter")
				config.Set("server.llm.endpoint", "https://proxy.internal/v1")
				config.Set("server.llm.api_key", "global-key")
				config.Set("server.llm.proxy_url", "http://127.0.0.1:8080")
				config.Set("server.llm.timeout", 33)
			},
			wantProvider: "openrouter",
			wantBaseURL:  "https://proxy.internal/v1",
			wantAPIKey:   "global-key",
			wantProxyURL: "http://127.0.0.1:8080",
			wantTimeout:  33 * time.Second,
		},
		{
			name: "env fallback still works without server config",
			opts: ProviderOpts{Provider: "openai"},
			envs: map[string]string{
				"BRIDGE_OPENAI_BASE_URL": "https://env.endpoint/v1",
				"BRIDGE_API_KEY":         "env-key",
			},
			wantProvider: "openai",
			wantBaseURL:  "https://env.endpoint/v1",
			wantAPIKey:   "env-key",
			wantTimeout:  120 * time.Second,
		},
		{
			name: "preset fallback works for known provider",
			opts: ProviderOpts{
				Provider: "groq",
				APIKey:   "groq-key",
			},
			wantProvider: "groq",
			wantBaseURL:  "https://api.groq.com/openai/v1",
			wantAPIKey:   "groq-key",
			wantTimeout:  120 * time.Second,
		},
		{
			name: "unknown provider with server config is allowed",
			opts: ProviderOpts{Provider: "custom-llm"},
			setupConfig: func() {
				config.Set("server.llm.providers.custom-llm.endpoint", "https://custom-llm.api/v1")
				config.Set("server.llm.providers.custom-llm.api_key", "custom-key")
			},
			wantProvider: "custom-llm",
			wantBaseURL:  "https://custom-llm.api/v1",
			wantAPIKey:   "custom-key",
			wantTimeout:  120 * time.Second,
		},
		{
			name:          "unknown provider without config or env returns error",
			opts:          ProviderOpts{Provider: "nonexistent"},
			wantErrSubstr: "unknown provider",
		},
		{
			name: "missing API key returns error",
			opts: ProviderOpts{
				Provider: "openai",
				Endpoint: "https://some.endpoint/v1",
			},
			wantErrSubstr: "missing API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			initResolveConfigRuntime(t)
			for k, v := range tt.envs {
				t.Setenv(k, v)
			}
			if tt.setupConfig != nil {
				tt.setupConfig()
			}

			got, err := resolve(tt.opts)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSubstr)
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErrSubstr, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Provider != tt.wantProvider {
				t.Fatalf("provider = %q, want %q", got.Provider, tt.wantProvider)
			}
			if got.BaseURL != tt.wantBaseURL {
				t.Fatalf("baseURL = %q, want %q", got.BaseURL, tt.wantBaseURL)
			}
			if got.APIKey != tt.wantAPIKey {
				t.Fatalf("apiKey = %q, want %q", got.APIKey, tt.wantAPIKey)
			}
			if got.ProxyURL != tt.wantProxyURL {
				t.Fatalf("proxyURL = %q, want %q", got.ProxyURL, tt.wantProxyURL)
			}
			if got.Timeout != tt.wantTimeout {
				t.Fatalf("timeout = %s, want %s", got.Timeout, tt.wantTimeout)
			}
		})
	}
}

func initResolveConfigRuntime(t *testing.T) {
	t.Helper()

	config.Reset()
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yamlDriver.Driver)
	t.Cleanup(config.Reset)
}
