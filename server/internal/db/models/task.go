package models

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"gorm.io/gorm"
	"time"
)

type Task struct {
	ID          string    `gorm:"primaryKey;->;<-:create;"`
	Created     time.Time `gorm:"->;<-:create;"`
	Deadline    time.Time
	CallBy      string
	Seq         uint32
	Type        string
	SessionID   string
	Session     Session `gorm:"foreignKey:SessionID"`
	Cur         int
	Total       int
	Description string
	ClientName  string
	FinishTime  time.Time
	LastTime    time.Time
}

func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	if err != nil {
		return err
	}
	t.Created = time.Now()
	return nil
}

func (t *Task) UpdateCur(db *gorm.DB, newCur int) error {
	t.LastTime = time.Now()
	return db.Model(t).Updates(map[string]interface{}{
		"cur":       newCur,
		"last_time": t.LastTime,
	}).Error
}

func (t *Task) UpdateTotal(db *gorm.DB, newTotal int) error {
	return db.Model(t).Update("total", newTotal).Error
}

func (t *Task) UpdateFinish(db *gorm.DB) error {
	t.FinishTime = time.Now()
	return db.Save(t).Error
}

func (t *Task) ToProtobuf() *clientpb.Task {
	if t == nil {
		return nil
	}
	return &clientpb.Task{
		TaskId:      uint32(t.Seq),
		Type:        t.Type,
		SessionId:   t.SessionID,
		Cur:         int32(t.Cur),
		Total:       int32(t.Total),
		Description: t.Description,
		Callby:      t.ClientName,
		Timeout:     time.Now().After(t.Deadline),
		Finished:    t.Cur == t.Total,
	}
}
