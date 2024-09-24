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

func NewPlugin(manifest *MalManiFest) (*Plugin, error) {
	path := filepath.Join(assets.GetMalsDir(), manifest.Name)
	content, err := os.ReadFile(filepath.Join(path, manifest.EntryFile))
	if err != nil {
		return nil, err
	}

	plug := &Plugin{
		MalManiFest: manifest,
		Enable:      true,
		Content:     content,
		Path:        path,
	}

	return plug, nil
}

type Plugin struct {
	*MalManiFest
	Enable  bool
	Content []byte
	Path    string
}
