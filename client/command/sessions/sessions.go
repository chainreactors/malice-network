package sessions

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/client/tui"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/charmbracelet/bubbles/table"
	"github.com/muesli/termenv"
	"github.com/pterm/pterm"
	"regexp"
	"strconv"
	"strings"
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
	var colorIndex = 1
	var rowEntries []table.Row
	var row table.Row
	groupColors := make(map[string]termenv.ANSIColor)
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
	})
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
		if _, exists := groupColors[session.GroupName]; !exists {
			groupColors[session.GroupName] = termenv.ANSIColor(colorIndex)
			colorIndex++
		}
		currentTime := time.Now()
		lastCheckinTime := time.Unix(int64(session.Timer.LastCheckin), 0)
		timeDiff := currentTime.Sub(lastCheckinTime)
		secondsDiff := uint64(timeDiff.Seconds())
		username := strings.TrimPrefix(session.Os.Username, session.Os.Hostname+"\\")
		row = table.Row{
			termenv.String(helper.ShortSessionID(session.SessionId)).Foreground(groupColors[session.GroupName]).String(),
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
	tableModel.Rows = rowEntries
	tableModel.SetRows()
	tableModel.SetHandle(func() {
		SessionLogin(tableModel, con)()
	})
	err := tui.Run(tableModel)
	if err != nil {
		return
	}
}

func SessionLogin(tableModel *tui.TableModel, con *console.Console) func() {
	var sessionId string
	con.UpdateSession()
	selectRow := tableModel.GetSelectedRow()
	re := regexp.MustCompile(`\x1b\[[0-9;]*m(.*?)\x1b\[0m`)
	matches := re.FindStringSubmatch(selectRow[0])

	if len(matches) < 1 {
		console.Log.Errorf("No match found")
		return nil
	}
	for _, s := range con.Sessions {
		if strings.HasPrefix(s.SessionId, matches[1]) {
			sessionId = s.SessionId
		}
	}
	session := con.Sessions[sessionId]

	if session == nil {
		console.Log.Errorf(console.ErrNotFoundSession.Error())
		return nil
	}

	return func() {
		con.ActiveTarget.Set(session)
		console.Log.Infof("Active session %s (%s)\n", session.Note, session.SessionId)
	}
}
