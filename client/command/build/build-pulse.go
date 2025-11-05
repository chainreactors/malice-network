package build

import (
	"fmt"
	consts "github.com/chainreactors/IoM-go/consts"
	"strings"

	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func PulseFlagSet(f *pflag.FlagSet) {
	f.String("address", "", "Only support single address")
	f.String("path", "/pulse", "")
	f.String("user-agent", "", "HTTP User-Agent string")
	f.Uint32("beacon_artifact_id", 0, "beacon's artifact_id")
}

func PulseCmd(cmd *cobra.Command, con *repl.Console) error {
	//buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildPulse)
	buildConfig, err := parseBasicConfig(cmd, con)
	if err != nil {
		return err
	}
	if !strings.Contains(buildConfig.Target, "windows") {
		con.Log.Warn("Pulse only supports Windows targets\n")
		return nil
	}
	source, _ := cmd.Flags().GetString("source")
	buildConfig.Source = source
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)
	if err != nil {
		return fmt.Errorf("failed to parse build config: %w", err)
	}
	buildConfig.BuildType = consts.CommandBuildPulse
	if err != nil {
		return err
	}
	profile, err := parsePulseBuildFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse pulse's build flags: %w", err)
	}
	buildConfig.MaleficConfig, err = profile.ToYAML()

	executeBuild(con, buildConfig)
	return nil
}

func parsePulseBuildFlags(cmd *cobra.Command) (*implanttypes.ProfileConfig, error) {
	newProfile, _ := implanttypes.LoadProfile(consts.DefaultProfile)
	//println(string(consts.DefaultProfile))
	//println(newProfile.Pulse.Http.Headers)
	//newProfile.SetDefaults()
	// Basic profile flags - only override if explicitly provided
	if cmd.Flags().Changed("address") {
		address, _ := cmd.Flags().GetString("address")
		if strings.Contains(address, "http://") {
			address = strings.TrimPrefix(address, "http://")
			if !strings.Contains(address, ":") {
				address += ":80"
			}
			newProfile.Pulse.Protocol = "http"
			newProfile.Pulse.Target = address
			newProfile.Pulse.Http.Method = "POST"
			newProfile.Pulse.Http.Version = "1.1"
			newProfile.Pulse.Http.Host = address
			newProfile.Pulse.Http.Headers["Host"] = address
		} else if strings.Contains(address, "tcp://") {
			address = strings.TrimPrefix(address, "tcp://")
			if !strings.Contains(address, ":") {
				address += ":5001"
			}
			newProfile.Pulse.Protocol = "tcp"
			newProfile.Pulse.Target = address
		}
	}
	beacon_artifact_id, _ := cmd.Flags().GetUint32("beacon_artifact_id")
	newProfile.Pulse.Flags.ArtifactID = beacon_artifact_id
	if cmd.Flags().Changed("user-agent") {
		ua, _ := cmd.Flags().GetString("user-agent")
		newProfile.Pulse.Http.Headers["User-Agent"] = ua
	}
	//content, _ := newProfile.ToYAML()
	//println(string(content))
	return newProfile, nil
}
