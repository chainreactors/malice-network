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
	name, address, buildTarget, modules, ca, interval, jitter, srdi := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := con.Rpc.Build(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Type:        consts.CommandBuildBeacon,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			Srdi: srdi,
		})
		if err != nil {
			con.Log.Errorf("Build beacon failed: %v", err)
			return
		}
		con.Log.Infof("Build beacon success")
	}()
	return nil
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, interval, jitter, srdi := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := con.Rpc.Build(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Type:        consts.CommandBuildBind,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			Srdi: srdi,
		})
		if err != nil {
			con.Log.Errorf("Build bind failed: %v", err)
			return
		}
		con.Log.Infof("Build bind success")
	}()
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, interval, jitter, srdi := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := con.Rpc.Build(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Type:        consts.CommandBuildPrelude,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			Srdi: srdi,
		})
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v\n", err)
			return
		}
		con.Log.Infof("Build prelude success")
	}()
	return nil
}

func ModulesCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, interval, jitter, srdi := common.ParseGenerateFlags(cmd)
	if len(modules) == 0 {
		modules = []string{"full"}
	}
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := con.Rpc.Build(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Target:      buildTarget,
			Type:        consts.CommandBuildModules,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			Srdi: srdi,
		})
		if err != nil {
			con.Log.Errorf("Build modules failed: %v\n", err)
			return
		}
		con.Log.Infof("Build modules success")
	}()
	return nil
}

func PulseCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, _, _, _, _, srdi := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := con.Rpc.Build(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Target:      buildTarget,
			Type:        consts.CommandBuildPulse,
			Srdi:        srdi,
		})
		if err != nil {
			con.Log.Errorf("Build loader failed: %v", err)
			return
		}
		con.Log.Infof("Build loader success")
	}()
	return nil
}
