package modules

import (
	"errors"
	"fmt"
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
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func LoadModuleCmd(cmd *cobra.Command, con *repl.Console) error {
	bundle, _ := cmd.Flags().GetString("bundle")
	path, _ := cmd.Flags().GetString("path")
	modules, _ := cmd.Flags().GetStringSlice("modules")
	artifactName, _ := cmd.Flags().GetString("artifact")
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
		con.GetInteractive().Console(task, fmt.Sprintf("load %s %s", modules, modulePath))
		return nil
	}
	if path == "" && len(modules) == 0 {
		return errors.New("path or modules is required")
	}

	if len(modules) > 0 && path == "" {
		isSelfModule, _ := cmd.Flags().GetBool("module")
		is3rdModule, _ := cmd.Flags().GetBool("3rd")
		if isSelfModule && is3rdModule {
			return errors.New("--module and --3rd options are mutually exclusive. please specify only one of them")
		} else if !isSelfModule && !is3rdModule {
			return errors.New("must specify either --module or --3rd. One of them is required")
		}
		return handleModuleBuild(con, modules, isSelfModule, is3rdModule)
	}

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
}

// handleModuleBuild handles module build based on the builder resource (docker/action)
func handleModuleBuild(con *repl.Console, modules []string, isModule, is3rdMdule bool) error {
	setting, err := assets.GetSetting()
	if err != nil {
		return err
	}
	source, err := build.CheckResource(con, "", setting.Github.ToProtobuf())
	if err != nil {
		return err
	}
	target, ok := consts.GetBuildTargetNameByArchOS(con.GetInteractive().Session.Os.Arch, con.Session.Os.Name)
	if !ok {
		return errs.ErrInvalidateTarget
	}
	params := &types.ProfileParams{
		Modules: strings.Join(modules, ","),
	}
	if isModule {
		params.Module3rd = false
	} else {
		params.Module3rd = true
	}
	buildConfig := &clientpb.BuildConfig{
		Target:      target,
		ParamsBytes: []byte(params.String()),
		Type:        consts.CommandBuildModules,
		Source:      source,
	}
	return buildModule(con, buildConfig, strings.Join(modules, ","))
}

// buildWithDocker handles module build via Docker
func buildModule(con *repl.Console, buildConfig *clientpb.BuildConfig, modules string) error {
	var modulePath string
	go func() {
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build modules failed: %v", err)
			return
		}
		modulePath, err = handleModuleDownload(con, artifact.Name, artifact.Bin)
		if err != nil {
			con.Log.Errorf("Write modules failed: %v\n", err)
			return
		}

		task, err := LoadModule(con.Rpc, con.GetInteractive(), artifact.Name, modulePath)
		if err != nil {
			con.Log.Errorf("Load modules failed: %v\n", err)
			return
		}
		con.GetInteractive().Console(task, fmt.Sprintf("load %s %s", modules, modulePath))
	}()
	return nil
}

// handleModuleDownload handles module download and saves to disk
func handleModuleDownload(con *repl.Console, moduleName string, moduleBin []byte) (string, error) {
	var modulePath string
	if len(moduleBin) > 0 {
		modulePath = filepath.Join(assets.GetTempDir(), moduleName)
		err := os.WriteFile(modulePath, moduleBin, 0666)
		if err != nil {
			return "", err
		}
	} else {
		for {
			time.Sleep(30 * time.Second)
			artifact, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
				Name: moduleName,
			})
			if err == nil {
				modulePath = filepath.Join(assets.GetTempDir(), artifact.Name)
				err = os.WriteFile(modulePath, artifact.Bin, 0666)
				if err != nil {
					return "", err
				}
				break
			}
		}
	}
	return modulePath, nil
}

func LoadModule(rpc clientrpc.MaliceRPCClient, session *core.Session, bundle string, path string) (*clientpb.Task, error) {
	data, err := os.ReadFile(path)
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
