package build

import (
	"fmt"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"os"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
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
			"Name":      p.Name,
			"Pipeline":  p.PipelineId,
			"CreatedAt": createdDisplay,
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func ProfileLoadCmd(cmd *cobra.Command, con *repl.Console) error {
	profileName, basicPipeline := common.ParseProfileFlags(cmd)

	profilePath := cmd.Flags().Arg(0)
	content, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}

	profile := &clientpb.Profile{
		Name:       profileName,
		PipelineId: basicPipeline,
		Content:    content,
	}
	_, err = con.Rpc.NewProfile(con.Context(), profile)
	if err != nil {
		return fmt.Errorf("failed to create profile on server: %w", err)
	}
	con.Log.Infof("Successfully loaded profile '%s' for pipeline '%s'", profileName, basicPipeline)
	return nil
}

func ProfileNewCmd(cmd *cobra.Command, con *repl.Console) error {
	profileName, basicPipeline := common.ParseProfileFlags(cmd)
	profile := &clientpb.Profile{
		Name:       profileName,
		PipelineId: basicPipeline,
	}
	var params implanttypes.ProfileParams
	if cmd.Flags().Changed("rem") {
		rem, _ := cmd.Flags().GetString("rem")
		params.REMPipeline = rem
	}
	profile.Params = params.String()

	_, err := con.Rpc.NewProfile(con.Context(), profile)
	if err != nil {
		return fmt.Errorf("failed to create profile on server: %w", err)
	}

	con.Log.Infof("Successfully created new profile '%s' for pipeline '%s'", profileName, basicPipeline)
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
