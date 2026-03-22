package server

import (
	"fmt"
	"os"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/server/assets"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
)

func init() {
	err := configs.InitConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		return
	}
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yaml.Driver)
	codenames.SetupCodenames()
	assets.SetupGithubFile()
}

func Start(defaultConfig []byte) error {
	var opt Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.SubcommandsOptional = true
	args, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		return nil
	}
	if _, statErr := os.Stat(opt.Config); opt.Quickstart || os.IsNotExist(statErr) {
		if err := RunQuickstart(&opt); err != nil {
			return fmt.Errorf("quickstart failed: %w", err)
		}
	}

	err = opt.PrepareConfig(defaultConfig)
	if err != nil {
		return err
	}

	if parser.Active != nil {
		err = opt.Execute(args, parser)
		if err != nil {
			logs.Log.Error(err)
		}
		return nil
	}

	if !opt.ListenerOnly && opt.Server.Enable {
		err = opt.PrepareServer()
		if err != nil {
			return fmt.Errorf("cannot prepare server, %s", err.Error())
		}
	}

	if !opt.ServerOnly && opt.Listeners.Enable {
		err = opt.PrepareListener()
		if err != nil {
			return fmt.Errorf("cannot prepare listener, %s", err.Error())
		}
	}
	return opt.Handler()
}
