package website

import (
	"errors"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/utils/pe"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// AddWebContentCmd - 添加网站内容
func AddWebContentCmd(cmd *cobra.Command, con *core.Console) error {
	filePath := cmd.Flags().Arg(0)
	websiteName, _ := cmd.Flags().GetString("website")
	webPath, _ := cmd.Flags().GetString("path")
	contentType, _ := cmd.Flags().GetString("type")
	auth, _ := cmd.Flags().GetString("auth")
	name, _ := cmd.Flags().GetString("name")
	comment, _ := cmd.Flags().GetString("comment")
	artifactName, _ := cmd.Flags().GetString("artifact")
	format, _ := cmd.Flags().GetString("format")
	rdi, _ := cmd.Flags().GetString("RDI")

	if artifactName != "" {
		if !cmd.Flags().Changed("type") {
			contentType = "application/octet-stream"
		}
		_, err := AddArtifactContent(con, artifactName, websiteName, format, rdi, webPath, contentType, name, comment, auth)
		return err
	}

	if webPath == "" {
		webPath = "/" + filepath.Base(filePath)
	}
	if name == "" {
		name = filepath.Base(filePath)
	}

	_, err := AddWebContentWithMetadata(con, filePath, webPath, websiteName, contentType, auth, name, comment)
	if err != nil {
		return err
	}
	return nil
}

func AddWebContent(con *core.Console, localFile, webPath, webPipe, typ, auth string) (*clientpb.WebContent, error) {
	return AddWebContentWithMetadata(con, localFile, webPath, webPipe, typ, auth, "", "")
}

func AddWebContentWithMetadata(con *core.Console, localFile, webPath, webPipe, typ, auth, name, comment string) (*clientpb.WebContent, error) {
	content, err := pe.Unpack(localFile)
	if err != nil {
		return nil, err
	}
	websiteName, listenerID, _ := resolveWebsiteTarget(con, webPipe)

	website := &clientpb.Website{
		Name:       websiteName,
		ListenerId: listenerID,
		Contents: map[string]*clientpb.WebContent{
			webPath: {
				WebsiteId:   websiteName,
				ListenerId:  listenerID,
				File:        localFile,
				Path:        webPath,
				Name:        name,
				Content:     content,
				ContentType: typ,
				Comment:     comment,
				Auth:        auth,
			},
		},
	}
	c, err := con.Rpc.AddWebsiteContent(con.Context(), website)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// AddWebContentDirect adds raw content bytes to a website without reading from disk.
func AddWebContentDirect(con *core.Console, websiteName string, data []byte, webPath, contentType string) error {
	_, err := AddWebContentData(con, websiteName, data, webPath, contentType, "", "", "")
	return err
}

func AddWebContentData(con *core.Console, websiteName string, data []byte, webPath, contentType, name, comment, auth string) (*clientpb.WebContent, error) {
	resolvedWebsite, listenerID, _ := resolveWebsiteTarget(con, websiteName)
	website := &clientpb.Website{
		Name:       resolvedWebsite,
		ListenerId: listenerID,
		Contents: map[string]*clientpb.WebContent{
			webPath: {
				WebsiteId:   resolvedWebsite,
				ListenerId:  listenerID,
				Name:        name,
				Path:        webPath,
				Content:     data,
				ContentType: contentType,
				Comment:     comment,
				Auth:        auth,
			},
		},
	}
	return con.Rpc.AddWebsiteContent(con.Context(), website)
}

// UpdateWebContentCmd - 更新网站内容
func UpdateWebContentCmd(cmd *cobra.Command, con *core.Console) error {
	contentId := cmd.Flags().Arg(0)
	filePath := cmd.Flags().Arg(1)
	websiteName, _ := cmd.Flags().GetString("website")
	contentType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")
	comment, _ := cmd.Flags().GetString("comment")
	updateFields := changedWebContentMetadataFields(cmd)

	var (
		updated *clientpb.WebContent
		err     error
	)
	if filePath != "" {
		updated, err = UpdateWebContent(con, contentId, filePath, websiteName, contentType)
		if err != nil {
			return err
		}
	}
	if len(updateFields) > 0 {
		updated, err = UpdateWebContentMetadataFields(con, contentId, name, comment, updateFields)
		if err != nil {
			return err
		}
	}
	if updated == nil {
		return errors.New("nothing to update: provide a file path or metadata flags")
	}
	con.Log.Importantf("Content %s updated in website %s\n", contentId, websiteName)
	return nil
}

func UpdateWebContent(con *core.Console, contentId, localFile, webPipe, typ string) (*clientpb.WebContent, error) {
	content, err := os.ReadFile(localFile)
	if err != nil {
		return nil, err
	}
	websiteName, listenerID, _ := resolveWebsiteTarget(con, webPipe)

	website := &clientpb.WebContent{
		Id:          contentId,
		WebsiteId:   websiteName,
		ListenerId:  listenerID,
		File:        localFile,
		Content:     content,
		ContentType: typ,
	}
	c, err := con.Rpc.UpdateWebsiteContent(con.Context(), website)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func UpdateWebContentMetadataCmd(cmd *cobra.Command, con *core.Console) error {
	contentID := cmd.Flags().Arg(0)
	name, _ := cmd.Flags().GetString("name")
	comment, _ := cmd.Flags().GetString("comment")

	fields := changedWebContentMetadataFields(cmd)
	if len(fields) == 0 {
		fields = []string{"name", "comment"}
	}
	_, err := UpdateWebContentMetadataFields(con, contentID, name, comment, fields)
	if err != nil {
		return err
	}
	con.Log.Importantf("Content %s metadata updated\n", contentID)
	return nil
}

func UpdateWebContentMetadata(con *core.Console, contentID, name, comment string) (*clientpb.WebContent, error) {
	fields := []string{}
	if name != "" {
		fields = append(fields, "name")
	}
	if comment != "" {
		fields = append(fields, "comment")
	}
	return UpdateWebContentMetadataFields(con, contentID, name, comment, fields)
}

func UpdateWebContentMetadataFields(con *core.Console, contentID, name, comment string, fields []string) (*clientpb.WebContent, error) {
	content := &clientpb.WebContent{
		Id:           contentID,
		Name:         name,
		Comment:      comment,
		UpdateFields: fields,
	}
	return con.Rpc.UpdateWebsiteContentMetadata(con.Context(), content)
}

func changedWebContentMetadataFields(cmd *cobra.Command) []string {
	fields := []string{}
	if cmd.Flags().Changed("name") {
		fields = append(fields, "name")
	}
	if cmd.Flags().Changed("comment") {
		fields = append(fields, "comment")
	}
	return fields
}

func AddArtifactContentCmd(cmd *cobra.Command, con *core.Console) error {
	artifactName := cmd.Flags().Arg(0)
	websiteName, _ := cmd.Flags().GetString("website")
	webPath, _ := cmd.Flags().GetString("path")
	format, _ := cmd.Flags().GetString("format")
	rdi, _ := cmd.Flags().GetString("RDI")
	contentType, _ := cmd.Flags().GetString("type")
	name, _ := cmd.Flags().GetString("name")
	comment, _ := cmd.Flags().GetString("comment")
	auth, _ := cmd.Flags().GetString("auth")

	_, err := AddArtifactContent(con, artifactName, websiteName, format, rdi, webPath, contentType, name, comment, auth)
	return err
}

func AddArtifactContent(con *core.Console, artifactName, websiteName, format, rdi, webPath, contentType, name, comment, auth string) (*clientpb.WebContent, error) {
	rpcFormat := normalizeArtifactContentFormat(format)
	if name == "" {
		name = artifactName
	}

	resolvedWebsite, listenerID, _ := resolveWebsiteTarget(con, websiteName)
	website := &clientpb.Website{
		Name:       resolvedWebsite,
		ListenerId: listenerID,
		Contents: map[string]*clientpb.WebContent{
			webPath: {
				WebsiteId:   resolvedWebsite,
				ListenerId:  listenerID,
				File:        artifactName,
				Path:        webPath,
				Type:        consts.ArtifactWebcontent,
				Url:         artifactContentOptions(rpcFormat, rdi),
				ContentType: contentType,
				Name:        name,
				Comment:     comment,
				Auth:        auth,
			},
		},
	}
	return con.Rpc.AddWebsiteContent(con.Context(), website)
}

func normalizeArtifactContentFormat(format string) string {
	if strings.EqualFold(format, "shellcode") {
		return "raw"
	}
	return format
}

func artifactContentOptions(format, rdi string) string {
	values := url.Values{}
	if format != "" {
		values.Set("format", format)
	}
	if rdi != "" {
		values.Set("rdi", rdi)
	}
	return values.Encode()
}

// RemoveWebContentCmd - 删除网站内容
func RemoveWebContentCmd(cmd *cobra.Command, con *core.Console) error {
	contentId := cmd.Flags().Arg(0)

	_, err := RemoveWebContent(con, contentId)
	if err != nil {
		return err
	}

	con.Log.Importantf("Content %s removed\n", contentId)
	return nil
}

func RemoveWebContent(con *core.Console, contentId string) (bool, error) {
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
func ListWebContentCmd(cmd *cobra.Command, con *core.Console) error {
	websiteName := cmd.Flags().Arg(0)
	resolvedWebsite, listenerID, _ := resolveWebsiteTarget(con, websiteName)

	website := &clientpb.Website{
		Name:       resolvedWebsite,
		ListenerId: listenerID,
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
		table.NewColumn("WebsiteName", "Website Name", 15),
		table.NewColumn("ListenerID", "Listener ID", 15),
		table.NewFlexColumn("Name", "Name", 1),
		table.NewFlexColumn("Path", "Path", 1),
		table.NewColumn("Size", "Size", 8),
		table.NewFlexColumn("ContentType", "Content Type", 1),
		table.NewFlexColumn("Comment", "Comment", 1),
	}, true)

	for _, content := range contents.Contents {
		row := table.NewRow(table.RowData{
			"ID":          content.Id[:8],
			"WebsiteName": content.WebsiteId,
			"ListenerID":  content.ListenerId,
			"Name":        content.Name,
			"Path":        content.Path,
			"Size":        strconv.FormatUint(content.Size, 10),
			"ContentType": content.ContentType,
			"Comment":     content.Comment,
		})
		rowEntries = append(rowEntries, row)
	}

	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func resolveWebsiteTarget(con *core.Console, key string) (string, string, bool) {
	if con != nil && con.Pipelines != nil {
		if pipeline, ok := con.Pipelines[key]; ok && pipeline != nil {
			if pipeline.GetWeb() != nil {
				return pipeline.Name, pipeline.ListenerId, true
			}
		}
	}
	if listenerID, name, ok := strings.Cut(key, ":"); ok && listenerID != "" && name != "" {
		return name, listenerID, false
	}
	return key, "", false
}
