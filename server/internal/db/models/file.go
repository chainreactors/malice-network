package models

import (
	"encoding/json"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"gorm.io/gorm"
	"regexp"
	"strconv"
	"time"
)

type File struct {
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
	Name       string `json:"name"`
	SourcePath string `json:"source_path"`
	SavePath   string `json:"save_path"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
	Command    string `json:"command"`
}

// BeforeCreate - GORM hook
func (f *File) BeforeCreate(tx *gorm.DB) (err error) {
	if err != nil {
		return err
	}
	f.CreatedAt = time.Now()
	return nil
}

func (f *File) UpdateCur(db *gorm.DB, newCur int) error {
	return db.Model(f).Update("cur", newCur).Error
}

func (td *FileDescription) ToJsonString() (string, error) {
	jsonString, err := json.Marshal(td)
	if err != nil {
		return "", err
	}
	return string(jsonString), nil
}

func (f *File) ToProtobuf() *clientpb.Task {
	re := regexp.MustCompile(`-(\d+)$`)
	match := re.FindStringSubmatch(f.ID)
	if len(match) < 1 {
		return &clientpb.Task{}
	}
	id, _ := strconv.ParseUint(match[1], 10, 32)
	return &clientpb.Task{
		TaskId:    uint32(id),
		Type:      f.Type,
		SessionId: f.SessionID,
		Cur:       int32(f.Cur),
		Total:     int32(f.Total),
	}
}

func (f *File) toFileDescription() (*FileDescription, error) {
	var desc FileDescription
	err := json.Unmarshal([]byte(f.Description), &desc)
	if err != nil {
		return nil, err
	}
	return &desc, nil
}

func (f *File) ToFileProtobuf() *clientpb.File {
	re := regexp.MustCompile(`-(\d+)$`)
	match := re.FindStringSubmatch(f.ID)
	if len(match) < 1 {
		return &clientpb.File{}
	}
	file, err := f.toFileDescription()
	if err != nil {
		logs.Log.Errorf("Error parsing task file JSON: %v", err)
		return &clientpb.File{}
	}
	return &clientpb.File{
		TaskId:    match[1],
		Name:      file.Name,
		Local:     file.Name,
		Checksum:  file.Checksum,
		Remote:    file.SourcePath,
		SessionId: f.SessionID,
		Op:        f.Type,
	}
}
