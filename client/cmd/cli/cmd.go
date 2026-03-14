package cli

import (
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
		client.Debug:     client.NewLine + tui.DarkGrayFg.Render("●") + " %s",
		client.Warn:      client.NewLine + tui.YellowFg.Bold(true).Render("●") + " %s",
		client.Important: client.NewLine + tui.PurpleFg.Bold(true).Render("●") + " %s",
		client.Info:      client.NewLine + tui.CyanFg.Render("●") + " %s",
		client.Error:     client.NewLine + tui.RedFg.Bold(true).Render("●") + " %s",
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
	if err := cmd.Execute(); err != nil {
		os.Stderr.WriteString("root command: " + err.Error() + "\n")
		os.Exit(1)
	}

	return nil
}
