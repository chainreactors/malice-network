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
	TableStyle        string `config:"tables"`
	AutoAdult         bool   `config:"autoadult"`
	BeaconAutoResults bool   `config:"beacon_autoresults"`
	SmallTermWidth    int    `config:"small_term_width"`
	AlwaysOverflow    bool   `config:"always_overflow"`
	VimMode           bool   `config:"vim_mode"`
	DefaultTimeout    int    `config:"default_timeout" default:""`
	MaxServerLogSize  int    `config:"max_server_log_size" default:"10"`
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
