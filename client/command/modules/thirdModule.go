package modules

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/spf13/cobra"
	"path/filepath"
	"strings"
)

func Load3rdModuleCmd(cmd *cobra.Command, con *repl.Console) error {
	bundle, _ := cmd.Flags().GetString("bundle")
	path, _ := cmd.Flags().GetString("path")
	artifactName, _ := cmd.Flags().GetString("artifact")
	modules, _ := cmd.Flags().GetString("modules")
	if artifactName != "" {
		err := loadExistArtifact(con, artifactName, modules)
		if err != nil {
			return err
		}
	} else if modules != "" {
		go func() {
			err := handleModuleBuild(con, strings.Split(modules, ","), true)
			if err != nil {
				logs.Log.Errorf("Error loading modules: %s\n", err)
			}
		}()
		return nil
	} else if path != "" {
		// Default bundle handling
		if bundle == "" {
			bundle = filepath.Base(path)
		}
		session := con.GetInteractive()
		task, err := LoadModule(con.Rpc, session, bundle, path)
		if err != nil {
			return err
		}
		session.Console(task, fmt.Sprintf("load %s %s", bundle, path))
		return nil
	} else {
		return errors.New("must specify either --path or --modules. One of them is required")
	}
	return nil
}
