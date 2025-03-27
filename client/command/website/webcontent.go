package website

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"strconv"
)

// AddWebContentCmd - 添加网站内容
func AddWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	filePath := cmd.Flags().Arg(0)
	websiteName, _ := cmd.Flags().GetString("website")
	webPath, _ := cmd.Flags().GetString("path")
	contentType, _ := cmd.Flags().GetString("type")
	parser, encryption := common.ParseEncryptionFlags(cmd)
	if webPath == "" {
		webPath = "/" + filepath.Base(filePath)
	}

	_, err := AddWebContent(con, filePath, webPath, websiteName, contentType, parser, encryption)
	if err != nil {
		return err
	}
	con.Log.Importantf("Content added to website %s: %s -> %s\n", websiteName, webPath, filePath)
	return nil
}

func AddWebContent(con *repl.Console, localFile, webPath, webPipe, typ, parser string, enc *clientpb.Encryption) (bool, error) {
	content, err := os.ReadFile(localFile)
	if err != nil {
		return false, err
	}

	website := &clientpb.Website{
		Contents: map[string]*clientpb.WebContent{
			webPath: {
				WebsiteId:   webPipe,
				File:        localFile,
				Path:        webPath,
				Type:        parser,
				Content:     content,
				Encryption:  enc,
				ContentType: typ,
			},
		},
	}
	_, err = con.Rpc.AddWebsiteContent(con.Context(), website)
	if err != nil {
		return false, err
	}

	return true, nil
}

// UpdateWebContentCmd - 更新网站内容
func UpdateWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	contentId := cmd.Flags().Arg(0)
	filePath := cmd.Flags().Arg(1)
	websiteName, _ := cmd.Flags().GetString("website")
	contentType, _ := cmd.Flags().GetString("type")
	parser, encryption := common.ParseEncryptionFlags(cmd)

	_, err := UpdateWebContent(con, contentId, filePath, websiteName, contentType, parser, encryption)
	if err != nil {
		return err
	}
	con.Log.Importantf("Content %s updated in website %s\n", contentId, websiteName)
	return nil
}

func UpdateWebContent(con *repl.Console, contentId, localFile, webPipe, typ, parser string, enc *clientpb.Encryption) (bool, error) {
	content, err := os.ReadFile(localFile)
	if err != nil {
		return false, err
	}

	website := &clientpb.WebContent{
		Id:          contentId,
		WebsiteId:   webPipe,
		File:        localFile,
		Type:        parser,
		Content:     content,
		Encryption:  enc,
		ContentType: typ,
	}
	_, err = con.Rpc.UpdateWebsiteContent(con.Context(), website)
	if err != nil {
		return false, err
	}
	return true, nil
}

// RemoveWebContentCmd - 删除网站内容
func RemoveWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	contentId := cmd.Flags().Arg(0)

	_, err := RemoveWebContent(con, contentId)
	if err != nil {
		return err
	}

	con.Log.Importantf("Content %s removed\n", contentId)
	return nil
}

func RemoveWebContent(con *repl.Console, contentId string) (bool, error) {
	webContent := &clientpb.WebContent{
		Id: contentId,
	}

	_, err := con.Rpc.RemoveWebsiteContent(con.Context(), webContent)
	if err != nil {
		return false, err
	}

	return true, nil
}

// ListWebContentCmd - 列出网站内容
func ListWebContentCmd(cmd *cobra.Command, con *repl.Console) error {
	websiteName := cmd.Flags().Arg(0)

	website := &clientpb.Website{
		Name: websiteName,
	}

	contents, err := con.Rpc.ListWebContent(con.Context(), website)
	if err != nil {
		return err
	}

	if len(contents.Contents) == 0 {
		con.Log.Importantf("No content found in website %s\n", websiteName)
		return nil
	}

	var rowEntries []table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 8),
		table.NewColumn("WebsiteName", "WebsiteName", 15),
		table.NewColumn("ListenerID", "ListenerID", 15),
		table.NewColumn("Path", "Path", 20),
		table.NewColumn("Type", "Type", 10),
		table.NewColumn("Size", "Size", 8),
		table.NewColumn("ContentType", "ContentType", 30),
	}, true)

	for _, content := range contents.Contents {
		row := table.NewRow(table.RowData{
			"ID":          content.Id[:8],
			"WebsiteName": content.WebsiteId,
			"ListenerID":  content.ListenerId,
			"Path":        content.Path,
			"Type":        content.Type,
			"Size":        strconv.FormatUint(content.Size, 10),
			"ContentType": content.ContentType,
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	fmt.Printf(tableModel.View())
	return nil
}
