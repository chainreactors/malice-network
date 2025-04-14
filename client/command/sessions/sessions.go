package sessions

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"io"
	"strconv"
	"strings"
)

func SessionsCmd(cmd *cobra.Command, con *repl.Console) error {
	isAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		return err
	}
	isStatic, err := cmd.Flags().GetBool("static")
	if err != nil {
		return err
	}
	err = con.UpdateSessions(isAll)
	if err != nil {
		return err
	}
	if 0 < len(con.Sessions) {
		PrintSessions(con.Sessions, con, isAll, isStatic)
	} else {
		con.Log.Info("No sessions\n")
	}
	return nil
}

func PrintSessions(sessions map[string]*core.Session, con *repl.Console, isAll, isStastic bool) {
	//var colorIndex = 1
	var rowEntries []table.Row
	var row table.Row
	maxLengths := map[string]int{
		"ID":             8,
		"Group":          14,
		"Pipeline":       16,
		"Remote Address": 22,
		"Username":       16,
		"System":         16,
		"Sleep":          9,
		"Last Msg":       8,
		"Health":         7,
	}
	for _, session := range sessions {
		updateMaxLength(maxLengths, "ID", len(session.SessionId[:8]))
		updateMaxLength(maxLengths, "Group", len(fmt.Sprintf("%s/%s", session.GroupName, session.Note)))
		updateMaxLength(maxLengths, "Pipeline", len(session.PipelineId))
		//updateMaxLength(&maxLengths, "Remote Address", len(session.Target))
		updateMaxLength(maxLengths, "Username", len(fmt.Sprintf("%s/%s", session.Os.Hostname, session.Os.Username)))
		updateMaxLength(maxLengths, "System", len(fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch)))
		//updateMaxLength(&maxLengths, "Sleep", len(fmt.Sprintf("%d %.2f", session.Timer.Interval, session.Timer.Jitter)))
		//updateMaxLength(&maxLengths, "Last Message", len(strconv.FormatUint(uint64(session.Timediff), 10)+"s"))
		//updateMaxLength(&maxLengths, "Health", len(pterm.FgGreen.Sprint("[ALIVE]"))) // Assuming ALIVE is longer than DEAD
		var SessionHealth string
		if !session.IsAlive {
			if !isAll {
				continue
			}
			SessionHealth = tui.RedFg.Render("DEAD")
		} else {
			SessionHealth = tui.GreenFg.Render("ALIVE")
		}
		row = table.NewRow(
			table.RowData{
				"ID":            session.SessionId[:8],
				"Group":         fmt.Sprintf("%s/%s", session.GroupName, session.Note),
				"Pipeline":      session.PipelineId,
				"RemoteAddress": session.Target,
				"Username":      fmt.Sprintf("%s/%s", session.Os.Hostname, session.Os.Username),
				"System":        fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				"Sleep":         fmt.Sprintf("%d  %.2f", session.Timer.Interval, session.Timer.Jitter),
				"Last Msg":      strconv.FormatUint(uint64(session.Timediff), 10) + "s",
				"Health":        SessionHealth,
			})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", maxLengths["ID"]),
		table.NewColumn("Group", "Group", maxLengths["Group"]),
		table.NewColumn("Pipeline", "Pipeline", maxLengths["Pipeline"]),
		table.NewColumn("RemoteAddress", "RemoteAddress", maxLengths["Remote Address"]),
		table.NewColumn("Username", "Username", maxLengths["Username"]),
		table.NewColumn("System", "System", maxLengths["System"]),
		table.NewColumn("Sleep", "Sleep", maxLengths["Sleep"]),
		table.NewColumn("Last Msg", "Last Msg", maxLengths["Last Msg"]),
		table.NewColumn("Health", "Health", maxLengths["Health"]),
	}, false)

	newTable := tui.NewModel(tableModel, nil, false, false)
	var err error
	tableModel.SetRows(rowEntries)
	if isStastic {
		con.Log.Infof(newTable.View())
		return
	}
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

func updateMaxLength(maxLengths map[string]int, key string, newLength int) {
	if (maxLengths)[key] < newLength {
		(maxLengths)[key] = newLength
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
