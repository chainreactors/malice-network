package filesystem

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/file"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"time"
)

func LsCmd(cmd *cobra.Command, con *repl.Console) error {
	path := cmd.Flags().Arg(0)
	if path == "" {
		path = "./"
	}
	session := con.GetInteractive()
	task, err := Ls(con.Rpc, session, path)
	if err != nil {
		return err
	}
	session.Console(task, path)
	return nil
}

func Ls(rpc clientrpc.MaliceRPCClient, session *core.Session, path string) (*clientpb.Task, error) {
	task, err := rpc.Ls(session.Context(), &implantpb.Request{
		Name:  consts.ModuleLs,
		Input: path,
	})
	if err != nil {
		return nil, err
	}
	return task, err
}

func RegisterLsFunc(con *repl.Console) {
	con.RegisterImplantFunc(
		consts.ModuleLs,
		Ls,
		"bls",
		Ls,
		func(ctx *clientpb.TaskContext) (interface{}, error) {
			err := handler.HandleMaleficError(ctx.Spite)
			if err != nil {
				return "", err
			}
			resp := ctx.Spite.GetLsResponse()
			var fileDetails []string
			for _, file := range resp.GetFiles() {
				fileStr := fmt.Sprintf("%s|%s|%s|%s|%s",
					file.Name,
					strconv.FormatBool(file.IsDir),
					strconv.FormatUint(file.Size, 10),
					strconv.FormatInt(file.ModTime, 10),
					file.Link,
				)
				fileDetails = append(fileDetails, fileStr)
			}
			return strings.Join(fileDetails, ","), nil
		},
		func(content *clientpb.TaskContext) (string, error) {
			msg := content.Spite
			resp := msg.GetLsResponse()
			var rowEntries []table.Row
			var row table.Row
			tableModel := tui.NewTable([]table.Column{
				{Title: "name", Width: 25},
				{Title: "size", Width: 10},
				{Title: "mod", Width: 16},
				{Title: "link", Width: 15},
			}, true)
			for _, f := range resp.GetFiles() {
				var size string
				if f.IsDir {
					size = "dir"
				} else {
					size = file.Bytes(f.Size)
				}
				row = table.Row{
					f.Name,
					size,
					time.Unix(f.ModTime, 0).Format("2006-01-02 15:04"),
					f.Link,
				}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.AddInternalFuncHelper(
		consts.ModuleLs,
		consts.ModuleLs,
		consts.ModuleLs+"(active(),\"/tmp\")",
		[]string{
			"session: special session",
			"path: path to list files",
		},
		[]string{"task"})
}
