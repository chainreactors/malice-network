package cli

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"os"
)

func init() {
	logs.Log.SetFormatter(core.DefaultLogStyle)
	core.Log.SetFormatter(core.DefaultLogStyle)
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	}, config.WithHookFunc(assets.HookFn))
	config.AddDriver(yaml.Driver)
}

func Start() error {
	con, err := repl.NewConsole()
	if err != nil {
		return err
	}
	cmd, err := rootCmd(con)
	if err != nil {
		return err
	}
	fmt.Print("\x1b[0m")
	if err := cmd.Execute(); err != nil {
		fmt.Printf("root command: %s\n", err)
		os.Exit(1)
	}

	return nil
}
