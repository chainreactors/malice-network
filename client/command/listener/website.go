package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/utils/webutils"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strconv"
)

func NewWebsiteCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, _, port := common.ParsePipelineFlags(cmd)
	contentType, _ := cmd.Flags().GetString("content_type")
	name := cmd.Flags().Arg(0)
	webPath := cmd.Flags().Arg(1)
	cPath := cmd.Flags().Arg(2)
	var err error
	var webAsserts *clientpb.WebsiteAssets
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

	webutils.WebAddFile(addWeb, webPath, contentType, cPath)
	content, err := os.ReadFile(cPath)
	if err != nil {
		return err
	}
	webAsserts = &clientpb.WebsiteAssets{}
	webAsserts.Assets = append(webAsserts.Assets, &clientpb.WebsiteAsset{
		WebName: name,
		Content: content,
	})
	resp, err := con.Rpc.RegisterWebsite(context.Background(), &clientpb.Pipeline{
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
	})

	if err != nil {
		return err
	}
	webAsserts.GetAssets()[0].FileName = resp.ID
	_, err = con.Rpc.UploadWebsite(context.Background(), webAsserts)
	if err != nil {
		return err
	}
	_, err = con.Rpc.StartWebsite(context.Background(), &clientpb.CtrlPipeline{
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
	_, err := con.Rpc.StartWebsite(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) error {
	name := cmd.Flags().Arg(1)
	listenerID, _ := cmd.Flags().GetString("listener")
	_, err := con.Rpc.StopWebsite(context.Background(), &clientpb.CtrlPipeline{
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
	websites, err := con.Rpc.ListWebsites(context.Background(), &clientpb.Listener{
		Id: listenerID,
	})
	if err != nil {
		return err
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "Port", Width: 7},
		{Title: "RootPath", Width: 15},
		{Title: "Enable", Width: 7},
	}, true)
	if len(websites.Pipelines) == 0 {
		con.Log.Importantf("No websites found")
		return nil
	}
	for _, p := range websites.Pipelines {
		w := p.GetWeb()
		row = table.Row{
			w.ID,
			strconv.Itoa(int(w.Port)),
			w.Root,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}
