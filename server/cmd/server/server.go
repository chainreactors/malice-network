package server

import (
	"fmt"
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
		fmt.Println(err.Error())
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
			fmt.Println(err.Error())
		}
		return nil
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

	go opt.Handler()
	select {}
}
