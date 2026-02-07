package build

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func PulseFlagSet(f *pflag.FlagSet) {
	f.String("address", "", "Only support single address")
	f.String("path", "/pulse", "")
	f.String("user-agent", "", "HTTP User-Agent string")
	f.Uint32("artifact-id", 0, "pulse artifact id")
	f.Uint32("beacon-artifact-id", 0, "beacon artifact id used by pulse relink")
}

func PulseCmd(cmd *cobra.Command, con *core.Console) error {
	//buildConfig, err := prepareBuildConfig(cmd, con, consts.CommandBuildPulse)
	buildConfig, err := parseBasicConfig(cmd, con)
	if err != nil {
		return err
	}
	source, _ := cmd.Flags().GetString("source")
	buildConfig.Source = source
	buildConfig, err = parseSourceConfig(cmd, con, buildConfig)
	if err != nil {
		return fmt.Errorf("failed to parse build config: %w", err)
	}
	buildConfig.BuildType = consts.CommandBuildPulse
	if err := parseLibFlag(cmd, buildConfig); err != nil {
		return err
	}
	if err != nil {
		return err
	}

	pulseArtifactID, _ := cmd.Flags().GetUint32("artifact-id")
	buildConfig.ArtifactId = pulseArtifactID

	profile, err := parsePulseBuildFlags(cmd)
	if err != nil {
		return fmt.Errorf("failed to parse pulse's build flags: %w", err)
	}
	buildConfig.MaleficConfig, err = profile.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to encode profile: %w", err)
	}

	return ExecuteBuild(con, buildConfig)
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
	beaconArtifactID, _ := cmd.Flags().GetUint32("beacon-artifact-id")
	newProfile.Pulse.Flags.ArtifactID = beaconArtifactID
	if cmd.Flags().Changed("user-agent") {
		ua, _ := cmd.Flags().GetString("user-agent")
		newProfile.Pulse.Http.Headers["User-Agent"] = ua
	}
	//content, _ := newProfile.ToYAML()
	//println(string(content))
	return newProfile, nil
}
