package mal

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func RemoveMalCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	if name == "" {
		return errors.New("mal name is required")
	}
	confirmModel := tui.NewConfirm(fmt.Sprintf("Remove '%s' extension?", name))
	newConfirm := tui.NewModel(confirmModel, nil, false, true)
	err := newConfirm.Run()
	if err != nil {
		return err
	}
	if !confirmModel.Confirmed {
		return nil
	}
	err = RemoveMal(name, con)
	if err != nil {
		return err
	}
	return nil
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
	file.ForceRemoveAll(extPath)
	return nil
}
