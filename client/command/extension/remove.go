package extension

import (
	"errors"
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/utils"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
)

// ExtensionsRemoveCmd - Remove an extension
func ExtensionsRemoveCmd(ctx *grumble.Context, con *console.Console) {
	name := ctx.Args.String("name")
	if name == "" {
		console.Log.Errorf("Extension name is required\n")
		return
	}
	confirm := false
	prompt := &survey.Confirm{Message: fmt.Sprintf("Remove '%s' extension?", name)}
	survey.AskOne(prompt, &confirm)
	if !confirm {
		return
	}
	err := RemoveExtensionByCommandName(name, con)
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
	con.App.Commands().Remove(commandName)
	extPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(commandName))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	forceRemoveAll(extPath)
	return nil
}

func forceRemoveAll(rootPath string) {
	utils.ChmodR(rootPath, 0o600, 0o700)
	os.RemoveAll(rootPath)
}
