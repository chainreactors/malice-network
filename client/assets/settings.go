package assets

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/gookit/config/v2"
)

//var (
//	DefaultSettings, _ = LoadSettings()
//)

type Settings struct {
	MaxServerLogSize int            `yaml:"max_server_log_size" config:"max_server_log_size" default:"10"`
	OpsecThreshold   float64        `yaml:"opsec_threshold" config:"opsec_threshold" default:"6.0"`
	McpEnable        bool           `yaml:"mcp_enable" config:"mcp_enable" default:"false"`
	McpAddr          string         `yaml:"mcp_addr" config:"mcp_addr" default:"127.0.0.1:5005"`
	LocalRPCEnable   bool           `yaml:"localrpc_enable" config:"localrpc_enable" default:"false"`
	LocalRPCAddr     string         `yaml:"localrpc_addr" config:"localrpc_addr" default:"127.0.0.1:15004"`
	Github           *GithubSetting `yaml:"github" config:"github"`
	AI               *AISettings    `yaml:"ai" config:"ai"`

	//VtApiKey           string `yaml:"vt_api_key" config:"vt_api_key" default:""`
}

// AISettings holds configuration for AI assistant integration
type AISettings struct {
	Enable      bool   `yaml:"enable" config:"enable" default:"false"`
	Provider    string `yaml:"provider" config:"provider" default:"openai"` // openai, claude
	APIKey      string `yaml:"api_key" config:"api_key" default:""`
	Endpoint    string `yaml:"endpoint" config:"endpoint" default:"https://api.openai.com/v1"`
	Model       string `yaml:"model" config:"model" default:"gpt-4"`
	MaxTokens   int    `yaml:"max_tokens" config:"max_tokens" default:"1024"`
	Timeout     int    `yaml:"timeout" config:"timeout" default:"30"`
	HistorySize int    `yaml:"history_size" config:"history_size" default:"20"`
	OpsecCheck  bool   `yaml:"opsec_check" config:"opsec_check" default:"false"` // Enable AI OPSEC risk assessment
}

type GithubSetting struct {
	Repo     string `yaml:"repo" config:"repo" default:""`
	Owner    string `yaml:"owner" config:"owner" default:""`
	Token    string `yaml:"token" config:"token" default:""`
	Workflow string `yaml:"workflow" config:"workflow" default:"generate.yaml"`
}

func (github *GithubSetting) ToProtobuf() *clientpb.GithubActionBuildConfig {
	if github == nil || github.Token == "" || github.Owner == "" || github.Repo == "" || github.Workflow == "" {
		return nil
	}
	return &clientpb.GithubActionBuildConfig{
		Owner:      github.Owner,
		Repo:       github.Repo,
		Token:      github.Token,
		WorkflowId: github.Workflow,
	}
}

func LoadSettings() (*Settings, error) {
	setting, err := GetSetting()
	if err == nil && setting != nil {
		return setting, nil
	}

	_, loadErr := LoadProfile()
	if loadErr != nil {
		return defaultSettings(), loadErr
	}

	setting, err = GetSetting()
	if err != nil {
		return defaultSettings(), err
	}
	if setting == nil {
		return defaultSettings(), nil
	}
	return setting, nil
}

func defaultSettings() *Settings {
	return &Settings{
		MaxServerLogSize: 10,
		OpsecThreshold:   6.0,
		McpEnable:        false, // 默认关闭 MCP
		McpAddr:          "127.0.0.1:5005",
		LocalRPCEnable:   false, // 默认关闭 Local RPC
		LocalRPCAddr:     "127.0.0.1:15004",
	}
}

// setConfigs sets multiple config key-value pairs, returning the first error encountered.
func setConfigs(kvs [][2]interface{}) error {
	for _, kv := range kvs {
		if err := config.Set(kv[0].(string), kv[1]); err != nil {
			return err
		}
	}
	return nil
}

// SaveSettings - Save the current settings to disk
func SaveSettings(settings *Settings) error {
	if settings == nil {
		settings = defaultSettings()
	}

	// Ensure profile is loaded so we don't overwrite unrelated config sections.
	if _, err := LoadProfile(); err != nil {
		return err
	}

	// Top-level settings
	if err := setConfigs([][2]interface{}{
		{"settings.max_server_log_size", settings.MaxServerLogSize},
		{"settings.opsec_threshold", settings.OpsecThreshold},
		{"settings.mcp_enable", settings.McpEnable},
		{"settings.mcp_addr", settings.McpAddr},
		{"settings.localrpc_enable", settings.LocalRPCEnable},
		{"settings.localrpc_addr", settings.LocalRPCAddr},
	}); err != nil {
		return err
	}

	// Github settings
	if settings.Github != nil {
		if err := setConfigs([][2]interface{}{
			{"settings.github.repo", settings.Github.Repo},
			{"settings.github.owner", settings.Github.Owner},
			{"settings.github.token", settings.Github.Token},
			{"settings.github.workflow", settings.Github.Workflow},
		}); err != nil {
			return err
		}
	}

	// AI settings
	if settings.AI != nil {
		if err := setConfigs([][2]interface{}{
			{"settings.ai.enable", settings.AI.Enable},
			{"settings.ai.provider", settings.AI.Provider},
			{"settings.ai.api_key", settings.AI.APIKey},
			{"settings.ai.endpoint", settings.AI.Endpoint},
			{"settings.ai.model", settings.AI.Model},
			{"settings.ai.max_tokens", settings.AI.MaxTokens},
			{"settings.ai.timeout", settings.AI.Timeout},
			{"settings.ai.history_size", settings.AI.HistorySize},
			{"settings.ai.opsec_check", settings.AI.OpsecCheck},
		}); err != nil {
			return err
		}
	}

	return nil
}
