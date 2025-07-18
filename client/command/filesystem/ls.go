package filesystem

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/handler"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"os"
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
	session.Console(cmd, task, path)
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
			if len(resp.GetFiles()) == 0 {
				con.Log.Infof("No files")
				return "", nil
			}
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
				table.NewColumn("Name", "Name", 25),
				table.NewColumn("Size", "Size", 10),
				table.NewColumn("Mode", "Mode", 10),
				table.NewColumn("Time", "Time", 16),
				table.NewColumn("Link", "Link", 15),
				//{Title: "name", Width: 25},
				//{Title: "size", Width: 10},
				//{Title: "mod", Width: 16},
				//{Title: "link", Width: 15},
			}, true)
			for _, f := range resp.GetFiles() {
				var size string
				if f.IsDir {
					size = tui.GreenFg.Render("dir")
				} else {
					size = fileutils.Bytes(f.Size)
				}
				row = table.NewRow(
					table.RowData{
						"Name": f.Name,
						"Size": size,
						"Mode": os.FileMode(f.Mode).String(),
						"Time": time.Unix(f.ModTime, 0).Format("2006-01-02 15:04"),
						"Link": f.Link,
					})
				//	table.Row{
				//	f.Name,
				//	size,
				//	time.Unix(f.ModTime, 0).Format("2006-01-02 15:04"),
				//	f.Link,
				//}
				rowEntries = append(rowEntries, row)
			}
			tableModel.SetMultiline()
			tableModel.SetRows(rowEntries)
			return tableModel.View(), nil
		})

	con.AddCommandFuncHelper(
		consts.ModuleLs,
		consts.ModuleLs,
		consts.ModuleLs+`(active(),"/tmp")`,
		[]string{
			"session: special session",
			"path: path to list files",
		},
		[]string{"task"})

	con.AddCommandFuncHelper(
		"bls",
		"bls",
		`bls(active(),"/tmp")`,
		[]string{
			"session: special session",
			"path: path to list files",
		},
		[]string{"task"})
}
