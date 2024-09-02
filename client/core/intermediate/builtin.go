package intermediate

import (
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/kballard/go-shellquote"
	"os"
	"path/filepath"
)

func GetResourceFile(pluginName, filename string) (string, error) {
	resourcePath := filepath.Join(assets.GetMalsDir(), pluginName, "resources", filename)
	return resourcePath, nil
}

func ReadResourceFile(pluginName, filename string) (string, error) {
	resourcePath, _ := GetResourceFile(pluginName, filename)
	content, err := os.ReadFile(resourcePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func NewBinaryMessage(pluginName, module, filename, args string, sarcifice *implantpb.SacrificeProcess) (*implantpb.ExecuteBinary, error) {
	content, _ := ReadResourceFile(pluginName, filename)
	params, err := shellquote.Split(args)
	if err != nil {
		return nil, err
	}
	return &implantpb.ExecuteBinary{
		Name:      filename,
		Bin:       []byte(content),
		Type:      module,
		Params:    params,
		Output:    true,
		Sacrifice: sarcifice,
	}, nil
}

func NewSacrificeProcessMessage(ppid int64, block_dll bool, argue string, args string) (*implantpb.SacrificeProcess, error) {
	params, err := shellquote.Split(args)
	if err != nil {
		return nil, err
	}
	return &implantpb.SacrificeProcess{
		Ppid:     uint32(ppid),
		Output:   true,
		BlockDll: block_dll,
		Argue:    argue,
		Params:   params,
	}, err
}
