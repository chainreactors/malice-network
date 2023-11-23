package printer

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/pterm/pterm"
	"testing"
)

func TestLog(t *testing.T) {
	log := logs.NewLogger(logs.Warn)
	formatter := map[logs.Level]string{
		logs.Debug:     consts.Cloud + pterm.BgLightYellow.Sprint("[debug]") + " %s ",
		logs.Warn:      consts.Zap + pterm.BgYellow.Sprint("[warn]") + " %s ",
		logs.Info:      consts.Rocket + pterm.BgCyan.Sprint("[+]") + " %s ",
		logs.Error:     consts.Monster + pterm.BgRed.Sprint("[-]") + " %s ",
		logs.Important: consts.Fire + pterm.BgMagenta.Sprint("[*]") + " %s ",
	}

	log.SetFormatter(formatter)
	log.Info("info test")
	log.Warn("warn test")
	log.Error("error test")
	log.Debug("debug test")
	log.Important("important test")
}
