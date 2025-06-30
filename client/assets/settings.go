package assets

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"io/ioutil"
	"path/filepath"
)

//var (
//	DefaultSettings, _ = LoadSettings()
//)

type Settings struct {
	MaxServerLogSize int            `yaml:"max_server_log_size" config:"max_server_log_size" default:"10"`
	OpsecThreshold   float64        `yaml:"opsec_threshold" config:"opsec_threshold" default:"6.0"`
	McpPort          int            `yaml:"mcp_port" config:"mcp_port" default:"5005"`
	Github           *GithubSetting `yaml:"github" config:"github"`

	//VtApiKey           string `yaml:"vt_api_key" config:"vt_api_key" default:""`
}

type GithubSetting struct {
	Repo     string `yaml:"repo" config:"repo" default:""`
	Owner    string `yaml:"owner" config:"owner" default:""`
	Token    string `yaml:"token" config:"token" default:""`
	Workflow string `yaml:"workflow" config:"workflow" default:"generate.yaml"`
}

func (github *GithubSetting) ToProtobuf() *clientpb.GithubWorkflowConfig {
	return &clientpb.GithubWorkflowConfig{
		Owner:      github.Owner,
		Repo:       github.Repo,
		Token:      github.Token,
		WorkflowId: github.Workflow,
	}
}

func LoadSettings() (*Settings, error) {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	//data, err := os.ReadFile(filepath.Join(rootDir, settingsFileName))
	//if err != nil {
	//	return defaultSettings(), err
	//}
	settings := defaultSettings()
	err := configutil.LoadConfig(filepath.Join(rootDir, maliceProfile), settings)
	if err != nil {
		return defaultSettings(), err
	}
	return settings, nil
}

func defaultSettings() *Settings {
	return &Settings{}
}

// SaveSettings - Save the current settings to disk
func SaveSettings(settings *Settings) error {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	if settings == nil {
		settings = defaultSettings()
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(rootDir, maliceProfile), data, 0600)
	return err
}
