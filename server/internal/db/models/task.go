package models

import (
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"gorm.io/gorm"
)

type Task struct {
	ID             string    `gorm:"primaryKey;->;<-:create;"`
	Created        time.Time `gorm:"->;<-:create;"`
	Deadline       time.Time
	CallBy         string
	Seq            uint32
	Type           string
	SessionID      string
	Session        Session `gorm:"foreignKey:SessionID;references:SessionID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Cur            int
	Total          int
	Description    string
	ClientName     string
	FinishTime     time.Time
	LastTime       time.Time
	CommandSummary string
	RequestSummary string `gorm:"type:text"`
	RequestSize    int64
	RequestSHA256  string `gorm:"column:request_sha256"`
	HasRequest     bool
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
	deadline := t.Deadline
	if deadline.IsZero() && !t.Created.IsZero() {
		deadline = t.Created.Add(configs.DefaultTaskTimeout)
	}
	return &clientpb.Task{
		TaskId:         uint32(t.Seq),
		Type:           t.Type,
		SessionId:      t.SessionID,
		Cur:            int32(t.Cur),
		Total:          int32(t.Total),
		Description:    t.Description,
		Callby:         t.ClientName,
		Timeout:        !deadline.IsZero() && time.Now().After(deadline),
		Finished:       !t.FinishTime.IsZero() || t.Cur == t.Total,
		CreatedAt:      t.Created.Unix(),
		FinishedAt:     t.FinishTime.Unix(),
		CommandSummary: t.CommandSummary,
		RequestSummary: t.RequestSummary,
		RequestSize:    t.RequestSize,
		RequestSha256:  t.RequestSHA256,
		HasRequest:     t.HasRequest,
	}
}
