package models

import (
	"github.com/chainreactors/malice-network/server/core"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

type Task struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	TaskID    uint32
	Type      string
	SessionId string
	done      chan bool
	Cur       int
	Total     int
}

// BeforeCreate - GORM hook
func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	t.CreatedAt = time.Now()
	return nil
}

func ConvertToTaskDB(task *core.Task) *Task {
	return &Task{
		TaskID:    task.Id,
		Type:      task.Type,
		SessionId: task.SessionId,
		Cur:       task.Cur,
		Total:     task.Total,
	}
}
