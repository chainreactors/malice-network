package printer

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/constant"
	"github.com/pterm/pterm"
	"testing"
)

func TestLog(t *testing.T) {
	log := logs.NewLogger(logs.Warn)
	formatter := map[logs.Level]string{
		logs.Debug:     constant.Cloud + pterm.BgLightYellow.Sprint("[debug]") + " %s ",
		logs.Warn:      constant.Zap + pterm.BgYellow.Sprint("[warn]") + " %s ",
		logs.Info:      constant.Rocket + pterm.BgCyan.Sprint("[+]") + " %s ",
		logs.Error:     constant.Monster + pterm.BgRed.Sprint("[-]") + " %s ",
		logs.Important: constant.Fire + pterm.BgMagenta.Sprint("[*]") + " %s ",
	}

	log.SetFormatter(formatter)
	log.Info("info test")
	log.Warn("warn test")
	log.Error("error test")
	log.Debug("debug test")
	log.Important("important test")
}
