package llm

import (
	"os"
	"strings"
	"testing"

	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
)

type liveTestConfig struct {
	BaseURL string
	APIKey  string
	Model   string
}

func loadLiveTestConfig(t *testing.T) liveTestConfig {
	t.Helper()

	apiKey := strings.TrimSpace(os.Getenv("MAL_AGENT_E2E_API_KEY"))
	if apiKey == "" {
		t.Skip("skipping live LLM test: MAL_AGENT_E2E_API_KEY is not set")
	}

	baseURL := strings.TrimSpace(os.Getenv("MAL_AGENT_E2E_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	model := strings.TrimSpace(os.Getenv("MAL_AGENT_E2E_MODEL"))
	if model == "" {
		model = "gpt-5.4"
	}

	return liveTestConfig{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
	}
}

func initLiveConfigRuntime(t *testing.T) {
	t.Helper()

	config.Reset()
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yamlDriver.Driver)
	t.Cleanup(config.Reset)
}
