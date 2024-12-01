package sessions

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"io"
	"strconv"
	"strings"
)

func SessionsCmd(cmd *cobra.Command, con *repl.Console) error {
	err := con.UpdateSessions(true)
	if err != nil {
		return err
	}
	isAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	if 0 < len(con.Sessions) {
		PrintSessions(con.Sessions, con, isAll)
	} else {
		con.Log.Info("No sessions")
	}
	return nil
}

func PrintSessions(sessions map[string]*core.Session, con *repl.Console, isAll bool) {
	//var colorIndex = 1
	var rowEntries []table.Row
	var row table.Row
	//groupColors := make(map[string]termenv.ANSIColor)
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 8),
		table.NewColumn("Group", "Group", 20),
		table.NewColumn("Pipeline", "Pipeline", 15),
		table.NewColumn("Remote Address", "Remote Address", 16),
		table.NewColumn("Username", "Username", 15),
		table.NewColumn("System", "System", 20),
		table.NewColumn("Last Message", "Last Message", 15),
		table.NewColumn("Health", "Health", 15),
	}, false)
	for _, session := range sessions {
		var SessionHealth string
		if !session.IsAlive {
			if !isAll {
				continue
			}
			SessionHealth = pterm.FgRed.Sprint("[DEAD]")
		} else {
			SessionHealth = pterm.FgGreen.Sprint("[ALIVE]")
		}
		row = table.NewRow(
			table.RowData{
				"ID":             session.SessionId[:8],
				"Group":          fmt.Sprintf("[%s]%s", session.GroupName, session.Note),
				"Pipeline":       session.PipelineId,
				"Remote Address": session.Target,
				"Username":       fmt.Sprintf("%s/%s", session.Os.Hostname, session.Os.Username),
				"System":         fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				"Last Message":   strconv.FormatUint(uint64(session.LastCheckin), 10) + "s",
				"Health":         SessionHealth,
			})
		rowEntries = append(rowEntries, row)
	}
	newTable := tui.NewModel(tableModel, nil, false, false)
	var err error
	tableModel.SetRows(rowEntries)
	tableModel.SetMultiline()
	tableModel.SetHandle(func() {
		SessionLogin(tableModel, newTable.Buffer, con)()
	})
	err = newTable.Run()
	if err != nil {
		return
	}
	tui.Reset()
	if con.ActiveTarget.Session != nil {
		con.Session.GetHistory()
	}
}

func SessionLogin(tableModel *tui.TableModel, writer io.Writer, con *repl.Console) func() {
	var sessionId string
	selectRow := tableModel.GetHighlightedRow()
	if selectRow.Data == nil {
		return func() {
			con.Log.FErrorf(writer, "No row selected\n")
			return
		}
	}
	for _, s := range con.Sessions {
		if strings.HasPrefix(s.SessionId, selectRow.Data["ID"].(string)) {
			sessionId = s.SessionId
		}
	}
	session := con.Sessions[sessionId]

	if session == nil {
		return func() {
			con.Log.Errorf(repl.ErrNotFoundSession.Error())
		}
	}

	return func() {
		Use(con, session)
	}
}

func SessionInfoCmd(cmd *cobra.Command, con *repl.Console) error {
	session := con.GetInteractive()
	if session == nil {
		return repl.ErrNotFoundSession
	}
	result := tui.RendStructDefault(session, "Tasks")
	con.Log.Info(result)
	return nil
}
