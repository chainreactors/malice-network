package build

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
)

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, buildType, modules, ca, interval, jitter := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildType == "" || buildTarget == "" {
			return errors.New("require build format/target")
		}
	}
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:    name,
		Url:     url,
		Stager:  consts.CommandBeacon,
		Type:    buildType,
		Target:  buildTarget,
		Modules: modules,
		Ca:      ca,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		return err
	}
	con.Log.Infof("Generate PE success")
	return err
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, buildType, modules, ca, interval, jitter := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildType == "" || buildTarget == "" {
			return errors.New("require build format/target")
		}
	}
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:    name,
		Url:     url,
		Stager:  consts.CommandBind,
		Type:    buildType,
		Target:  buildTarget,
		Modules: modules,
		Ca:      ca,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		return err
	}
	con.Log.Infof("Generate Module success")
	return nil
}

func ShellCodeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, buildType, modules, ca, interval, jitter := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildType == "" || buildTarget == "" {
			return errors.New("require build format/target")
		}
	}
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:    name,
		Url:     url,
		Stager:  consts.CommandShellCode,
		Type:    buildType,
		Target:  buildTarget,
		Modules: modules,
		Ca:      ca,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		return err
	}
	con.Log.Infof("Generate ShellCode success")
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, buildType, modules, ca, interval, jitter := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildType == "" || buildTarget == "" {
			return errors.New("require build format/target")
		}
	}
	_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
		Name:    name,
		Url:     url,
		Stager:  consts.CommandPrelude,
		Type:    buildType,
		Target:  buildTarget,
		Modules: modules,
		Ca:      ca,
		Params: map[string]string{
			"interval": interval,
			"jitter":   jitter,
		},
	})
	if err != nil {
		return err
	}
	con.Log.Infof("Generate Stage0 success")
	return nil
}
