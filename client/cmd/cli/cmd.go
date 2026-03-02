package cli

import (
	"fmt"
	"os"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/tui"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

func init() {
	styledLogStyle := map[logs.Level]string{
		client.Debug:     client.NewLine + tui.BlueBg.Bold(true).Render(tui.Rocket+"[+]") + " %s",
		client.Warn:      client.NewLine + tui.YellowBg.Bold(true).Render(tui.Zap+"[warn]") + " %s",
		client.Important: client.NewLine + tui.PurpleBg.Bold(true).Render(tui.Fire+"[*]") + " %s",
		client.Info:      client.NewLine + tui.GreenBg.Bold(true).Render(tui.HotSpring+"[i]") + " %s",
		client.Error:     client.NewLine + tui.RedBg.Bold(true).Render(tui.Monster+"[-]") + " %s",
	}
	logs.Log.SetFormatter(styledLogStyle)
	client.Log.SetFormatter(styledLogStyle)
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
