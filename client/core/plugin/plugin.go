package plugin

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core/intermediate"
	"os"
	"path/filepath"
)

const (
	LuaScript = "lua"
	TCLScript = "tcl"
	CNAScript = "cna"
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
