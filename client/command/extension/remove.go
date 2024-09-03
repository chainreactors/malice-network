package extension

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// ExtensionsRemoveCmd - Remove an extension
func ExtensionsRemoveCmd(cmd *cobra.Command, con *console.Console) {
	name := cmd.Flags().Arg(0)
	if name == "" {
		console.Log.Errorf("Extension name is required\n")
		return
	}
	confirmModel := tui.NewConfirm(fmt.Sprintf("Remove '%s' extension?", name))
	newConfirm := tui.NewModel(confirmModel, nil, false, true)
	err := newConfirm.Run()
	if err != nil {
		console.Log.Errorf("Error running confirm model: %s", err)
		return
	}
	if !confirmModel.Confirmed {
		return
	}
	err = RemoveExtensionByCommandName(name, con)
	if err != nil {
		console.Log.Errorf("Error removing extension: %s\n", err)
		return
	} else {
		console.Log.Infof("Extension '%s' removed\n", name)
	}
}

// RemoveExtensionByCommandName - Remove an extension by command name
func RemoveExtensionByCommandName(commandName string, con *console.Console) error {
	if commandName == "" {
		return errors.New("command name is required")
	}
	if _, ok := loadedExtensions[commandName]; !ok {
		return errors.New("extension not loaded")
	}
	delete(loadedExtensions, commandName)
	implantMenu := con.App.Menu(consts.ImplantMenu)
	for _, cmd := range implantMenu.Commands() {
		if cmd.Name() == commandName {
			implantMenu.RemoveCommand(cmd)
		}
	}
	extPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(commandName))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	utils.ForceRemoveAll(extPath)
	return nil
}
