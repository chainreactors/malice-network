package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/spf13/cobra"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AuditExport 用于导出 JSON
// 保证字段顺序和命名符合需求
// 注意 taskResult 保留
type AuditExport struct {
	SessionID  string      `json:"session"`
	TaskID     string      `json:"task"`
	Command    string      `json:"command"`
	Total      int32       `json:"total"`
	Cur        int32       `json:"cur"`
	Response   interface{} `json:"response"`
	Request    interface{} `json:"request"`
	Created    string      `json:"created"`
	Finished   string      `json:"finished"`
	Lasted     string      `json:"lasted"`
	TaskResult string      `json:"taskResult"`
}

func AuditSessionCmd(cmd *cobra.Command, con *repl.Console) error {
	sessionID := cmd.Flags().Arg(0)
	output, _ := cmd.Flags().GetString("output")
	path, _ := cmd.Flags().GetString("file")
	ext := strings.ToLower(output)
	var isJson bool
	var format string
	switch ext {
	case "json":
		isJson = true
		format = ".json"
	case "html", "htm":
		isJson = false
		format = ".html"
	default:
		return fmt.Errorf("unsupported export format: %s", ext)
	}
	auditLog, err := con.Rpc.GetAudit(con.Context(), &clientpb.SessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return err
	}
	if path == "" {
		path = filepath.Join(assets.GetTempDir(), sessionID+format)
	}

	if isJson {
		// 组装导出结构体
		var exportList []AuditExport
		for _, a := range auditLog.Audit {
			exportList = append(exportList, AuditExport{
				SessionID: a.Context.Session.SessionId,
				TaskID:    strconv.Itoa(int(a.Context.Task.TaskId)),
				Total:     a.Context.Task.Total,
				Cur:       a.Context.Task.Cur,
				Command:   a.Command,
				Response:  a.Context.Spite,
				Request:   a.Request,
				Created:   a.Created,
				Finished:  a.Finished,
				Lasted:    a.Lasted,
			})
		}
		data, err := json.MarshalIndent(exportList, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(path, data, 0644)
		if err != nil {
			return err
		}
		con.Log.Infof("%s audit log saved at %s\n", sessionID, path)
		return nil
	}

	// HTML 渲染
	data, err := renderAuditHTML(auditLog.Audit)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	con.Log.Infof("%s audit log saved at %s\n", sessionID, path)
	return nil
}

// renderAuditHTML
func renderAuditHTML(entries []*clientpb.Audit) ([]byte, error) {
	type AuditView struct {
		*clientpb.Audit
		RequestOmitted bool
		TaskResult     string
	}
	var auditsView []AuditView
	for _, a := range entries {
		reqBytes, _ := json.Marshal(a.Request)
		audit := AuditView{
			Audit:          a,
			RequestOmitted: len(reqBytes) > 100*1024,
		}
		fn, ok := intermediate.InternalFunctions[a.Context.Task.Type]
		if ok && fn.FinishCallback != nil {
			resp, err := fn.FinishCallback(a.Context)
			if err != nil {
				logs.Log.Errorf("failed to parse task: %s", err)
				audit.TaskResult = fmt.Sprintf("Error parsing task: %s", err.Error())
			} else {
				audit.TaskResult = resp
			}
		} else {
			audit.TaskResult = "No task result available"
		}
		auditsView = append(auditsView, audit)
	}

	funcMap := template.FuncMap{
		"formatjson": func(v interface{}) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		"len": func(v interface{}) int {
			switch val := v.(type) {
			case []AuditView:
				return len(val)
			default:
				return 0
			}
		},
		"js": func(s string) string {
			return template.JSEscapeString(s)
		},
	}

	data := struct {
		Entries       []AuditView
		GeneratedTime string
	}{
		Entries:       auditsView,
		GeneratedTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	var buf bytes.Buffer
	t := template.Must(template.New("audit").Funcs(funcMap).Parse(string(assets.AuditHtml)))
	err := t.Execute(&buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
