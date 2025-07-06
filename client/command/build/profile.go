package build

import (
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
	"time"
)

func ProfileShowCmd(cmd *cobra.Command, con *repl.Console) error {
	resp, err := con.Rpc.GetProfiles(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(resp.Profiles) == 0 {
		con.Log.Info("No profiles found")
		return nil
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Pipeline", "Pipeline", 16),
		table.NewColumn("Pulse Pipeline", "Pulse Pipeline", 16),
		table.NewColumn("CreatedAt", "CreatedAt", 16),
	}, true)

	var rowEntries []table.Row
	for _, p := range resp.Profiles {
		// Format creation time
		createdDisplay := "-"
		if p.CreatedAt > 0 {
			createdTime := time.Unix(p.CreatedAt, 0)
			createdDisplay = createdTime.Format("2006-01-02 15:04")
		}

		row := table.NewRow(table.RowData{
			"Name":           p.Name,
			"Pipeline":       p.PipelineId,
			"Pulse Pipeline": p.PulsePipelineId,
			"CreatedAt":      createdDisplay,
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func ProfileLoadCmd(cmd *cobra.Command, con *repl.Console) error {
	profileName, basicPipeline, pulsePipeline := common.ParseProfileFlags(cmd)

	profilePath := cmd.Flags().Arg(0)
	content, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}

	profile := &clientpb.Profile{
		Name:            profileName,
		PipelineId:      basicPipeline,
		PulsePipelineId: pulsePipeline,
		Content:         content,
	}
	_, err = con.Rpc.NewProfile(con.Context(), profile)
	if err != nil {
		return err
	}
	con.Log.Infof("load implant profile %s for %s\n", profileName, basicPipeline)
	return nil
}

func ProfileNewCmd(cmd *cobra.Command, con *repl.Console) error {
	profileName, basicPipeline, pulsePipeline := common.ParseProfileFlags(cmd)
	profile := &clientpb.Profile{
		Name:            profileName,
		PipelineId:      basicPipeline,
		PulsePipelineId: pulsePipeline,
	}
	_, err := con.Rpc.NewProfile(con.Context(), profile)
	if err != nil {
		return err
	}
	con.Log.Infof("create new profile %s for %s success\n", profileName, basicPipeline)
	return nil
}

func ProfileDeleteCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	_, err := con.Rpc.DeleteProfile(con.Context(), &clientpb.Profile{
		Name: name,
	})
	if err != nil {
		return err
	}
	con.Log.Infof("delete profile %s success\n", name)
	return nil
}
