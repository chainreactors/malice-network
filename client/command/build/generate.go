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
	go func() {
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
			con.Log.Errorf("Build Beacon failed: %v", err)
			return
		}
		con.Log.Infof("Build Beacon success")
	}()
	return nil
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, buildType, modules, ca, interval, jitter := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildType == "" || buildTarget == "" {
			return errors.New("require build format/target")
		}
	}
	go func() {
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
			con.Log.Errorf("Build Bind failed: %v", err)
			return
		}
		con.Log.Infof("Build Bind success")
	}()
	return nil
}

func ShellCodeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, buildType, modules, ca, interval, jitter := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildType == "" || buildTarget == "" {
			return errors.New("require build format/target")
		}
	}
	go func() {
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
			con.Log.Errorf("Build ShellCode failed: %v", err)
			return
		}
		con.Log.Infof("Build ShellCode success")
	}()
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, buildType, modules, ca, interval, jitter := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildType == "" || buildTarget == "" {
			return errors.New("require build format/target")
		}
	}
	go func() {
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
			con.Log.Errorf("Build Prelude failed: %v", err)
			return
		}
		con.Log.Infof("Build Prelude success")
	}()
	return nil
}
