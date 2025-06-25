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
	err = con.UpdateSessions(isAll)
	if err != nil {
		return err
	}
	if 0 < len(con.Sessions) {
		PrintSessions(con.Sessions, con, isAll)
	} else {
		con.Log.Info("No sessions\n")
	}
	return nil
}

func PrintSessions(sessions map[string]*core.Session, con *repl.Console, isAll bool) {
	//var colorIndex = 1
	var rowEntries []table.Row
	var row table.Row
	maxLengths := map[string]int{
		"ID":             8,
		"Group":          14,
		"Pipeline":       14,
		"Remote Address": 22,
		"UserName":       18,
		"System":         16,
		"Sleep":          9,
		"Last Msg":       8,
		"Health":         7,
	}
	plus_flag := false
	for _, session := range sessions {
		updateMaxLength(maxLengths, "ID", len(session.SessionId[:8]))
		updateMaxLength(maxLengths, "Group", len(fmt.Sprintf("%s/%s", session.GroupName, session.Note)))
		updateMaxLength(maxLengths, "Pipeline", len(session.PipelineId))
		//updateMaxLength(&maxLengths, "Remote Address", len(session.Target))
		updateMaxLength(maxLengths, "UserName", len(fmt.Sprintf("%s/%s", session.Os.Hostname, session.Os.Username)))
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
		var computer string
		if session.IsPrivilege {
			computer = fmt.Sprintf("%s/%s *", session.Os.Hostname, session.Os.Username)
			if !plus_flag {
				maxLengths["UserName"] += 2
				plus_flag = true
			}
		} else {
			computer = fmt.Sprintf("%s/%s", session.Os.Hostname, session.Os.Username)
		}

		row = table.NewRow(
			table.RowData{
				"ID":            session.SessionId[:8],
				"Group":         fmt.Sprintf("%s/%s", session.GroupName, session.Note),
				"Pipeline":      session.PipelineId,
				"RemoteAddress": session.Target,
				"UserName":      computer,
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
		table.NewColumn("UserName", "UserName", maxLengths["UserName"]),
		table.NewColumn("System", "System", maxLengths["System"]),
		table.NewColumn("Sleep", "Sleep", maxLengths["Sleep"]),
		table.NewColumn("Last Msg", "Last Msg", maxLengths["Last Msg"]),
		table.NewColumn("Health", "Health", maxLengths["Health"]),
	}, false)
	var err error
	tableModel.SetRows(rowEntries)
	tableModel.SetMultiline()
	tableModel.SetHandle(func() {
		SessionLogin(tableModel, tableModel.Buffer, con)()
	})
	err = tableModel.Run()
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
