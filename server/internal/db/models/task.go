package models

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"gorm.io/gorm"
	"regexp"
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

type FileDescription struct {
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

func (td *FileDescription) ToJson() (string, error) {
	jsonString, err := json.Marshal(td)
	if err != nil {
		return "", err
	}
	return string(jsonString), nil
}

func (t *Task) ToProtobuf() *clientpb.Task {
	re := regexp.MustCompile(`-(\d+)$`)
	match := re.FindStringSubmatch(t.ID)
	if len(match) < 1 {
		return &clientpb.Task{}
	}
	id, _ := strconv.ParseUint(match[1], 10, 32)
	return &clientpb.Task{
		TaskId:    uint32(id),
		Type:      t.Type,
		SessionId: t.SessionID,
		Cur:       int32(t.Cur),
		Total:     int32(t.Total),
	}
}

func (t *Task) ToDescProtobuf() *clientpb.TaskDesc {
	re := regexp.MustCompile(`-(\d+)$`)
	match := re.FindStringSubmatch(t.ID)
	if len(match) < 1 {
		return &clientpb.TaskDesc{}
	}
	id, _ := strconv.ParseUint(match[1], 10, 32)
	return &clientpb.TaskDesc{
		TaskId:      uint32(id),
		Type:        t.Type,
		Cur:         int32(t.Cur),
		Total:       int32(t.Total),
		Description: t.Description,
	}
}
