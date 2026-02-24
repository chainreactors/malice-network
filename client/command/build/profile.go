package build

import (
	"fmt"
	"os"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ProfileShowCmd(cmd *cobra.Command, con *core.Console) error {
	resp, err := con.Rpc.GetProfiles(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(resp.Profiles) == 0 {
		con.Log.Info("No profiles found")
		return nil
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Name", "Name", 1),
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

func ProfileLoadCmd(cmd *cobra.Command, con *core.Console) error {
	profileName, basicPipeline := common.ParseProfileFlags(cmd)

	profilePath := cmd.Flags().Arg(0)
	content, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}

	profile := &clientpb.Profile{
		Name:          profileName,
		PipelineId:    basicPipeline,
		ImplantConfig: content,
	}
	_, err = con.Rpc.NewProfile(con.Context(), profile)
	if err != nil {
		return fmt.Errorf("failed to create profile on server: %w", err)
	}
	con.Log.Infof("Successfully loaded profile '%s' for pipeline '%s'", profileName, basicPipeline)
	return nil
}

func ProfileNewCmd(cmd *cobra.Command, con *core.Console) error {
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

func ProfileDeleteCmd(cmd *cobra.Command, con *core.Console) error {
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

func ProfileDetailCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	if name == "" {
		return fmt.Errorf("profile name is required")
	}

	profile, err := con.Rpc.GetProfileByName(con.Context(), &clientpb.Profile{Name: name})
	if err != nil {
		return fmt.Errorf("failed to get profile '%s': %w", name, err)
	}

	// 1. 展示元数据 KV 表
	printProfileMetadata(profile)

	// 2. 展示 implant.yaml 内容
	if len(profile.ImplantConfig) > 0 {
		con.Log.Console("\n--- implant.yaml ---\n\n")
		con.Log.Console(string(profile.ImplantConfig))
		con.Log.Console("\n")
	}

	// 3. 展示 prelude.yaml 内容（如果存在）
	if len(profile.PreludeConfig) > 0 {
		con.Log.Console("\n--- prelude.yaml ---\n\n")
		con.Log.Console(string(profile.PreludeConfig))
		con.Log.Console("\n")
	}

	// 4. 展示 resources 列表（如果存在）
	if profile.Resources != nil && len(profile.Resources.Entries) > 0 {
		con.Log.Console("\n--- resources ---\n\n")
		tableModel := tui.NewTable([]table.Column{
			table.NewFlexColumn("Filename", "Filename", 1),
			table.NewColumn("Size", "Size", 12),
		}, true)

		var rowEntries []table.Row
		for _, entry := range profile.Resources.Entries {
			row := table.NewRow(table.RowData{
				"Filename": entry.Filename,
				"Size":     fileutils.Bytes(uint64(len(entry.Content))),
			})
			rowEntries = append(rowEntries, row)
		}
		tableModel.SetMultiline()
		tableModel.SetRows(rowEntries)
		con.Log.Console(tableModel.View())
	}

	return nil
}

func printProfileMetadata(profile *clientpb.Profile) {
	createdDisplay := "-"
	if profile.CreatedAt > 0 {
		createdTime := time.Unix(profile.CreatedAt, 0)
		createdDisplay = createdTime.Format("2006-01-02 15:04:05")
	}

	data := map[string]interface{}{
		"Name":      profile.Name,
		"Pipeline":  profile.PipelineId,
		"Params":    profile.Params,
		"CreatedAt": createdDisplay,
	}
	orderedKeys := []string{"Name", "Pipeline", "Params", "CreatedAt"}
	tui.RenderKVWithOptions(data, orderedKeys, tui.KVOptions{ShowHeader: true})
}
