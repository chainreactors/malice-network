package exec

import (
	"fmt"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

const (
	CommandSpawn           = "spawn"
	spawnDelayMilliseconds = 2000
)

func SpawnCmd(cmd *cobra.Command, con *core.Console) error {
	artifactName, _ := cmd.Flags().GetString("artifact")
	process, _ := cmd.Flags().GetString("process")
	quiet, _ := cmd.Flags().GetBool("quiet")
	timeout, _ := cmd.Flags().GetUint32("timeout")

	task, err := Spawn(
		con.Rpc,
		con.GetInteractive(),
		strings.TrimSpace(artifactName),
		process,
		!quiet,
		timeout,
		common.ParseSacrificeFlags(cmd),
	)
	if err != nil {
		return err
	}
	return common.HandleTaskOutput(cmd, con, task)
}

func Spawn(rpc clientrpc.MaliceRPCClient, sess *client.Session, artifactName, process string,
	output bool, timeout uint32, sacrifice *implantpb.SacrificeProcess) (*clientpb.Task, error) {
	if sess == nil {
		return nil, fmt.Errorf("no active session")
	}
	artifactName = strings.TrimSpace(artifactName)
	if artifactName == "" {
		return nil, fmt.Errorf("artifact name is required")
	}

	return rpc.ExecuteSpawn(sess.Context(), &implantpb.ExecuteBinary{
		Name:        artifactName,
		ProcessName: process,
		Output:      output,
		Timeout:     timeout,
		Sacrifice:   sacrifice,
		Delay:       spawnDelayMilliseconds,
	})
}

func SpawnArtifactCompleter(con *core.Console) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		artifacts, err := con.Rpc.ListArtifact(con.Context(), &clientpb.Empty{})
		if err != nil {
			con.Log.Errorf("Error get artifacts: %v\n", err)
			return carapace.Action{}
		}

		results := make([]string, 0)
		for _, artifact := range artifacts.GetArtifacts() {
			if artifact == nil ||
				!strings.EqualFold(artifact.GetType(), consts.CommandBuildBeacon) ||
				!strings.EqualFold(artifact.GetPlatform(), consts.Windows) ||
				!strings.EqualFold(artifact.GetStatus(), consts.BuildStatusCompleted) {
				continue
			}
			results = append(results,
				artifact.GetName(),
				fmt.Sprintf("format %s, arch %s, target %s", artifact.GetFormat(), artifact.GetArch(), artifact.GetTarget()),
			)
		}
		return carapace.ActionValuesDescribed(results...).Tag("artifact")
	})
}
