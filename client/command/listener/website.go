package listener

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

// NewWebsiteCmd - 创建新的网站
func NewWebsiteCmd(cmd *cobra.Command, con *repl.Console) error {
	listenerID, _, port := common.ParsePipelineFlags(cmd)
	name := cmd.Flags().Arg(0)
	root, _ := cmd.Flags().GetString("root")

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

	req := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     false,
		Tls:        tls,
		Body: &clientpb.Pipeline_Web{
			Web: &clientpb.Website{
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
		Name:       name,
		ListenerId: listenerID,
	})
	if err != nil {
		return err
	}
	con.Log.Importantf("Website %s created on port %d\n", name, port)
	return nil
}

// AddWebContentCmd - 添加网站内容
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

// AddWebContentCmd - 添加网站内容
func AddWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	filePath := cmd.Flags().Arg(0)
	websiteName, _ := cmd.Flags().GetString("website")
	webPath, _ := cmd.Flags().GetString("path")
	contentType, _ := cmd.Flags().GetString("type")

	if webPath == "" {
		webPath = "/" + filepath.Base(filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	website := &clientpb.Website{
		Host: websiteName,
		Contents: map[string]*clientpb.WebContent{
			webPath: {
				WebsiteId: websiteName,
				File:      filePath,
				Path:      webPath,
				Type:      contentType,
				Content:   content,
			},
		},
	}

	_, err = con.Rpc.WebsiteAddContent(con.Context(), website)
	if err != nil {
		return err
	}

	con.Log.Importantf("Content added to website %s: %s -> %s\n", websiteName, webPath, filePath)
	return nil
}

// UpdateWebContentCmd - 更新网站内容
func UpdateWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	contentId := cmd.Flags().Arg(0)
	filePath := cmd.Flags().Arg(1)
	websiteName, _ := cmd.Flags().GetString("website")
	contentType, _ := cmd.Flags().GetString("type")

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	webContent := &clientpb.WebContent{
		Id:        contentId,
		WebsiteId: websiteName,
		File:      filepath.Base(filePath),
		Type:      contentType,
		Content:   content,
	}

	_, err = con.Rpc.WebsiteUpdateContent(con.Context(), webContent)
	if err != nil {
		return err
	}

	con.Log.Importantf("Content %s updated in website %s\n", contentId, websiteName)
	return nil
}

// RemoveWebContentCmd - 删除网站内容
func RemoveWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	contentId := cmd.Flags().Arg(0)

	webContent := &clientpb.WebContent{
		Id: contentId,
	}

	_, err := con.Rpc.WebsiteRemoveContent(con.Context(), webContent)
	if err != nil {
		return err
	}

	con.Log.Importantf("Content %s removed\n", contentId)
	return nil
}

// ListWebContentCmd - 列出网站内容
func ListWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	websiteName := cmd.Flags().Arg(0)

	website := &clientpb.Website{
		Host: websiteName,
	}

	websites, err := con.Rpc.ListWebContent(con.Context(), website)
	if err != nil {
		return err
	}

	if len(websites.Websites) == 0 {
		con.Log.Importantf("No content found in website %s\n", websiteName)
		return nil
	}

	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("WebsiteName", "WebsiteName", 20),
		table.NewColumn("ListenerID", "ListenerID", 20),
		table.NewColumn("ContentID", "ContentID", 36),
		table.NewColumn("Path", "Path", 20),
		table.NewColumn("Type", "Type", 15),
		table.NewColumn("Size", "Size", 10),
		table.NewColumn("ContentType", "ContentType", 15),
	}, true)

	for _, website := range websites.Websites {
		for path, content := range website.Contents {
			row := table.NewRow(table.RowData{
				"WebsiteName": websiteName,
				"ListenerID":  content.WebsiteId,
				"ContentID":   content.Id,
				"Path":        path,
				"Type":        content.Type,
				"Size":        strconv.FormatUint(content.Size, 10),
				"ContentType": content.ContentType,
			})
			rowEntries = append(rowEntries, row)
		}
	}

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}
