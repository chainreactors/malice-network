package sessions

import (
	"fmt"
	"net"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

// formatTimeDiff formats time difference in seconds to human readable format
// Uses compact format: <60s: "45s", <60m: "5m30s", <24h: "2h30m", >=24h: "15d6h"
func formatTimeDiff(timestamp int64, isAlive bool) string {
	now := time.Now().Unix()
	diff := now - timestamp

	var timeStr string
	if diff <= 0 {
		timeStr = "now"
	} else {
		seconds := diff % 60
		minutes := (diff / 60) % 60
		hours := (diff / 3600) % 24
		days := diff / 86400

		switch {
		case diff < 60:
			timeStr = fmt.Sprintf("%ds", seconds)
		case diff < 3600:
			timeStr = fmt.Sprintf("%dm%ds", minutes, seconds)
		case diff < 86400:
			timeStr = fmt.Sprintf("%dh%dm", hours, minutes)
		default:
			timeStr = fmt.Sprintf("%dd%dh", days, hours)
		}
	}

	if isAlive {
		return tui.GreenFg.Render(timeStr)
	} else {
		return tui.RedFg.Render(timeStr)
	}
}

func SessionsCmd(cmd *cobra.Command, con *core.Console) error {
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

func PrintSessions(sessions map[string]*client.Session, con *core.Console, isAll bool) {
	//var colorIndex = 1
	var rowEntries []table.Row
	var row table.Row
	// Convert map to slice for sorting
	var sessionList []*client.Session
	for _, session := range sessions {
		sessionList = append(sessionList, session)
	}

	// Sort by CreatedAt timestamp (descending - newest first)
	sort.Slice(sessionList, func(i, j int) bool {
		return sessionList[i].CreatedAt > sessionList[j].CreatedAt
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

		// Strip port from Remote Address, keep IP only
		remoteAddr := session.Target
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			remoteAddr = host
		}

		// Extract PID and process name
		var pid string
		var processName string
		if session.Process != nil {
			pid = fmt.Sprintf("%d", session.Process.Pid)
			processName = filepath.Base(session.Process.Name)
			if processName == "." || processName == "" {
				processName = filepath.Base(session.Process.Path)
			}
		}

		row = table.NewRow(
			table.RowData{
				"ID":             shortSessionID(session.SessionId),
				"Group/Note":     fmt.Sprintf("%s/%s", session.GroupName, session.Note),
				"Pipeline":       session.PipelineId,
				"Remote Address": remoteAddr,
				"UserName":       computer,
				"System":         fmt.Sprintf("%s/%s", session.Os.Name, session.Os.Arch),
				"PID":            pid,
				"Process":        processName,
				"Sleep":          fmt.Sprintf("%s [%.1f%%]", session.Timer.Expression, session.Timer.Jitter*100),
				"Last":           formatTimeDiff(session.LastCheckin, session.IsAlive),
				"LastRaw":        fmt.Sprintf("%020d", session.LastCheckin),
				"CreatedAt":      time.Unix(session.CreatedAt, 0).Format("2006-01-02 15:04"),
			})
		rowEntries = append(rowEntries, row)
	}

	// Use tui.NewTable's isStatic parameter to control static mode
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 10),
		table.NewFlexColumn("Group/Note", "Group/Note", 1),
		table.NewFlexColumn("Pipeline", "Pipeline", 1),
		table.NewFlexColumn("Remote Address", "Remote Address", 1),
		table.NewFlexColumn("UserName", "User Name", 1),
		table.NewFlexColumn("System", "System", 1),
		table.NewFlexColumn("Process", "Process", 1),
		table.NewColumn("PID", "PID", 7),
		table.NewColumn("Sleep", "Sleep", 12),
		table.NewColumn("Last", "Last", 8),
		table.NewColumn("LastRaw", "", 0),
		table.NewColumn("CreatedAt", "Created At", 16),
	}, common.ShouldUseStaticOutput(con))
	tableModel.SetAscSort("LastRaw")
	tableModel.SetRows(rowEntries)
	tableModel.SetMultiline()
	tableModel.SetHandle(func() {
		SessionLogin(tableModel, con)()
	})
	rendered, err := common.RunTable(con, tableModel)
	if err != nil {
		return
	}
	if rendered {
		return
	}
	tui.Reset()
}

func SessionLogin(tableModel *tui.TableModel, con *core.Console) func() {
	var sessionId string
	selectRow := tableModel.GetHighlightedRow()
	if selectRow.Data == nil {
		return func() {
			con.Log.Errorf("No row selected\n")
			return
		}
	}
	prefix := selectRow.Data["ID"].(string)
	var matches []string
	for id := range con.Sessions {
		if strings.HasPrefix(id, prefix) {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 0:
		return func() {
			con.Log.Errorf("%s\n", core.ErrNotFoundSession.Error())
		}
	case 1:
		sessionId = matches[0]
	default:
		return func() {
			con.Log.Errorf("ambiguous session prefix '%s'\n", prefix)
		}
	}
	session := con.Sessions[sessionId]

	if session == nil {
		return func() {
			con.Log.Errorf("%s", core.ErrNotFoundSession.Error())
		}
	}

	return func() {
		Use(con, session)
	}
}
