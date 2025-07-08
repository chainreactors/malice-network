package sessions

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"io"
	"sort"
	"strings"
	"time"
)

// formatTimeDiff formats time difference in seconds to human readable format
func formatTimeDiff(timestamp int64, isAlive bool) string {
	now := time.Now().Unix()
	diff := now - timestamp

	var timeStr string
	if diff > 0 {
		duration := time.Duration(diff) * time.Second
		timeStr = duration.String()
	} else {
		timeStr = "now"
	}

	if isAlive {
		return tui.GreenFg.Render(timeStr)
	} else {
		return tui.RedFg.Render(timeStr)
	}
}

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
		"Group/Note":     10,
		"Pipeline":       14,
		"Remote Address": 18,
		"UserName":       20,
		"System":         16,
		"Sleep":          9,
		"Last":           6,
		"CreatedAt":      16,
	}

	// Convert map to slice for sorting
	var sessionList []*core.Session
	for _, session := range sessions {
		sessionList = append(sessionList, session)
	}

	// Sort by CreatedAt timestamp (descending - newest first)
	sort.Slice(sessionList, func(i, j int) bool {
		return sessionList[i].CreatedAt < sessionList[j].CreatedAt
	})

	for _, session := range sessionList {
		if !session.IsAlive && !isAll {
			continue
		}
		var computer string
		if session.IsPrivilege {
			computer = fmt.Sprintf("%s/%s *", session.Os.Hostname, session.Os.Username)
		} else {
			computer = fmt.Sprintf("%s/%s", session.Os.Hostname, session.Os.Username)
		}

		row = table.NewRow(
			table.RowData{
				"ID":             session.SessionId[:8],
				"Group/Note":     fmt.Sprintf("%s/%s", session.GroupName, session.Note),
				"Pipeline":       session.PipelineId,
				"Remote Address": session.Target,
				"UserName":       computer,
				"System":         fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				"Sleep":          fmt.Sprintf("%d/%.1f%%", session.Timer.Interval, session.Timer.Jitter*100),
				"Last":           formatTimeDiff(session.LastCheckin, session.IsAlive),
				"CreatedAt":      time.Unix(session.CreatedAt, 0).Format("2006-01-02 15:04"),
			})
		rowEntries = append(rowEntries, row)
	}

	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", maxLengths["ID"]),
		table.NewColumn("Group/Note", "Group/Note", maxLengths["Group/Note"]),
		table.NewColumn("Pipeline", "Pipeline", maxLengths["Pipeline"]),
		table.NewColumn("Remote Address", "Remote Address", maxLengths["Remote Address"]),
		table.NewColumn("UserName", "UserName", maxLengths["UserName"]),
		table.NewColumn("System", "System", maxLengths["System"]),
		table.NewColumn("Sleep", "Sleep", maxLengths["Sleep"]),
		table.NewColumn("Last", "Last", maxLengths["Last"]),
		table.NewColumn("CreatedAt", "CreatedAt", maxLengths["CreatedAt"]),
	}, false)
	tableModel.SetRows(rowEntries)
	tableModel.SetMultiline()
	tableModel.SetHandle(func() {
		SessionLogin(tableModel, tableModel.Buffer, con)()
	})
	err := tableModel.Run()
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
