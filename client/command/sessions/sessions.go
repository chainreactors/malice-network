package sessions

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"strconv"
	"strings"
	"time"
)

func SessionsCmd(cmd *cobra.Command, con *repl.Console) {
	err := con.UpdateSessions(true)
	if err != nil {
		repl.Log.Errorf("%s", err)
		return
	}
	isAll, err := cmd.Flags().GetBool("all")
	if err != nil {
		repl.Log.Errorf("%s", err)
		return
	}
	if 0 < len(con.Sessions) {
		PrintSessions(con.Sessions, con, isAll)
	} else {
		repl.Log.Info("No sessions")
	}
}

func PrintSessions(sessions map[string]*repl.Session, con *repl.Console, isAll bool) {
	//var colorIndex = 1
	var rowEntries []table.Row
	var row table.Row
	//groupColors := make(map[string]termenv.ANSIColor)
	tableModel := tui.NewTable([]table.Column{
		{Title: "ID", Width: 15},
		{Title: "Group", Width: 7},
		{Title: "Note", Width: 7},
		{Title: "Transport", Width: 10},
		{Title: "Remote Address", Width: 15},
		{Title: "Hostname", Width: 10},
		{Title: "Username", Width: 10},
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
		lastCheckinTime := time.Unix(int64(session.Timer.LastCheckin), 0)
		timeDiff := currentTime.Sub(lastCheckinTime)
		secondsDiff := uint64(timeDiff.Seconds())
		username := strings.TrimPrefix(session.Os.Username, session.Os.Hostname+"\\")
		row = table.Row{
			session.SessionId,
			session.GroupName,
			session.Note,
			"",
			session.RemoteAddr,
			session.Os.Hostname,
			username,
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
	tableModel.Title = "Sessions"
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
			repl.Log.Errorf("No row selected")
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
			repl.Log.Errorf(repl.ErrNotFoundSession.Error())
		}
	}

	return func() {
		con.SwitchImplant(session)
		repl.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
	}
}
