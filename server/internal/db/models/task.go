package models

import (
	"encoding/json"
	"github.com/chainreactors/logs"
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
	Name     string `json:"name"`
	NickName string `json:"nick_name"`
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Command  string `json:"command"`
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

func (td *FileDescription) ToJsonString() (string, error) {
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

func (t *Task) toFileDescription() (*FileDescription, error) {
	var desc FileDescription
	err := json.Unmarshal([]byte(t.Description), &desc)
	if err != nil {
		return nil, err
	}
	return &desc, nil
}

func (t *Task) ToFileProtobuf() *clientpb.File {
	re := regexp.MustCompile(`-(\d+)$`)
	match := re.FindStringSubmatch(t.ID)
	if len(match) < 1 {
		return &clientpb.File{}
	}
	file, err := t.toFileDescription()
	if err != nil {
		logs.Log.Errorf("Error parsing task file JSON: %v", err)
		return &clientpb.File{}
	}
	return &clientpb.File{
		Name:   file.Name,
		Local:  file.Name,
		TempId: file.NickName,
		Remote: file.Path,
		Op:     t.Type,
	}
}
