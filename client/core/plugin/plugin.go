package plugin

import (
	"github.com/chainreactors/malice-network/client/assets"
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
	}

	return plug, nil
}

type DefaultPlugin struct {
	*MalManiFest
	Enable  bool
	Content []byte
	Path    string
	CMDs    Commands
}

func (plug *DefaultPlugin) Manifest() *MalManiFest {
	return plug.MalManiFest
}

func (plug *DefaultPlugin) Commands() Commands {
	return plug.CMDs
}
