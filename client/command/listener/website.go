package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func newWebsiteCmd(cmd *cobra.Command, con *repl.Console) {
	name, _, portUint, certPath, keyPath, tlsEnable := common.ParsePipelineFlags(cmd)
	contentType, _ := cmd.Flags().GetString("content_type")
	listenerID := cmd.Flags().Arg(0)
	webPath := cmd.Flags().Arg(1)
	cPath := cmd.Flags().Arg(2)
	var cert, key string
	var err error
	var webAsserts *clientpb.WebsiteAssets
	if portUint == 0 {
		rand.Seed(time.Now().UnixNano())
		portUint = uint(15001 + rand.Int31n(5001))
	}
	port := uint32(portUint)
	if name == "" {
		name = fmt.Sprintf("%s-web-%d", listenerID, port)
	}
	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			con.Log.Error(err.Error())
			return
		}
		key, err = cryptography.ProcessPEM(keyPath)
		tlsEnable = true
		if err != nil {
			con.Log.Error(err.Error())
			return
		}
	}
	cPath, _ = filepath.Abs(cPath)

	fileIfo, err := os.Stat(cPath)
	if err != nil {
		con.Log.Errorf("Error adding content %s\n", err)
		return
	}
	if fileIfo.IsDir() {
		con.Log.Errorf("Error adding content %s\n", "file is a directory")
		return
	}
	addWeb := &clientpb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*clientpb.WebContent{},
	}

	types.WebAddFile(addWeb, webPath, contentType, cPath)
	content, err := os.ReadFile(cPath)
	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	webAsserts = &clientpb.WebsiteAssets{}
	webAsserts.Assets = append(webAsserts.Assets, &clientpb.WebsiteAsset{
		WebName: name,
		Content: content,
	})
	resp, err := con.LisRpc.RegisterWebsite(context.Background(), &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     false,
		Encryption: &clientpb.Encryption{
			Enable: false,
			Type:   "",
			Key:    "",
		},
		Tls: &clientpb.TLS{
			Cert:   cert,
			Key:    key,
			Enable: tlsEnable,
		},
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				RootPath: webPath,
				Port:     port,
				Contents: addWeb.Contents,
			},
		},
	})

	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	webAsserts.GetAssets()[0].FileName = resp.ID
	_, err = con.LisRpc.UploadWebsite(context.Background(), webAsserts)
	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	_, err = con.LisRpc.StartWebsite(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	con.Log.Importantf("Website %s added\n", name)
}

func startWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.LisRpc.StartWebsite(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
	}

}

func stopWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(1)
	listenerID := cmd.Flags().Arg(0)
	_, err := con.LisRpc.StopWebsite(context.Background(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
	}
}

func listWebsitesCmd(cmd *cobra.Command, con *repl.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		con.Log.Error("listener_id is required")
		return
	}
	websites, err := con.LisRpc.ListWebsites(context.Background(), &clientpb.ListenerName{
		Name: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
		return
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
		return
	}
	for _, p := range websites.Pipelines {
		w := p.GetWeb()
		row = table.Row{
			w.ID,
			strconv.Itoa(int(w.Port)),
			w.RootPath,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}
