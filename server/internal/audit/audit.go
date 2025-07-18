package audit

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/gookit/config/v2"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"regexp"
)

func AuditTaskLog(sessionID string) (*clientpb.Audits, error) {
	taskDir := filepath.Join(configs.ContextPath, sessionID, consts.TaskPath)
	requestDir := filepath.Join(configs.ContextPath, sessionID, consts.RequestPath)
	re := regexp.MustCompile(`^([0-9]+)_([0-9]+)$`)
	files, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, err
	}

	audits := &clientpb.Audits{}
	session, err := db.FindSession(sessionID)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matches := re.FindStringSubmatch(file.Name())
		if matches == nil {
			continue
		}
		taskID := matches[1]
		taskKey := sessionID + "-" + taskID
		task, err := db.GetTask(taskKey)
		if err != nil || task == nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(taskDir, file.Name()))
		if err != nil {
			logs.Log.Errorf("Error reading file: %s", err)
			continue
		}
		spite := &implantpb.Spite{}
		err = proto.Unmarshal(content, spite)
		if err != nil {
			logs.Log.Errorf("Error unmarshalling protobuf: %s", err)
			continue
		}
		audit := &clientpb.Audit{
			Context: &clientpb.TaskContext{
				Task:    task.ToProtobuf(),
				Session: session.ToProtobuf(),
				Spite:   spite,
			},
			Command:  task.Description,
			Created:  task.Created.Format("2006-01-02 15:04:05"),
			Finished: task.FinishTime.Format("2006-01-02 15:04:05"),
			Lasted:   task.LastTime.Format("2006-01-02 15:04:05"),
		}
		if auditLevel := config.Int(consts.ConfigAuditLevel); auditLevel > 1 {
			requestData, err := os.ReadFile(filepath.Join(requestDir, taskID))
			if err != nil {
				logs.Log.Errorf("Error reading request file: %s", err)
			}
			request := &implantpb.Spite{}
			err = proto.Unmarshal(requestData, request)
			if err != nil {
				logs.Log.Errorf("Error unmarshalling protobuf: %s", err)
				continue
			}
			audit.Request = request
		}

		audits.Audit = append(audits.Audit, audit)
	}
	return audits, nil
}
