package llm

import (
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name        string
		opts        ProviderOpts
		envs        map[string]string
		wantBaseURL string
		wantAPIKey  string
		wantErr     string
	}{
		{
			name: "opts take priority over everything",
			opts: ProviderOpts{
				Provider: "openai",
				APIKey:   "key-from-opts",
				Endpoint: "https://custom.endpoint/v1",
			},
			wantBaseURL: "https://custom.endpoint/v1",
			wantAPIKey:  "key-from-opts",
		},
		{
			name: "env fallback when opts empty",
			opts: ProviderOpts{Provider: "openai"},
			envs: map[string]string{
				"BRIDGE_OPENAI_BASE_URL": "https://env.endpoint/v1",
				"BRIDGE_API_KEY":         "key-from-env",
			},
			wantBaseURL: "https://env.endpoint/v1",
			wantAPIKey:  "key-from-env",
		},
		{
			name: "provider-specific env key before preset env key",
			opts: ProviderOpts{Provider: "openai"},
			envs: map[string]string{
				"BRIDGE_OPENAI_BASE_URL": "https://env.endpoint/v1",
				"BRIDGE_OPENAI_API_KEY":  "provider-specific-key",
				"BRIDGE_API_KEY":         "global-key",
			},
			wantBaseURL: "https://env.endpoint/v1",
			wantAPIKey:  "global-key", // BRIDGE_API_KEY is checked first
		},
		{
			name: "preset fallback for known provider",
			opts: ProviderOpts{
				Provider: "deepseek",
				APIKey:   "key-from-opts",
			},
			wantBaseURL: "https://api.deepseek.com/v1",
			wantAPIKey:  "key-from-opts",
		},
		{
			name: "groq preset",
			opts: ProviderOpts{
				Provider: "groq",
				APIKey:   "groq-key",
			},
			wantBaseURL: "https://api.groq.com/openai/v1",
			wantAPIKey:  "groq-key",
		},
		{
			name: "empty provider defaults to openai",
			opts: ProviderOpts{
				Provider: "",
				APIKey:   "default-key",
			},
			wantBaseURL: "https://api.openai.com/v1",
			wantAPIKey:  "default-key",
		},
		{
			name:    "unknown provider without env or endpoint",
			opts:    ProviderOpts{Provider: "nonexistent"},
			wantErr: "unknown provider",
		},
		{
			name: "missing API key",
			opts: ProviderOpts{
				Provider: "openai",
				Endpoint: "https://some.endpoint/v1",
			},
			wantErr: "missing API key",
		},
		{
			name: "preset env key fallback (OPENAI_API_KEY)",
			opts: ProviderOpts{Provider: "openai"},
			envs: map[string]string{
				"OPENAI_API_KEY": "preset-env-key",
			},
			wantBaseURL: "https://api.openai.com/v1",
			wantAPIKey:  "preset-env-key",
		},
		{
			name: "provider name is case insensitive and trimmed",
			opts: ProviderOpts{
				Provider: "  OpenAI  ",
				APIKey:   "trimmed-key",
			},
			wantBaseURL: "https://api.openai.com/v1",
			wantAPIKey:  "trimmed-key",
		},
		{
			name: "unknown provider with env vars set",
			opts: ProviderOpts{Provider: "custom-llm"},
			envs: map[string]string{
				"BRIDGE_CUSTOM_LLM_BASE_URL": "https://custom-llm.api/v1",
				"BRIDGE_CUSTOM_LLM_API_KEY":  "custom-key",
			},
			wantBaseURL: "https://custom-llm.api/v1",
			wantAPIKey:  "custom-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envs {
				t.Setenv(k, v)
			}

			baseURL, apiKey, err := resolve(tt.opts)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if baseURL != tt.wantBaseURL {
				t.Errorf("baseURL = %q, want %q", baseURL, tt.wantBaseURL)
			}
			if apiKey != tt.wantAPIKey {
				t.Errorf("apiKey = %q, want %q", apiKey, tt.wantAPIKey)
			}
		})
	}
}
