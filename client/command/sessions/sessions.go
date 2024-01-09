package sessions

import (
	"github.com/chainreactors/grumble"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/helper/helper"
	"github.com/chainreactors/malice-network/helper/styles"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
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
	if err != nil {
		width = 999
	}

	tw := table.NewWriter()
	tw.SetTitle("sessions")
	tw.SetStyle(styles.GetTableStyle(con))

	if con.Settings.SmallTermWidth < width {
		tw.AppendHeader(table.Row{
			"ID",
			"Name",
			"Transport",
			"Remote Address",
			"Hostname",
			"Username",
			"Operating System",
			"Locale",
			"Last Message",
			"Health",
		})
	} else {
		tw.AppendHeader(table.Row{
			"ID",
			"Transport",
			"Remote Address",
			"Hostname",
			"Username",
			"Operating System",
			"Health",
		})
	}

	tw.SortBy([]table.SortBy{
		{Name: "ID", Mode: table.Asc},
	})

	for _, session := range sessions {

		var SessionHealth string
		if session.IsDead {
			SessionHealth = pterm.FgRed.Sprint("[DEAD]")
		} else {
			SessionHealth = pterm.FgGreen.Sprint("[ALIVE]")
		}

		username := strings.TrimPrefix(session.Os.Username, session.Os.Hostname+"\\") // For non-AD Windows users

		var rowEntries []string
		if con.Settings.SmallTermWidth < width {
			rowEntries = []string{
				pterm.Sprint(helper.ShortSessionID(session.SessionId)),
				pterm.Sprint(session.Name),
				pterm.Sprint(session.ListenerId),
				pterm.Sprint(session.RemoteAddr),
				pterm.Sprint(session.Os.Hostname),
				pterm.Sprint(username),
				pterm.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				pterm.Sprint(time.Unix(int64(session.Timer.LastCheckin), 0).Format(time.RFC1123)),
				SessionHealth,
			}
		} else {
			rowEntries = []string{
				pterm.Sprint(helper.ShortSessionID(session.SessionId)),
				pterm.Sprint(session.ListenerId),
				pterm.Sprint(session.RemoteAddr),
				pterm.Sprint(session.Os.Hostname),
				pterm.Sprint(username),
				pterm.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				SessionHealth,
			}
		}
		// Build the row struct
		row := table.Row{}
		for _, entry := range rowEntries {
			row = append(row, entry)
		}
		tw.AppendRow(row)
		// Apply filters if any
		//if filter == "" && filterRegex == nil {
		//	tw.AppendRow(row)
		//} else {
		//	for _, rowEntry := range rowEntries {
		//		if filter != "" {
		//			if strings.Contains(rowEntry, filter) {
		//				tw.AppendRow(row)
		//				break
		//			}
		//		}
		//		if filterRegex != nil {
		//			if filterRegex.MatchString(rowEntry) {
		//				tw.AppendRow(row)
		//				break
		//			}
		//		}
		//	}
		//}
	}

	pterm.Println("%s\n", tw.Render())
}
