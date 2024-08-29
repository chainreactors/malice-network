package listener

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/client/command/website"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func startWebsiteCmd(cmd *cobra.Command, con *console.Console) {
	certPath, _ := cmd.Flags().GetString("cert_path")
	keyPath, _ := cmd.Flags().GetString("key_path")

	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	portStr := cmd.Flags().Arg(2)
	webPath := cmd.Flags().Arg(3)
	cPath := cmd.Flags().Arg(4)
	contentType := cmd.Flags().Arg(5)

	portUint, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		log.Fatalf("Invalid port number: %v", err)
	}
	port := uint32(portUint)
	var cert, key string

	if certPath != "" && keyPath != "" {
		cert, err = cryptography.ProcessPEM(certPath)
		if err != nil {
			console.Log.Error(err.Error())
			return
		}
		key, err = cryptography.ProcessPEM(keyPath)
		if err != nil {
			console.Log.Error(err.Error())
			return
		}
	}
	cPath, _ = filepath.Abs(cPath)

	fileIfo, err := os.Stat(cPath)
	if err != nil {
		console.Log.Errorf("Error adding content %s\n", err)
		return
	}
	addWeb := &lispb.WebsiteAddContent{
		Name:     name,
		Contents: map[string]*lispb.WebContent{},
	}
	if fileIfo.IsDir() {
		website.WebAddDirectory(addWeb, webPath, cPath)
	} else {
		website.WebAddFile(addWeb, webPath, contentType, cPath)
	}
	_, err = con.Rpc.StartWebsite(context.Background(), &lispb.Pipeline{
		Tls: &lispb.TLS{
			Cert: cert,
			Key:  key,
		},
		Body: &lispb.Pipeline_Web{
			Web: &lispb.Website{
				RootPath:   webPath,
				Port:       port,
				Name:       name,
				ListenerId: listenerID,
				Contents:   addWeb.Contents,
			},
		},
	})

	if err != nil {
		console.Log.Error(err.Error())
	}
}

func stopWebsitePipelineCmd(cmd *cobra.Command, con *console.Console) {
	name := cmd.Flags().Arg(0)
	listenerID := cmd.Flags().Arg(1)
	_, err := con.Rpc.StopWebsite(context.Background(), &lispb.Website{
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		console.Log.Error(err.Error())
	}
}

func listWebsitesCmd(cmd *cobra.Command, con *console.Console) {
	listenerID := cmd.Flags().Arg(0)
	if listenerID == "" {
		console.Log.Error("listener_id is required")
		return
	}
	websites, err := con.Rpc.ListWebsites(context.Background(), &lispb.ListenerName{
		Name: listenerID,
	})
	if err != nil {
		console.Log.Error(err.Error())
		return
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		{Title: "Name", Width: 10},
		{Title: "Port", Width: 7},
		{Title: "RootPath", Width: 15},
	}, true)
	if len(websites.Websites) == 0 {
		console.Log.Importantf("No websites found")
		return
	}
	for _, w := range websites.Websites {
		row = table.Row{
			w.Name,
			strconv.Itoa(int(w.Port)),
			w.RootPath,
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
}
