package saas

import (
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, _, params := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := con.Rpc.Build(con.Context(), &clientpb.BuildConfig{
			ProfileName: name,
			MaleficHost: address,
			Type:        consts.CommandBuildBeacon,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params:      params.String(),
			Srdi:        true,
			Resource:    consts.ArtifactFromSaas,
		})
		if err != nil {
			con.Log.Errorf("Build beacon failed: %v\n", err)
			return
		}
	}()
	return nil
}

func BindCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, _, params := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := con.Rpc.Build(con.Context(), &clientpb.BuildConfig{
			ProfileName: name,
			MaleficHost: address,
			Type:        consts.CommandBuildBind,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params:      params.String(),
			Srdi:        true,
			Resource:    consts.ArtifactFromSaas,
		})
		if err != nil {
			con.Log.Errorf("Build bind failed: %v\n", err)
			return
		}
	}()
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, ca, _, _ := common.ParseGenerateFlags(cmd)
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
		_, err := con.Rpc.Build(con.Context(), &clientpb.BuildConfig{
			ProfileName: name,
			MaleficHost: address,
			Type:        consts.CommandBuildPrelude,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Srdi:        true,
			Bin:         file,
			Resource:    consts.ArtifactFromSaas,
		})
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v\n", err)
			return
		}
	}()
	return nil
}

func ModulesCmd(cmd *cobra.Command, con *repl.Console) error {
	name, address, buildTarget, modules, _, srdi, _ := common.ParseGenerateFlags(cmd)
	if buildTarget == "" {
		return errors.New("require build target")
	}
	go func() {
		_, err := BuildModules(con, name, address, buildTarget, modules, srdi)
		if err != nil {
			con.Log.Errorf("Build modules failed: %v\n", err)
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
		_, err := con.Rpc.Build(con.Context(), &clientpb.BuildConfig{
			ProfileName: profile,
			MaleficHost: address,
			Target:      buildTarget,
			Type:        consts.CommandBuildPulse,
			Srdi:        true,
			ArtifactId:  artifactId,
			Resource:    consts.ArtifactFromSaas,
		})
		if err != nil {
			con.Log.Errorf("Build loader failed: %v\n", err)
			return
		}
	}()
	return nil
}
func BuildModules(con *repl.Console, name, address, buildTarget string, modules []string, srdi bool) (bool, error) {
	_, err := con.Rpc.Build(con.Context(), &clientpb.BuildConfig{
		ProfileName: name,
		MaleficHost: address,
		Target:      buildTarget,
		Type:        consts.CommandBuildModules,
		Modules:     modules,
		Srdi:        srdi,
		Resource:    consts.ArtifactFromSaas,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}
