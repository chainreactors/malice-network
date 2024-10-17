package assets

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

var (
	settingsFileName   = "malice.config"
	DefaultSettings, _ = LoadSettings()
)

type Settings struct {
	TableStyle        string `yaml:"tables" config:"tables"`
	AutoAdult         bool   `yaml:"autoadult" config:"autoadult"`
	BeaconAutoResults bool   `yaml:"beacon_autoresults" config:"beacon_autoresults"`
	SmallTermWidth    int    `yaml:"small_term_width" config:"small_term_width"`
	AlwaysOverflow    bool   `yaml:"always_overflow" config:"always_overflow"`
	VimMode           bool   `yaml:"vim_mode" config:"vim_mode"`
	DefaultTimeout    int    `yaml:"default_timeout" config:"default_timeout" default:""`
	MaxServerLogSize  int    `yaml:"max_server_log_size" config:"max_server_log_size" default:"10"`
}

func LoadSettings() (*Settings, error) {
	rootDir, _ := filepath.Abs(GetRootAppDir())
	data, err := ioutil.ReadFile(filepath.Join(rootDir, settingsFileName))
	if err != nil {
		return defaultSettings(), err
	}
	settings := defaultSettings()
	err = json.Unmarshal(data, settings)
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
	err = ioutil.WriteFile(filepath.Join(rootDir, settingsFileName), data, 0600)
	return err
}
