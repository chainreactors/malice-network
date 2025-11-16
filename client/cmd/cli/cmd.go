package cli

import (
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"os"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

func init() {
	logs.Log.SetFormatter(client.DefaultLogStyle)
	client.Log.SetFormatter(client.DefaultLogStyle)
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	}, config.WithHookFunc(assets.HookFn))
	config.AddDriver(yaml.Driver)
}

func Start() error {
	con, err := core.NewConsole()
	if err != nil {
		return err
	}
	cryptography.InitAES("")
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
