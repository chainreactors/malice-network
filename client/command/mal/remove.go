package mal

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/client/utils"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func RemoveMalCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	if name == "" {
		repl.Log.Errorf("Extension name is required\n")
		return
	}
	confirmModel := tui.NewConfirm(fmt.Sprintf("Remove '%s' extension?", name))
	newConfirm := tui.NewModel(confirmModel, nil, false, true)
	err := newConfirm.Run()
	if err != nil {
		repl.Log.Errorf("Error running confirm model: %s", err)
		return
	}
	if !confirmModel.Confirmed {
		return
	}
	err = RemoveMal(name, con)
	if err != nil {
		repl.Log.Errorf(err.Error())
	}
}

func RemoveMal(name string, con *repl.Console) error {
	if name == "" {
		return errors.New("command name is required")
	}
	if plug, ok := con.Plugins.Plugins[name]; !ok {
		return errors.New("extension not loaded")
	} else {
		implantMenu := con.App.Menu(consts.ImplantMenu)
		for _, cmd := range plug.CMDs {
			implantMenu.RemoveCommand(cmd)
		}
	}

	extPath := filepath.Join(assets.GetExtensionsDir(), filepath.Base(name))
	if _, err := os.Stat(extPath); os.IsNotExist(err) {
		return nil
	}
	delete(con.Plugins.Plugins, name)
	utils.ForceRemoveAll(extPath)
	return nil
}
