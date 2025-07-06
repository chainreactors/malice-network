package mal

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/plugin"
	"github.com/chainreactors/tui"
	"os"
	"path/filepath"

	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/spf13/cobra"
)

func RemoveMalCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	if name == "" {
		return errors.New("mal name is required")
	}
	confirmModel := tui.NewConfirm(fmt.Sprintf("Remove '%s' mal?", name))
	err := confirmModel.Run()
	if err != nil {
		return err
	}
	if !confirmModel.GetConfirmed() {
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
		return errors.New("mal name is required")
	}

	malManager := plugin.GetGlobalMalManager()

	if _, exists := malManager.GetEmbedPlugin(name); exists {
		return errors.New("cannot remove embedded mal")
	}

	plug, exists := malManager.GetExternalPlugin(name)
	if !exists {
		return errors.New("mal not found")
	}

	implantMenu := con.ImplantMenu()
	for _, cmd := range plug.Commands() {
		implantMenu.RemoveCommand(cmd.Command)
	}

	err := malManager.RemoveExternalMal(name)
	if err != nil {
		return err
	}

	// 从profile中移除mal记录
	profile, err := assets.GetProfile()
	if err != nil {
		con.Log.Warnf("Failed to get profile: %s\n", err)
	} else {
		profile.RemoveMal(name)
	}

	malPath := filepath.Join(assets.GetMalsDir(), name)
	if _, err := os.Stat(malPath); !os.IsNotExist(err) {
		err := fileutils.ForceRemoveAll(malPath)
		if err != nil {
			return err
		}
	}

	con.Log.Importantf("Successfully removed mal: %s\n", name)
	return nil
}
