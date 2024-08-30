package sessions

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/pterm/pterm"
	"strconv"
	"strings"
	"time"
)

func SessionsCmd(ctx *grumble.Context, con *console.Console) {
	err := con.UpdateSessions(true)
	if err != nil {
		console.Log.Errorf("%s", err)
		return
	}
	isAll := ctx.Flags.Bool("all")
	if 0 < len(con.Sessions) {
		PrintSessions(con, isAll)
	} else {
		console.Log.Info("No sessions")
	}
}

func PrintSessions(con *console.Console, isAll bool) {
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
	for _, session := range con.Sessions {
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

func SessionLogin(tableModel *tui.TableModel, con *console.Console) func() {
	var sessionId string
	selectRow := tableModel.GetSelectedRow()
	if selectRow == nil {
		return func() {
			console.Log.Errorf("No row selected")
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
			console.Log.Errorf(console.ErrNotFoundSession.Error())
		}
	}

	return func() {
		con.ActiveTarget.Set(session)
		con.EnableImplantCommands()
		console.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
	}
}
