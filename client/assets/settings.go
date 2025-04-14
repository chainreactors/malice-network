package assets

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"io/ioutil"
	"path/filepath"
)

//var (
//	DefaultSettings, _ = LoadSettings()
//)

type Settings struct {
	MaxServerLogSize   int     `yaml:"max_server_log_size" config:"max_server_log_size" default:"10"`
	GithubRepo         string  `yaml:"github_repo" config:"github_repo" default:""`
	GithubOwner        string  `yaml:"github_owner" config:"github_owner" default:""`
	GithubToken        string  `yaml:"github_token" config:"github_token" default:""`
	GithubWorkflowFile string  `yaml:"github_workflow_file" config:"github_workflow_file" default:"generate.yaml"`
	OpsecThreshold     float64 `yaml:"opsec_threshold" config:"opsec_threshold" default:"6.0"`
	McpPort            int     `yaml:"mcp_port" config:"mcp_port" default:"5005"`
	//VtApiKey           string `yaml:"vt_api_key" config:"vt_api_key" default:""`
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
