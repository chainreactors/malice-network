package build

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
)

func ProfileShowCmd(cmd *cobra.Command, con *repl.Console) error {
	resp, err := con.Rpc.GetProfiles(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(resp.Profiles) == 0 {
		con.Log.Info("No profiles")
		return nil
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Target", "Target", 15),
		table.NewColumn("Type", "Type", 15),
		table.NewColumn("Obfuscate", "Obfuscate", 10),
		table.NewColumn("Pipeline", "Pipeline", 15),
	}, true)

	for _, p := range resp.Profiles {
		row = table.NewRow(
			table.RowData{
				"Name":      p.Name,
				"Target":    p.Target,
				"Type":      p.Type,
				"Obfuscate": p.Obfuscate,
				"Pipeline":  p.PipelineId,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}

func ProfileNewCmd(cmd *cobra.Command, con *repl.Console) error {
	profileName, pipelineName := common.ParseProfileFlags(cmd)

	profilePath := cmd.Flags().Arg(0)
	content, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}

	if profileName == "" {
		profileName = fmt.Sprintf("%s", codenames.GetCodename())
	}
	profile := &clientpb.Profile{
		Name:       profileName,
		PipelineId: pipelineName,
		Content:    content,
	}
	_, err = con.Rpc.NewProfile(con.Context(), profile)
	if err != nil {
		return err
	}
	con.Log.Infof("load new profile %s success\n", profileName)
	return nil
}
