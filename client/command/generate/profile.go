package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strings"
)

func ProfileShowCmd(cmd *cobra.Command, con *repl.Console) {
	resp, err := con.Rpc.GetProfiles(context.Background(), &clientpb.Empty{})
	if err != nil {
		con.Log.Errorf("get profiles failed: %s", err)
		return
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "name", Width: 20},
		{Title: "target", Width: 15},
		{Title: "type", Width: 15},
		{Title: "obfuscate", Width: 10},
		{Title: "pipeline", Width: 15},
	}, true)

	for _, p := range resp.Profiles {
		row = table.Row{
			p.Name,
			p.Target,
			p.Type,
			p.Obfuscate,
			p.PipelineId,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}

func ProfileNewCmd(cmd *cobra.Command, con *repl.Console) {
	listenerID := cmd.Flags().Arg(0)
	pipelineName := cmd.Flags().Arg(1)
	profileName := cmd.Flags().Arg(2)
	buildTarget := cmd.Flags().Arg(3)
	buildType, _ := cmd.Flags().GetString("type")
	proxy, _ := cmd.Flags().GetString("proxy")
	obfuscate, _ := cmd.Flags().GetString("obfuscate")
	modules, _ := cmd.Flags().GetStringSlice("modules")
	modulesStr := strings.Join(modules, ",")
	ca, _ := cmd.Flags().GetString("ca")

	interval, _ := cmd.Flags().GetInt("interval")
	jitter, _ := cmd.Flags().GetInt("jitter")

	params := map[string]interface{}{
		"interval": interval,
		"jitter":   jitter,
	}

	paramsJson, err := json.Marshal(params)
	if err != nil {
		con.Log.Errorf("json marshal failed: %s", err)
		return
	}
	if profileName == "" {
		profileName = fmt.Sprintf("%s-%s", buildTarget, profileName)
	}
	profile := &clientpb.Profile{
		Name:       profileName,
		Target:     buildTarget,
		Type:       buildType,
		Proxy:      proxy,
		Obfuscate:  obfuscate,
		Modules:    modulesStr,
		Ca:         ca,
		Params:     string(paramsJson),
		ListenerId: listenerID,
		PipelineId: pipelineName,
	}
	_, err = con.Rpc.NewProfile(context.Background(), profile)
	if err != nil {
		con.Log.Errorf("new profile failed: %s", err)
		return
	}
	con.Log.Infof("New profile %s success", profileName)
}
