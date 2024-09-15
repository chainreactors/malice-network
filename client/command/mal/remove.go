package mal

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func RemoveMalCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	if name == "" {
		con.Log.Errorf("Extension name is required\n")
		return
	}
	confirmModel := tui.NewConfirm(fmt.Sprintf("Remove '%s' extension?", name))
	newConfirm := tui.NewModel(confirmModel, nil, false, true)
	err := newConfirm.Run()
	if err != nil {
		con.Log.Errorf("Error running confirm model: %s", err)
		return
	}
	if !confirmModel.Confirmed {
		return
	}
	err = RemoveMal(name, con)
	if err != nil {
		con.Log.Errorf(err.Error())
	}
}

func RemoveMal(name string, con *repl.Console) error {
	if name == "" {
		return errors.New("command name is required")
	}
	if plug, ok := loadedMals[name]; !ok {
		return errors.New("extension not loaded")
	} else {
		implantMenu := con.ImplantMenu()
		for _, cmd := range plug.CMDs {
			implantMenu.RemoveCommand(cmd)
		}
	}

	extPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(name))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	delete(loadedMals, name)
	utils.ForceRemoveAll(extPath)
	return nil
}
