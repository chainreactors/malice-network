package website

import (
	"github.com/chainreactors/malice-network/client/core"
	"strconv"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

// NewWebsiteCmd - 创建新的网站
func NewWebsiteCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	root, _ := cmd.Flags().GetString("root")
	listenerID, _, host, port := common.ParsePipelineFlags(cmd)
	if port == 0 {
		port = cryptography.RandomInRange(10240, 65535)
	}
	tls, certName, err := common.ParseTLSFlags(cmd)
	if err != nil {
		return err
	}
	return NewWebsite(con, name, root, host, port, listenerID, certName, tls)
}

// NewWebsite
func NewWebsite(con *core.Console, websiteName, root, host string, port uint32, listenerId, certName string, tls *clientpb.TLS) error {
	var err error
	if root == "" {
		root = "/"
	}
	host = "0.0.0.0"
	req := &clientpb.Pipeline{
		Name:       websiteName,
		ListenerId: listenerId,
		Enable:     false,
		Tls:        tls,
		CertName:   certName,
		Ip:         host, // this has not taken effect yet
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
				Name:     websiteName,
				Root:     root,
				Port:     port,
				Contents: make(map[string]*clientpb.WebContent),
			},
		},
	}
	_, err = con.Rpc.RegisterWebsite(con.Context(), req)
	if err != nil {
		return err
	}

	_, err = con.Rpc.StartWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       websiteName,
		ListenerId: listenerId,
		Pipeline:   req,
	})
	if err != nil {
		return err
	}
	con.Log.Importantf("Website %s created on port %d\n", websiteName, port)
	return nil
}

// StartWebsitePipelineCmd
func StartWebsitePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	websiteName := cmd.Flags().Arg(0)
	certName, _ := cmd.Flags().GetString("cert-name")
	return StartWebsite(con, websiteName, certName)
}

func StartWebsite(con *core.Console, websiteName, certName string) error {
	if _, ok := con.Pipelines[websiteName]; ok {
		_, err := con.Rpc.StopWebsite(con.Context(), &clientpb.CtrlPipeline{
			Name: websiteName,
		})
		if err != nil {
			return err
		}
	}
	_, err := con.Rpc.StartWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       websiteName,
		ListenerId: "",
		CertName:   certName,
	})
	if err != nil {
		return err
	}
	return nil
}

func StopWebsitePipelineCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	return StopWebsite(con, name)
}

// StopWebsite
func StopWebsite(con *core.Console, name string) error {
	_, err := con.Rpc.StopWebsite(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: "",
	})
	if err != nil {
		return err
	}
	return nil
}

func ListWebsitesCmd(cmd *cobra.Command, con *core.Console) error {
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
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Port", "Port", 7),
		table.NewFlexColumn("RootPath", "RootPath", 1),
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
	con.Log.Console(tableModel.View())
	return nil
}
