package plugin

import (
	"errors"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

const (
	LuaScript = "lua"
	GoPlugin  = "go"
)

type Plugin interface {
	Run() error
	Manifest() *MalManiFest
	Commands() Commands
	GetEvents() map[intermediate.EventCondition]intermediate.OnEventFunc
}

func NewPlugin(manifest *MalManiFest) (*DefaultPlugin, error) {
	path := filepath.Join(assets.GetMalsDir(), manifest.Name)
	content, err := os.ReadFile(filepath.Join(path, manifest.EntryFile))
	if err != nil {
		return nil, err
	}

	plug := &DefaultPlugin{
		MalManiFest: manifest,
		Enable:      true,
		Content:     content,
		Path:        path,
		CMDs:        make(Commands),
		Events:      make(map[intermediate.EventCondition]intermediate.OnEventFunc),
	}

	return plug, nil
}

type DefaultPlugin struct {
	*MalManiFest
	Enable  bool
	Content []byte
	Path    string
	CMDs    Commands
	Events  map[intermediate.EventCondition]intermediate.OnEventFunc
}

func (plug *DefaultPlugin) Manifest() *MalManiFest {
	return plug.MalManiFest
}

func (plug *DefaultPlugin) Commands() Commands {
	return plug.CMDs
}

func (plug *DefaultPlugin) GetEvents() map[intermediate.EventCondition]intermediate.OnEventFunc {
	return plug.Events
}

func ParseMalManifest(data []byte) (*MalManiFest, error) {
	extManifest := &MalManiFest{}
	err := yaml.Unmarshal(data, &extManifest)
	if err != nil {
		return nil, err
	}
	return extManifest, validManifest(extManifest)
}

func validManifest(manifest *MalManiFest) error {
	if manifest.Name == "" {
		return errors.New("missing `name` field in mal manifest")
	}
	return nil
}

func LoadMalManiFest(filename string) (*MalManiFest, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	manifest, err := ParseMalManifest(content)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}

func GetPluginManifest() []*MalManiFest {
	var manifests []*MalManiFest
	for _, malfile := range assets.GetInstalledMalManifests() {
		manifest, err := LoadMalManiFest(malfile)
		if err != nil {
			logs.Log.Errorf(err.Error())
			continue
		}
		if manifest.Lib {
			continue
		}
		manifests = append(manifests, manifest)
	}
	return manifests
}

func LoadGlobalLuaPlugin() []*DefaultPlugin {
	var plugins []*DefaultPlugin
	for _, malfile := range assets.GetInstalledMalManifests() {
		manifest, err := LoadMalManiFest(malfile)
		if err != nil {
			logs.Log.Errorf(err.Error())
			continue
		}
		if !manifest.Lib {
			continue
		}
		plug, err := NewPlugin(manifest)
		if err != nil {
			logs.Log.Errorf(err.Error())
			continue
		}
		plugins = append(plugins, plug)
	}
	return plugins
}
