package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
)

func TestSaveSettingsClearsRemovedNestedConfigs(t *testing.T) {
	initClientConfigTest(t)

	settings := &Settings{
		MaxServerLogSize: 11,
		OpsecThreshold:   7.5,
		McpEnable:        true,
		McpAddr:          "127.0.0.1:6001",
		LocalRPCEnable:   true,
		LocalRPCAddr:     "127.0.0.1:16001",
		Github: &GithubSetting{
			Owner:    "chainreactors",
			Repo:     "malice-network",
			Token:    "gh-token",
			Workflow: "generate.yml",
		},
		AI: &AISettings{
			Enable:      true,
			Provider:    "openai",
			APIKey:      "sk-test",
			Endpoint:    "https://api.openai.com/v1",
			Model:       "gpt-4",
			MaxTokens:   2048,
			Timeout:     45,
			HistorySize: 30,
			OpsecCheck:  true,
		},
	}

	if err := SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings initial write failed: %v", err)
	}

	settings.Github = nil
	settings.AI = nil
	if err := SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings cleanup write failed: %v", err)
	}

	reloaded, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}
	if reloaded.Github != nil {
		t.Fatalf("expected github config to be cleared, got %#v", reloaded.Github)
	}
	if reloaded.AI != nil {
		t.Fatalf("expected ai config to be cleared, got %#v", reloaded.AI)
	}

	content, err := os.ReadFile(filepath.Join(GetRootAppDir(), maliceProfile))
	if err != nil {
		t.Fatalf("failed to read saved profile: %v", err)
	}
	if string(content) == "" {
		t.Fatal("expected saved profile content")
	}
	if containsAny(string(content), "github:", "ai:", "api_key:", "workflow:") {
		t.Fatalf("expected nested settings to be removed from file, got:\n%s", string(content))
	}
}

func TestLoadSettingsReturnsSavedNestedConfigs(t *testing.T) {
	initClientConfigTest(t)

	want := &Settings{
		MaxServerLogSize: 12,
		OpsecThreshold:   8.0,
		McpEnable:        true,
		McpAddr:          "127.0.0.1:7001",
		LocalRPCEnable:   true,
		LocalRPCAddr:     "127.0.0.1:17001",
		Github: &GithubSetting{
			Owner:    "owner",
			Repo:     "repo",
			Token:    "token",
			Workflow: "build.yml",
		},
		AI: &AISettings{
			Enable:      true,
			Provider:    "claude",
			APIKey:      "anthropic-key",
			Endpoint:    "https://api.anthropic.com/v1",
			Model:       "claude-3-5-sonnet",
			MaxTokens:   4096,
			Timeout:     60,
			HistorySize: 50,
			OpsecCheck:  true,
		},
	}

	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	got, err := LoadSettings()
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	if got.MaxServerLogSize != 12 || got.OpsecThreshold != 8.0 || !got.McpEnable || got.McpAddr != "127.0.0.1:7001" || !got.LocalRPCEnable || got.LocalRPCAddr != "127.0.0.1:17001" {
		t.Fatalf("unexpected top-level settings: %#v", got)
	}
	if got.Github == nil || got.Github.Owner != "owner" || got.Github.Workflow != "build.yml" {
		t.Fatalf("unexpected github settings: %#v", got.Github)
	}
	if got.AI == nil || !got.AI.Enable || got.AI.Provider != "claude" || got.AI.APIKey != "anthropic-key" || got.AI.Model != "claude-3-5-sonnet" || got.AI.MaxTokens != 4096 || got.AI.Timeout != 60 || got.AI.HistorySize != 50 || !got.AI.OpsecCheck {
		t.Fatalf("unexpected ai settings: %#v", got.AI)
	}
}

func TestGetValidAISettingsUsesConfigAIHint(t *testing.T) {
	initClientConfigTest(t)

	if err := SaveSettings(&Settings{}); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	_, err := GetValidAISettings()
	if err == nil {
		t.Fatal("expected GetValidAISettings to fail when AI is disabled")
	}
	if !strings.Contains(err.Error(), "config ai --enable --api-key <key>") {
		t.Fatalf("expected config ai hint, got %q", err.Error())
	}
	if strings.Contains(err.Error(), "ai-config") {
		t.Fatalf("unexpected legacy alias in error: %q", err.Error())
	}
}

func initClientConfigTest(t *testing.T) {
	t.Helper()

	config.Reset()
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	}, config.WithHookFunc(HookFn))
	config.AddDriver(yamlDriver.Driver)

	root := t.TempDir()
	oldMaliceDirName := MaliceDirName
	MaliceDirName = root
	t.Cleanup(func() {
		MaliceDirName = oldMaliceDirName
		config.Reset()
	})
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if sub != "" && strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
