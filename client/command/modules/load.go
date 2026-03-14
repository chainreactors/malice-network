package modules

import (
	"errors"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/build"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strings"
)

func LoadModuleCmd(cmd *cobra.Command, con *core.Console) error {
	bundle, _ := cmd.Flags().GetString("bundle")
	path, _ := cmd.Flags().GetString("path")
	artifactName, _ := cmd.Flags().GetString("artifact")
	modules, _ := cmd.Flags().GetString("modules")
	thirdModules, _ := cmd.Flags().GetString("3rd")
	if modules != "" && thirdModules != "" {
		return errors.New("--modules and --3rd options are mutually exclusive. please specify only one of them")
	}

	selectedSources := 0
	if artifactName != "" {
		selectedSources++
	}
	if path != "" {
		selectedSources++
	}
	if modules != "" || thirdModules != "" {
		selectedSources++
	}

	switch {
	case selectedSources == 0:
		return errors.New("must specify one of --path, --artifact, --modules or --3rd")
	case selectedSources > 1:
		return errors.New("--path, --artifact, --modules and --3rd are mutually exclusive. please specify only one source")
	case artifactName != "":
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
		con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
		return nil
	case modules != "" || thirdModules != "":
		return handleModuleBuild(cmd, con, strings.Split(modules, ","), strings.Split(thirdModules, ","))
	case path != "":
		// Default bundle handling
		if bundle == "" {
			bundle = filepath.Base(path)
		}
		session := con.GetInteractive()
		task, err := LoadModule(con.Rpc, session, bundle, path)
		if err != nil {
			return err
		}
		session.Console(task, string(*con.App.Shell().Line()))
		return nil
	}

	return errors.New("unreachable module load input state")
}

// handleModuleBuild handles module build based on the builder resource (docker/action)
func handleModuleBuild(_ *cobra.Command, con *core.Console, modules, thirdModules []string) error {
	sess := con.GetInteractive()
	if sess == nil {
		return errors.New("no active session")
	}

	source, err := build.CheckSource(con, &clientpb.BuildConfig{})
	if err != nil {
		return err
	}
	target, ok := consts.GetBuildTargetNameByArchOS(sess.Os.Arch, sess.Os.Name)
	if !ok {
		return types.ErrInvalidateTarget
	}

	maleficConfig, err := build.BuildModuleMaleficConfig(modules, thirdModules)
	if err != nil {
		return err
	}

	buildConfig := &clientpb.BuildConfig{
		Target:        target,
		BuildType:     consts.CommandBuildModules,
		Source:        source,
		MaleficConfig: maleficConfig,
	}
	if err := build.ValidateOutputType(buildConfig, false, false, false); err != nil {
		return err
	}

	artifact, err := con.Rpc.SyncBuild(con.SyncBuildContext(), buildConfig)
	if err != nil {
		return err
	}

	task, err := con.Rpc.LoadModule(sess.Context(), &implantpb.LoadModule{
		Bundle: artifact.Name,
		Bin:    artifact.Bin,
	})
	if err != nil {
		return err
	}
	sess.Console(task, string(*con.App.Shell().Line()))
	return nil
}

func LoadModule(rpc clientrpc.MaliceRPCClient, session *client.Session, bundle string, path string) (*clientpb.Task, error) {
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
