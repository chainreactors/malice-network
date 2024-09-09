package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/website"
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
	certPath, _ := cmd.Flags().GetString("cert_path")
	keyPath, _ := cmd.Flags().GetString("key_path")
	contentType, _ := cmd.Flags().GetString("content_type")

	name, _ := cmd.Flags().GetString("name")
	listenerID := cmd.Flags().Arg(0)
	portUint, _ := cmd.Flags().GetUint("port")
	webPath := cmd.Flags().Arg(1)
	cPath := cmd.Flags().Arg(2)
	var cert, key string
	var err error
	var tleEnable = false
	var webAsserts *lispb.WebsiteAssets
	if name == "" {
		name, err = codenames.RandomAdjective()
		if err != nil {
			repl.Log.Error(err.Error())
			return
		}
	}
	if portUint == 0 {
		rand.Seed(time.Now().UnixNano())
		portUint = uint(10000 + rand.Int31n(5001))
	}
	port := uint32(portUint)

	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			repl.Log.Error(err.Error())
			return
		}
		key, err = cryptography.ProcessPEM(keyPath)
		tleEnable = true
		if err != nil {
			repl.Log.Error(err.Error())
			return
		}
	}
	cPath, _ = filepath.Abs(cPath)

	fileIfo, err := os.Stat(cPath)
	if err != nil {
		repl.Log.Errorf("Error adding content %s\n", err)
		return
	}
	addWeb := &lispb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*lispb.WebContent{},
	}
	if fileIfo.IsDir() {
		webAsserts = website.WebAddDirectory(addWeb, webPath, cPath)
	} else {
		website.WebAddFile(addWeb, webPath, contentType, cPath)
		content, err := os.ReadFile(cPath)
		if err != nil {
			repl.Log.Error(err.Error())
			return
		}
		webAsserts.Assets = append(webAsserts.Assets, &lispb.WebsiteAsset{
			WebName:  name,
			Content:  content,
			FileName: filepath.Base(cPath),
		})
	}
	_, err = con.Rpc.NewWebsite(context.Background(), &lispb.Pipeline{
		Tls: &lispb.TLS{
			Cert:   cert,
			Key:    key,
			Enable: tleEnable,
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
		repl.Log.Error(err.Error())
	}

	_, err = con.Rpc.UploadWebsite(context.Background(), webAsserts)
	if err != nil {
		repl.Log.Error(err.Error())
	}
}

func startWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.Rpc.StartWebsite(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		repl.Log.Error(err.Error())
	}

}

func stopWebsitePipelineCmd(cmd *cobra.Command, con *repl.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.Rpc.StopWebsite(context.Background(), &lispb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		repl.Log.Error(err.Error())
	}
}

func listWebsitesCmd(cmd *cobra.Command, con *repl.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		repl.Log.Error("listener_id is required")
		return
	}
	websites, err := con.Rpc.ListWebsites(context.Background(), &lispb.ListenerName{
		Name: listenerID,
	})
	if err != nil {
		repl.Log.Error(err.Error())
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
		repl.Log.Importantf("No websites found")
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
