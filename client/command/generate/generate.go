package generate

import (
	"context"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func PECmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	url, buildTarget, interval, jitter := common.ParseGenerateFlags(cmd)
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:   name,
		Url:    url,
		Type:   consts.CommandPE,
		Target: buildTarget,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		con.Log.Errorf("generate failed: %s", err)
		return
	}
	con.Log.Infof("Generate PE success")
}

func ModuleCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	url, buildTarget, interval, jitter := common.ParseGenerateFlags(cmd)
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:   name,
		Url:    url,
		Type:   consts.CommandModule,
		Target: buildTarget,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		con.Log.Errorf("generate failed: %s", err)
		return
	}
	con.Log.Infof("Generate Module success")
}

func ShellCodeCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	url, buildTarget, interval, jitter := common.ParseGenerateFlags(cmd)
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:   name,
		Url:    url,
		Type:   consts.CommandShellCode,
		Target: buildTarget,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		con.Log.Errorf("generate failed: %s", err)
		return
	}
	con.Log.Infof("Generate ShellCode success")
}

func Stage0Cmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	url, buildTarget, interval, jitter := common.ParseGenerateFlags(cmd)
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:   name,
		Url:    url,
		Type:   consts.CommandStage0,
		Target: buildTarget,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		con.Log.Errorf("generate failed: %s", err)
		return
	}
	con.Log.Infof("Generate Stage0 success")
}

func Stage1Cmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	url, buildTarget, interval, jitter := common.ParseGenerateFlags(cmd)
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:   name,
		Url:    url,
		Type:   consts.CommandStage1,
		Target: buildTarget,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		con.Log.Errorf("generate failed: %s", err)
		return
	}
	con.Log.Infof("Generate Stage1 success")
}
