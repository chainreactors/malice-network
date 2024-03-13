package models

import (
	"encoding/json"
	"errors"
	"github.com/chainreactors/malice-network/server/internal/core"
	"gorm.io/gorm"
	"gorm.io/gorm/utils"
	"strconv"
	"strings"
	"time"
)

type Task struct {
	ID          string    `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	Type        string
	SessionID   string
	Session     Session `gorm:"foreignKey:SessionID"`
	Cur         int
	Total       int
	Description string
}

type TaskDescription struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Command string `json:"command"`
}

// BeforeCreate - GORM hook
func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	if err != nil {
		return err
	}
	t.CreatedAt = time.Now()
	return nil
}

func (t *Task) UpdateCur(db *gorm.DB, newCur int) error {
	return db.Model(t).Update("cur", newCur).Error
}

func ConvertToTaskDB(task *core.Task, taskType string, td *TaskDescription) *Task {
	tdString, _ := td.toJSONString()
	return &Task{
		ID:          task.SessionId + "-" + utils.ToString(task.Id),
		Type:        taskType,
		SessionID:   task.SessionId,
		Cur:         task.Cur,
		Total:       task.Total,
		Description: tdString,
	}
}

func ToCoreTask(task Task) (*core.Task, error) {
	parts := strings.Split(task.ID, "-")
	if len(parts) != 2 {
		return nil, errors.New("invalid task id")
	}
	taskID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, err
	}
	return &core.Task{
		Id:        uint32(taskID),
		Type:      task.Type,
		SessionId: task.SessionID,
		Cur:       task.Cur,
		Total:     task.Total,
	}, nil
}

func (td *TaskDescription) toJSONString() (string, error) {
	jsonString, err := json.Marshal(td)
	if err != nil {
		return "", err
	}
	return string(jsonString), nil
}
