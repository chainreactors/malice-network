package sessions

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"time"
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
		{Title: "ID", Width: 15},
		{Title: "Group", Width: 10},
		{Title: "Pipeline", Width: 10},
		{Title: "Remote Address", Width: 15},
		{Title: "Username", Width: 15},
		{Title: "Operating System", Width: 20},
		{Title: "Last Message", Width: 15},
		{Title: "Health", Width: 15},
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
		currentTime := time.Now()
		lastCheckinTime := time.Unix(int64(session.LastCheckin), 0)
		timeDiff := currentTime.Sub(lastCheckinTime)
		secondsDiff := uint64(timeDiff.Seconds())
		row = table.Row{
			session.SessionId,
			fmt.Sprintf("[%s]%s", session.GroupName, session.Note),
			session.PipelineId,
			session.Target,
			fmt.Sprintf("%s/%s", session.Os.Hostname, session.Os.Username),
			fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
			strconv.FormatUint(secondsDiff, 10) + "s",
			SessionHealth,
		}
		rowEntries = append(rowEntries, row)
	}
	var err error
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {
		SessionLogin(tableModel, con)()
	})
	newTable := tui.NewModel(tableModel, tableModel.ConsoleHandler, true, false)
	err = newTable.Run()
	if err != nil {
		return
	}
	tui.Reset()
}

func SessionLogin(tableModel *tui.TableModel, con *repl.Console) func() {
	var sessionId string
	selectRow := tableModel.GetSelectedRow()
	if selectRow == nil {
		return func() {
			con.Log.Errorf("No row selected")
		}
	}
	for _, s := range con.Sessions {
		if strings.HasPrefix(s.SessionId, selectRow[0]) {
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
	result := tui.RenderColoredKeyValue(session, 5, 1, "Tasks")
	con.Log.Info(result)
	return nil
}
