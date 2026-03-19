package config

import (
	"strings"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

func GetNotifyCmd(cmd *cobra.Command, con *core.Console) error {
	notify, err := con.Rpc.GetNotifyConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}

	values := map[string]string{
		"Telegram":   enabledStatus(notify.TelegramEnable),
		"DingTalk":   enabledStatus(notify.DingtalkEnable),
		"Lark":       enabledStatus(notify.LarkEnable),
		"ServerChan": enabledStatus(notify.ServerchanEnable),
		"PushPlus":   enabledStatus(notify.PushplusEnable),
	}
	keys := []string{"Telegram", "DingTalk", "Lark", "ServerChan", "PushPlus"}
	con.Log.Console(common.NewKVTable("Notify", keys, values).View() + "\n")
	return nil
}

func enabledStatus(enabled bool) string {
	if enabled {
		return tui.GreenFg.Render("Enabled")
	}
	return tui.RedFg.Render("Disabled")
}

// notifyEnabledProviders returns a comma-separated list of enabled providers.
func notifyEnabledProviders(notify *clientpb.Notify) string {
	if notify == nil {
		return "None"
	}
	var providers []string
	if notify.TelegramEnable {
		providers = append(providers, "Telegram")
	}
	if notify.DingtalkEnable {
		providers = append(providers, "DingTalk")
	}
	if notify.LarkEnable {
		providers = append(providers, "Lark")
	}
	if notify.ServerchanEnable {
		providers = append(providers, "ServerChan")
	}
	if notify.PushplusEnable {
		providers = append(providers, "PushPlus")
	}
	if len(providers) == 0 {
		return "None"
	}
	return strings.Join(providers, ", ")
}

func UpdateNotifyCmd(cmd *cobra.Command, con *core.Console) error {
	current, err := con.Rpc.GetNotifyConfig(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}

	notify := mergeNotifyUpdate(current, cmd)
	_, err = UpdateNotify(con, notify)
	if err != nil {
		return err
	}
	con.Log.Console("Update notify config success\n")
	return nil
}

func UpdateNotify(con *core.Console, notify *clientpb.Notify) (*clientpb.Empty, error) {
	return con.Rpc.UpdateNotifyConfig(con.Context(), notify)
}
