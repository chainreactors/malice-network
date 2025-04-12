package build

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
)

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, interval, jitter, _ := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		params := &types.ProfileParams{
			Interval: interval,
			Jitter:   jitter,
		}
		_, err := con.Rpc.Build(con.Context(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Type:        consts.CommandBuildBeacon,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params:      params.String(),
			Srdi:        true,
		})
		if err != nil {
			con.Log.Errorf("Build beacon failed: %v", err)
			return
		}
	}()
	return nil
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, interval, jitter, _ := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		params := &types.ProfileParams{
			Interval: interval,
			Jitter:   jitter,
		}
		_, err := con.Rpc.Build(con.Context(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Type:        consts.CommandBuildBind,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params:      params.String(),
			Srdi:        true,
		})
		if err != nil {
			con.Log.Errorf("Build bind failed: %v", err)
			return
		}
	}()
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, _, _, _ := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	autorunPath, _ := cmd.Flags().GetString("autorun")
	if autorunPath == "" {
		return errors.New("require autorun.yaml path")
	}
	file, err := os.ReadFile(autorunPath)
	if err != nil {
		return err
	}
	go func() {
		_, err := con.Rpc.Build(con.Context(), &clientpb.Generate{
			ProfileName: name,
			Address:     address,
			Type:        consts.CommandBuildPrelude,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Srdi:        true,
			Bin:         file,
		})
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v\n", err)
			return
		}
	}()
	return nil
}

func ModulesCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, _, _, _, srdi := common.ParseGenerateFlags(cmd)
	if len(modules) == 0 {
		modules = []string{"full"}
	}
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := BuildModules(con, name, address, buildTarget, modules, srdi)
		if err != nil {
			con.Log.Errorf("Build modules failed: %v", err)
			return
		}
	}()
	return nil
}

func PulseCmd(cmd *cobra.Command, con *repl.Console) error {
	profile, _ := cmd.Flags().GetString("profile")
	address, _ := cmd.Flags().GetString("address")
	buildTarget, _ := cmd.Flags().GetString("target")
	artifactId, _ := cmd.Flags().GetUint32("artifact-id")
	if !strings.Contains(buildTarget, "windows") {
		con.Log.Warn("pulse only support windows target\n")
		return nil
	}
	go func() {
		_, err := con.Rpc.Build(con.Context(), &clientpb.Generate{
			ProfileName: profile,
			Address:     address,
			Target:      buildTarget,
			Type:        consts.CommandBuildPulse,
			Srdi:        true,
			ArtifactId:  artifactId,
		})
		if err != nil {
			con.Log.Errorf("Build loader failed: %v", err)
			return
		}
	}()
	return nil
}

func BuildLogCmd(cmd *cobra.Command, con *repl.Console) error {
	id := cmd.Flags().Arg(0)
	buildID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	num, _ := cmd.Flags().GetInt("limit")
	builder, err := con.Rpc.BuildLog(con.Context(), &clientpb.Builder{
		Id:  uint32(buildID),
		Num: uint32(num),
	})
	if err != nil {
		return err
	}
	if len(builder.Log) == 0 {
		con.Log.Infof("No log for %s", id)
		return nil
	}
	con.Log.Console(string(builder.Log))
	return nil
}

func BuildModules(con *repl.Console, name, address, buildTarget string, modules []string, srdi bool) (bool, error) {
	_, err := con.Rpc.Build(con.Context(), &clientpb.Generate{
		ProfileName: name,
		Address:     address,
		Target:      buildTarget,
		Type:        consts.CommandBuildModules,
		Modules:     modules,
		Srdi:        srdi,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
