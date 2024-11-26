package listener

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/webutils"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strconv"
)

func NewWebsiteCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, _, port := common.ParsePipelineFlags(cmd)
	contentType, _ := cmd.Flags().GetString("content_type")
	encryptionType, _ := cmd.Flags().GetString("encryption_type")
	parser, _ := cmd.Flags().GetString("parser")
	name := cmd.Flags().Arg(0)
	webPath := cmd.Flags().Arg(1)
	cPath := cmd.Flags().Arg(2)
	var err error
	if port == 0 {
		port = cryptography.RandomInRange(10240, 65535)
	}
	if name == "" {
		name = fmt.Sprintf("%s-web-%d", listenerID, port)
	}

	tls, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	cPath, _ = filepath.Abs(cPath)

	fileIfo, err := os.Stat(cPath)
	if err != nil {
		return err
	}
	if fileIfo.IsDir() {
		return fmt.Errorf("file is a directory")
	}
	addWeb := &clientpb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*clientpb.WebContent{},
	}

	webutils.WebAddFile(addWeb, webPath, contentType, cPath, encryptionType, parser)
	content, err := os.ReadFile(cPath)
	if err != nil {
		return err
	}
	req := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     false,
		Encryption: &clientpb.Encryption{
			Enable: false,
			Type:   "",
			Key:    "",
		},
		Tls: tls,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Root:     webPath,
				Port:     port,
				Contents: addWeb.Contents,
			},
		},
	}
	resp, err := con.Rpc.RegisterWebsite(con.Context(), req)

	if err != nil {
		return err
	}
	req.Body.(*clientpb.Pipeline_Web).Web.Contents[webPath].Content = content
	req.Body.(*clientpb.Pipeline_Web).Web.ID = resp.ID
	_, err = con.Rpc.UploadWebsite(con.Context(), req.GetWeb())
	if err != nil {
		return err
	}
	_, err = con.Rpc.StartWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	con.Log.Importantf("Website %s added\n", name)
	return nil
}

func StartWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	_, err := con.Rpc.StartWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(0)
	listenerID, _ := cmd.Flags().GetString("listener")
	_, err := con.Rpc.StopWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}

func ListWebsitesCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID := cmd.Flags().Arg(0)
	websites, err := con.Rpc.ListWebsites(con.Context(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("Name", "Name", 20),
		table.NewColumn("Port", "Port", 7),
		table.NewColumn("RootPath", "RootPath", 15),
		table.NewColumn("Enable", "Enable", 7),
	}, true)
	if len(websites.Pipelines) == 0 {
		con.Log.Importantf("No websites found")
		return nil
	}
	for _, p := range websites.Pipelines {
		w := p.GetWeb()
		row = table.NewRow(
			table.RowData{
				"Name":     p.Name,
				"Port":     strconv.Itoa(int(w.Port)),
				"RootPath": w.Root,
				"Enable":   p.Enable,
			})
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}
