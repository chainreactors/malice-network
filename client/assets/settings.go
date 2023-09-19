package assets

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
)

var (
	settingsFileName = "malice.config"
)

type Settings struct {
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
