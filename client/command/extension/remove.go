package extension

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// ExtensionsRemoveCmd - Remove an extension
func ExtensionsRemoveCmd(cmd *cobra.Command, con *core.Console) {
	name := cmd.Flags().Arg(0)
	if name == "" {
		con.Log.Errorf("Extension name is required\n")
		return
	}
	confirmed, err := common.Confirm(cmd, con, fmt.Sprintf("Remove '%s' extension?", name))
	if err != nil {
		con.Log.Errorf("Error running confirm model: %s", err)
		return
	}
	if !confirmed {
		return
	}
	err = RemoveExtensionByCommandName(name, con)
	if err != nil {
		con.Log.Errorf("Error removing extension: %s\n", err)
		return
	} else {
		con.Log.Infof("Extension '%s' removed\n", name)
	}
}

// RemoveExtensionByCommandName - Remove an extension by command name
func RemoveExtensionByCommandName(commandName string, con *core.Console) error {
	if commandName == "" {
		return errors.New("command name is required")
	}
	loadedExt, ok := loadedExtensions[commandName]
	if !ok {
		return errors.New("extension not loaded")
	}
	delete(loadedExtensions, commandName)
	implantMenu := con.ImplantMenu()
	common.RemoveCommandByName(implantMenu, commandName)
	if loadedExt.Manifest != nil && loadedExt.Manifest.Manifest != nil {
		delete(loadedManifests, loadedExt.Manifest.Manifest.Name)
	}
	profile, err := assets.GetProfile()
	installName := commandName
	if loadedExt.Manifest != nil && loadedExt.Manifest.Manifest != nil && loadedExt.Manifest.Manifest.Name != "" {
		installName = loadedExt.Manifest.Manifest.Name
	}
	if err == nil {
		profile.RemoveExtension(installName)
	}
	extPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(installName))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	fileutils.ForceRemoveAll(extPath)
	return nil
}
