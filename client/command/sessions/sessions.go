package sessions

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"github.com/pterm/pterm"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Command(con *console.Console) []*grumble.Command {
	return []*grumble.Command{
		&grumble.Command{
			Name: "sessions",
			Help: "List sessions",
			Flags: func(f *grumble.Flags) {
				f.String("i", "interact", "", "interact with a session")
				f.String("k", "kill", "", "kill the designated session")
				f.Bool("K", "kill-all", false, "kill all the sessions")
				f.Bool("C", "clean", false, "clean out any sessions marked as [DEAD]")
				f.Bool("F", "force", false, "force session action without waiting for results")
				f.Bool("a", "all", false, "show all sessions")
				//f.String("f", "filter", "", "filter sessions by substring")
				//f.String("e", "filter-re", "", "filter sessions by regular expression")

				f.Int("t", "timeout", assets.DefaultSettings.DefaultTimeout, "command timeout in seconds")
			},
			Run: func(ctx *grumble.Context) error {
				SessionsCmd(ctx, con)
				return nil
			},
		},
	}
}

func SessionsCmd(ctx *grumble.Context, con *console.Console) {
	con.UpdateSession()
	isAll := ctx.Flags.Bool("all")
	if 0 < len(con.Sessions) {
		PrintSessions(con.Sessions, con, isAll)
	} else {
		console.Log.Info("No sessions")
	}
}

func PrintSessions(sessions map[string]*clientpb.Session, con *console.Console, isAll bool) {
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
		if !session.IsDead {
			if !isAll {
				continue
			}
			SessionHealth = pterm.FgRed.Sprint("[DEAD]")
		} else {
			SessionHealth = pterm.FgGreen.Sprint("[ALIVE]")
		}
		//if _, exists := groupColors[session.GroupName]; !exists {
		//	groupColors[session.GroupName] = termenv.ANSIColor(colorIndex)
		//	colorIndex++
		//}
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
	var wg sync.WaitGroup
	var err error
	tableModel.SetRows(rowEntries)
	tableModel.SetHandle(func() {
		SessionLogin(tableModel, con)()
	})
	tableModel.Title = "Sessions"
	newTable := tui.NewModel(tableModel, tableModel.ConsoleHandler, true, false)
	wg.Add(1)
	go func() {
		defer wg.Done()
		err = newTable.Run()
	}()
	wg.Wait()
	if err != nil {
		return
	}
}

func SessionLogin(tableModel *tui.TableModel, con *console.Console) func() {
	var sessionId string
	con.UpdateSession()
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
		console.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
	}
}
