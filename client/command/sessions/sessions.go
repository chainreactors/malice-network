package sessions

import (
	"fmt"
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/helper/styles"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/charmbracelet/bubbles/table"
	"github.com/pterm/pterm"
	"golang.org/x/term"
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
	if 0 < len(con.Sessions) {
		PrintSessions(con.Sessions, con)
	} else {
		console.Log.Info("No sessions")
	}
}

func PrintSessions(sessions map[string]*clientpb.Session, con *console.Console) {
	width, _, err := term.GetSize(0)
	var tableModel styles.TableModel
	var rowEntries []table.Row
	var row table.Row
	if err != nil {
		width = 999
	}
	if con.Settings.SmallTermWidth < width {
		tableModel = styles.TableModel{Columns: []table.Column{
			{Title: "ID", Width: 4},
			{Title: "Name", Width: 4},
			{Title: "Transport", Width: 10},
			{Title: "Remote Address", Width: 15},
			{Title: "Hostname", Width: 10},
			{Title: "Username", Width: 10},
			{Title: "Operating System", Width: 20},
			{Title: "Locale", Width: 10},
			{Title: "Last Message", Width: 15},
			{Title: "Health", Width: 10},
		}}
	} else {
		tableModel = styles.TableModel{Columns: []table.Column{
			{Title: "ID", Width: 4},
			{Title: "Transport", Width: 10},
			{Title: "Remote Address", Width: 15},
			{Title: "Hostname", Width: 10},
			{Title: "Username", Width: 10},
			{Title: "Operating System", Width: 20},
			{Title: "Health", Width: 10},
		}}
	}
	for _, session := range sessions {

		var SessionHealth string
		if session.IsDead {
			SessionHealth = pterm.FgRed.Sprint("[DEAD]")
		} else {
			SessionHealth = pterm.FgGreen.Sprint("[ALIVE]")
		}

		username := strings.TrimPrefix(session.Os.Username, session.Os.Hostname+"\\") // For non-AD Windows users
		if con.Settings.SmallTermWidth < width {
			row = table.Row{
				helper.ShortSessionID(session.SessionId),
				session.Name,
				"",
				session.ListenerId,
				session.RemoteAddr,
				session.Os.Hostname,
				username,
				fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				time.Unix(int64(session.Timer.LastCheckin), 0).Format(time.RFC1123),
				SessionHealth,
			}
		} else {
			row = table.Row{
				helper.ShortSessionID(session.SessionId),
				"",
				session.ListenerId,
				session.RemoteAddr,
				session.Os.Hostname,
				username,
				fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				SessionHealth,
			}
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.Rows = rowEntries
	err = tableModel.Run()
	if err != nil {
		console.Log.Errorf("Can't print sessions: %s", err)
	}
}
