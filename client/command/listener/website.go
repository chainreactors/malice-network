package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
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
	listenerID, _, portUint, certPath, keyPath, tlsEnable := common.ParsePipelineSet(cmd)
	contentType, _ := cmd.Flags().GetString("content_type")
	name := cmd.Flags().Arg(0)
	webPath := cmd.Flags().Arg(1)
	cPath := cmd.Flags().Arg(2)
	var cert, key string
	var err error
	var webAsserts *lispb.WebsiteAssets
	if listenerID == "" {
		con.Log.Error("listener_id is required")
		return
	}
	if portUint == 0 {
		rand.Seed(time.Now().UnixNano())
		portUint = uint(10000 + rand.Int31n(5001))
	}
	port := uint32(portUint)

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
	addWeb := &lispb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*lispb.WebContent{},
	}

	types.WebAddFile(addWeb, webPath, contentType, cPath)
	content, err := os.ReadFile(cPath)
	if err != nil {
		con.Log.Error(err.Error())
		return
	}
	webAsserts = &lispb.WebsiteAssets{}
	webAsserts.Assets = append(webAsserts.Assets, &lispb.WebsiteAsset{
		WebName: name,
		Content: content,
	})
	resp, err := con.LisRpc.RegisterWebsite(context.Background(), &lispb.Pipeline{
		Encryption: &lispb.Encryption{
			Enable: false,
			Type:   "",
			Key:    "",
		},
		Tls: &lispb.TLS{
			Cert:   cert,
			Key:    key,
			Enable: tlsEnable,
		},
		Body: &lispb.Pipeline_Web{
			Web: &lispb.Website{
				RootPath:   webPath,
				Port:       port,
				Name:       name,
				ListenerId: listenerID,
				Contents:   addWeb.Contents,
				Enable:     false,
			},
		},
	})

	if err != nil {
		con.Log.Error(err.Error())
	}
	webAsserts.GetAssets()[0].FileName = resp.ID
	_, err = con.LisRpc.UploadWebsite(context.Background(), webAsserts)
	if err != nil {
		con.Log.Error(err.Error())
	}
	con.Log.Importantf("Website %s added\n", name)
}

func startWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.LisRpc.StartWebsite(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		con.Log.Error(err.Error())
	}

}

func stopWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.LisRpc.StopWebsite(context.Background(), &lispb.CtrlPipeline{
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
	websites, err := con.LisRpc.ListWebsites(context.Background(), &lispb.ListenerName{
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
	if len(websites.Websites) == 0 {
		con.Log.Importantf("No websites found")
		return
	}
	for _, w := range websites.Websites {
		row = table.Row{
			w.Name,
			strconv.Itoa(int(w.Port)),
			w.RootPath,
			strconv.FormatBool(w.Enable),
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}
