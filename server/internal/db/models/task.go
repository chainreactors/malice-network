package models

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/server/internal/core"
	"gorm.io/gorm"
	"strconv"
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
		ID:          task.SessionId + "-" + uint32ToString(task.Id),
		Type:        taskType,
		SessionID:   task.SessionId,
		Cur:         task.Cur,
		Total:       task.Total,
		Description: tdString,
	}
}

func (td *TaskDescription) toJSONString() (string, error) {
	jsonString, err := json.Marshal(td)
	if err != nil {
		return "", err
	}
	return string(jsonString), nil
}

func uint32ToString(num uint32) string {
	return strconv.FormatUint(uint64(num), 10) // 10 表示十进制
}

func GetTaskDescriptionByID(db *gorm.DB, taskID string) (*TaskDescription, error) {
	var task Task
	if err := db.Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}

	var td TaskDescription
	if err := json.Unmarshal([]byte(task.Description), &td); err != nil {
		return nil, err
	}

	return &td, nil
}

func FindTasksWithNonOneCurTotal(dbSession *gorm.DB, session Session) ([]Task, error) {
	var tasks []Task
	result := dbSession.Where("session_id = ?", session.SessionID).Where("cur != total").Find(&tasks)
	if result.Error != nil {
		return tasks, result.Error
	}
	if len(tasks) == 0 {
		return tasks, gorm.ErrRecordNotFound
	}
	return tasks, nil
}
