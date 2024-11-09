package build

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"strings"
)

func BeaconCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, _, modules, ca, interval, jitter, shellcodeType := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildTarget == "" {
			return errors.New("require build target")
		}
	}
	go func() {
		_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Url:         url,
			Type:        consts.CommandBeacon,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			ShellcodeType: shellcodeType,
			Platform:      consts.Windows,
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
	name, url, buildTarget, _, modules, ca, interval, jitter, shellcodeType := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildTarget == "" {
			return errors.New("require build target")
		}
	}
	go func() {
		_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Url:         url,
			Type:        consts.CommandBind,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			ShellcodeType: shellcodeType,
			Platform:      consts.Windows,
		})
		if err != nil {
			con.Log.Errorf("Build bind failed: %v", err)
			return
		}
		con.Log.Infof("Build bind success")
	}()
	return nil
}

func ShellCodeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, _, modules, ca, interval, jitter, shellcodeType := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildTarget == "" {
			return errors.New("require build target")
		}
	}
	go func() {
		_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Url:         url,
			Type:        consts.CommandShellCode,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			ShellcodeType: shellcodeType,
			Platform:      consts.Windows,
		})
		if err != nil {
			con.Log.Errorf("Build shellcode failed: %v", err)
			return
		}
		con.Log.Infof("Build shellcode success")
	}()
	return nil
}

func PreludeCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, _, modules, ca, interval, jitter, shellcodeType := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildTarget == "" {
			return errors.New("require build target")
		}
	}
	go func() {
		_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Url:         url,
			Type:        consts.CommandPrelude,
			Target:      buildTarget,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			ShellcodeType: shellcodeType,
			Platform:      consts.Windows,
		})
		if err != nil {
			con.Log.Errorf("Build prelude failed: %v", err)
			return
		}
		con.Log.Infof("Build prelude success")
	}()
	return nil
}

func ModulesCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, _, modules, ca, interval, jitter, shellcodeType := common.ParseGenerateFlags(cmd)
	features, _ := cmd.Flags().GetStringSlice("features")
	if len(features) == 0 {
		return errors.New("require features")
	}
	if name == "" {
		if buildTarget == "" {
			return errors.New("require build target")
		}
	}
	go func() {
		_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Url:         url,
			Target:      buildTarget,
			Type:        consts.CommandModules,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			Feature:       strings.Join(features, ","),
			ShellcodeType: shellcodeType,
			Platform:      consts.Windows,
		})
		if err != nil {
			con.Log.Errorf("Build modules failed: %v", err)
			return
		}
		con.Log.Infof("Build modules success")
	}()
	return nil
}

func LoaderCmd(cmd *cobra.Command, con *repl.Console) error {
	name, url, buildTarget, _, modules, ca, interval, jitter, shellcodeType := common.ParseGenerateFlags(cmd)
	if name == "" {
		if buildTarget == "" {
			return errors.New("require build target")
		}
	}
	go func() {
		_, err := con.Rpc.Generate(context.Background(), &clientpb.Generate{
			ProfileName: name,
			Url:         url,
			Target:      buildTarget,
			Type:        consts.CommandLoader,
			Modules:     modules,
			Ca:          ca,
			Params: map[string]string{
				"interval": interval,
				"jitter":   jitter,
			},
			ShellcodeType: shellcodeType,
			Platform:      consts.Windows,
		})
		if err != nil {
			con.Log.Errorf("Build loader failed: %v", err)
			return
		}
		con.Log.Infof("Build loader success")
	}()
	return nil
}
