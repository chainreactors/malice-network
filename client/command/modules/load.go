package modules

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/action"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/errs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func LoadModuleCmd(cmd *cobra.Command, con *repl.Console) error {
	bundle, _ := cmd.Flags().GetString("bundle")
	path, _ := cmd.Flags().GetString("path")
	modules, _ := cmd.Flags().GetStringSlice("modules")
	builderResource, _ := cmd.Flags().GetString("build")
	if path == "" && len(modules) == 0 {
		return errors.New("path or modules is required")
	}
	if len(modules) > 0 && path == "" {
		if builderResource == "docker" {
			go func() {
				builder, err := con.Rpc.BuildModules(con.Context(), &clientpb.Generate{
					Target:  "x86_64-pc-windows-msvc",
					Modules: modules,
				})
				if err != nil {
					con.Log.Errorf("Build modules failed: %v", err)
					return
				}
				path := filepath.Join(assets.TempDirName, builder.Name)
				err = os.WriteFile(path, builder.Bin, 0666)
				if err != nil {
					con.Log.Errorf("Write modules failed: %v\n", err)
					return
				}

				task, err := LoadModule(con.Rpc, con.GetInteractive(), builder.Name, path)
				if err != nil {
					con.Log.Errorf("Load modules failed: %v\n", err)
					return
				}
				con.GetInteractive().Console(task, fmt.Sprintf("load %s %s", modules, path))
			}()
			return nil
		} else if builderResource == "action" {
			if len(modules) == 0 {
				modules = []string{"full"}
			}
			var workflowID string
			setting := assets.GetProfile().Settings
			if setting.GithubOwner == "" || setting.GithubRepo == "" || setting.GithubToken == "" {
				return errors.New("require github owner/repo/token")
			}
			if setting.GithubWorkflowFile == "" {
				workflowID = "generate.yaml"
			} else {
				workflowID = setting.GithubWorkflowFile
			}
			configByte := types.DefaultProfile
			buildConfig, err := types.LoadProfile(configByte)
			if err != nil {
				return err
			}
			buildConfig.Implant.Modules = modules
			configByte, _ = yaml.Marshal(buildConfig)
			inputs := map[string]string{
				"malefic_config_yaml":      base64.StdEncoding.EncodeToString(configByte),
				"package":                  consts.CommandBuildModules,
				"targets":                  "x86_64-pc-windows-msvc",
				"malefic_modules_features": strings.Join(modules, ","),
			}
			go func() {
				builder, err := action.RunWorkFlow(con, &clientpb.WorkflowRequest{
					Inputs:     inputs,
					Owner:      setting.GithubOwner,
					Repo:       setting.GithubRepo,
					Token:      setting.GithubToken,
					WorkflowId: workflowID,
				})
				if err != nil {
					con.Log.Errorf("Run workflow failed: %v", err)
					return
				}
				for {
					time.Sleep(120 * time.Second)
					resp, err := con.Rpc.DownloadGithubArtifact(con.Context(), &clientpb.WorkflowRequest{
						Owner:      setting.GithubOwner,
						Repo:       setting.GithubRepo,
						Token:      setting.GithubToken,
						WorkflowId: workflowID,
						BuildName:  builder.Name,
					})
					if err == nil {
						modulePath := filepath.Join(assets.GetTempDir(), resp.Name)
						err := os.WriteFile(modulePath, resp.Zip, 0666)
						if err != nil {
							con.Log.Errorf("Write modules failed: %v\n", err)
							break
						}
						con.Log.Importantf("Download modules success in %s\n", modulePath)
						task, err := LoadModule(con.Rpc, con.GetInteractive(), resp.Name, modulePath)
						if err != nil {
							con.Log.Errorf("Load modules failed: %v\n", err)
							break
						}
						con.GetInteractive().Console(task, fmt.Sprintf("load %s %s\n", modules, modulePath))
						break
					} else if errors.Is(err, errs.ErrWorkflowFailed) {
						con.Log.Error("Workflow failure\n")
						break
					}
				}
			}()
			return nil
		}

	}
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
