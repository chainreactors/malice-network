package build

import (
	"fmt"
	"os"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/implanttypes"

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

	pulseArtifactID, _ := cmd.Flags().GetUint32("artifact-id")
	buildConfig.ArtifactId = pulseArtifactID

	// Load implant.yaml from file if specified
	var baseYAML []byte
	if cmd.Flags().Changed("implant-path") {
		implantPath, _ := cmd.Flags().GetString("implant-path")
		baseYAML, err = os.ReadFile(implantPath)
		if err != nil {
			return fmt.Errorf("failed to read implant file %s: %w", implantPath, err)
		}
	}

	profile, err := parsePulseBuildFlags(cmd, baseYAML)
	if err != nil {
		return fmt.Errorf("failed to parse pulse's build flags: %w", err)
	}
	buildConfig.MaleficConfig, err = profile.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to encode profile: %w", err)
	}

	return ExecuteBuild(con, buildConfig)
}

func parsePulseBuildFlags(cmd *cobra.Command, baseYAML []byte) (*implanttypes.ProfileConfig, error) {
	var newProfile *implanttypes.ProfileConfig
	var err error
	if baseYAML != nil {
		newProfile, err = implanttypes.LoadProfile(baseYAML)
	} else {
		newProfile, err = implanttypes.LoadProfile(consts.DefaultProfile)
	}
	if err != nil {
		return nil, err
	}

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
	return newProfile, nil
}
