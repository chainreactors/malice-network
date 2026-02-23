package build

import (
	"errors"
	"fmt"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func PreludeCmd(cmd *cobra.Command, con *core.Console) error {
	buildConfig, err := parseBasicConfig(cmd, con)
	if err != nil {
		return err
	}
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)
	if err != nil {
		return fmt.Errorf("failed to parse build config: %w", err)
	}
	buildConfig.BuildType = consts.CommandBuildPrelude

	// Layer 1: Load from profile (server-side)
	profileName, _ := cmd.Flags().GetString("profile")
	if profileName != "" {
		profilePB, err := con.Rpc.GetProfileByName(con.Context(), &clientpb.Profile{Name: profileName})
		if err != nil {
			return fmt.Errorf("failed to get profile: %w", err)
		}
		buildConfig.MaleficConfig = profilePB.ImplantConfig
		buildConfig.PreludeConfig = profilePB.PreludeConfig
		buildConfig.Resources = profilePB.Resources
	}

	// Layer 2+3: File inputs (archive < individual files)
	fileImplant, filePrelude, fileResources, err := loadBuildInputs(cmd)
	if err != nil {
		return fmt.Errorf("failed to load build inputs: %w", err)
	}
	if fileImplant != nil {
		buildConfig.MaleficConfig = fileImplant
	}
	if filePrelude != nil {
		buildConfig.PreludeConfig = filePrelude
	}
	if fileResources != nil {
		buildConfig.Resources = fileResources
	}

	// Prelude build requires prelude config
	if buildConfig.PreludeConfig == nil {
		return errors.New("prelude build requires prelude config (use --prelude-path, --archive-path, or --profile with prelude config)")
	}

	if err := parseLibFlag(cmd, buildConfig); err != nil {
		return err
	}

	return ExecuteBuild(con, buildConfig)
}
