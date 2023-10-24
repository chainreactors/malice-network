package main

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/pterm/pterm"
)

func init() {
	logs.Log.SetFormatter(map[logs.Level]string{
		logs.Debug:     consts.Cloud + pterm.BgLightYellow.Sprint("[debug]") + " %s ",
		logs.Warn:      consts.Zap + pterm.BgYellow.Sprint("[warn]") + " %s ",
		logs.Info:      consts.Rocket + pterm.BgCyan.Sprint("[+]") + " %s ",
		logs.Error:     consts.Monster + pterm.BgRed.Sprint("[-]") + " %s ",
		logs.Important: consts.Fire + pterm.BgMagenta.Sprint("[*]") + " %s ",
	})
	console.Log.SetFormatter(map[logs.Level]string{
		logs.Debug:     consts.Cloud + pterm.BgLightYellow.Sprint("[debug]") + " %s ",
		logs.Warn:      consts.Zap + pterm.BgYellow.Sprint("[warn]") + " %s ",
		logs.Info:      consts.Rocket + pterm.BgCyan.Sprint("[+]") + " %s ",
		logs.Error:     consts.Monster + pterm.BgRed.Sprint("[-]") + " %s ",
		logs.Important: consts.Fire + pterm.BgMagenta.Sprint("[*]") + " %s ",
	})
}

func main() {
	err := console.Start(command.BindCommands)
	if err != nil {
		return
	}
}
