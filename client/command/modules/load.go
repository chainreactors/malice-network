package modules

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func LoadModuleCmd(cmd *cobra.Command, con *repl.Console) error {
	bundle, _ := cmd.Flags().GetString("bundle")
	path, _ := cmd.Flags().GetString("path")
	artifactName, _ := cmd.Flags().GetString("artifact")
	modules, _ := cmd.Flags().GetString("modules")
	thirdModules, _ := cmd.Flags().GetString("3rd")
	if artifactName != "" {
		artifact, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
			Name: artifactName,
		})
		if err != nil {
			return err
		}
		modulePath := filepath.Join(assets.GetTempDir(), artifact.Name)
		err = os.WriteFile(modulePath, artifact.Bin, 0666)
		if err != nil {
			return err
		}
		task, err := LoadModule(con.Rpc, con.GetInteractive(), artifact.Name, modulePath)
		if err != nil {
			return err
		}
		con.GetInteractive().Console(cmd, task, fmt.Sprintf("load %s %s", modules, modulePath))
		return nil
	} else if modules != "" || thirdModules != "" {
		if modules != "" && thirdModules != "" {
			return errors.New("--module and --3rd options are mutually exclusive. please specify only one of them")
		} else {
			go func() {
				err := handleModuleBuild(cmd, con, strings.Split(modules, ","), strings.Split(thirdModules, ","))
				if err != nil {
					logs.Log.Errorf("Error loading modules: %s\n", err)
				}
			}()
			return nil
		}
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
		session.Console(cmd, task, fmt.Sprintf("load %s %s", bundle, path))
		return nil
	} else {
		return errors.New("must specify either --path, --modules or --3rd_modules. One of them is required")
	}
}

// handleModuleBuild handles module build based on the builder resource (docker/action)
func handleModuleBuild(cmd *cobra.Command, con *repl.Console, modules, thirdModules []string) error {
	source, err := build.CheckResource(con, "", nil)
	if err != nil {
		return err
	}
	target, ok := consts.GetBuildTargetNameByArchOS(con.GetInteractive().Session.Os.Arch, con.Session.Os.Name)
	if !ok {
		return errs.ErrInvalidateTarget
	}
	var params *types.ProfileParams
	if len(modules) != 0 {
		params = &types.ProfileParams{
			Modules: strings.Join(modules, ","),
		}
	} else if len(thirdModules) != 0 {
		params = &types.ProfileParams{
			Enable3RD: true,
			Modules:   strings.Join(modules, ","),
		}
	} else {
		return errors.New("must specify either --modules or --3rd. One of them is required")
	}
	artifact, err := con.Rpc.SyncBuild(con.SyncBuildContext(), &clientpb.BuildConfig{
		Target:      target,
		ParamsBytes: []byte(params.String()),
		Type:        consts.CommandBuildModules,
		Source:      source,
	})
	if err != nil {
		return err
	}

	sess := con.GetInteractive()
	task, err := con.Rpc.LoadModule(sess.Context(), &implantpb.LoadModule{
		Bundle: artifact.Name,
		Bin:    artifact.Bin,
	})
	if err != nil {
		return err
	}
	sess.Console(cmd, task, fmt.Sprintf("load module from artifact %s", artifact.Name))
	return nil
}

func LoadModule(rpc clientrpc.MaliceRPCClient, session *core.Session, bundle string, path string) (*clientpb.Task, error) {
	data, err := pe.Unpack(path)
	if err != nil {
		return nil, err
	}
	task, err := rpc.LoadModule(session.Context(), &implantpb.LoadModule{
		Bundle: bundle,
		Bin:    data,
	})

	if err != nil {
		return nil, err
	}
	return task, nil
}
