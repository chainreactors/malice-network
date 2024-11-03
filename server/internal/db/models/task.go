package models

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"gorm.io/gorm"
	"time"
)

type Task struct {
	ID          string    `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	Seq         int
	Type        string
	SessionID   string
	Session     Session `gorm:"foreignKey:SessionID"`
	Cur         int
	Total       int
	Description string
	ClientName  string
}

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

func (t *Task) UpdateTotal(db *gorm.DB, newTotal int) error {
	return db.Model(t).Update("total", newTotal).Error
}

func (t *Task) ToProtobuf() *clientpb.Task {
	return &clientpb.Task{
		TaskId:      uint32(t.Seq),
		Type:        t.Type,
		SessionId:   t.SessionID,
		Cur:         int32(t.Cur),
		Total:       int32(t.Total),
		Description: t.Description,
		ClientName:  t.ClientName,
	}
}
