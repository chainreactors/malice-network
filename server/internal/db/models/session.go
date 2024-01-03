package models

import (
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/server/core"
	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"time"
)

type Session struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	SessionId  string `gorm:"uniqueIndex"`
	RemoteAddr string
	ListenerId string
	Os         *Os         `gorm:"embedded"`
	Process    *Process    `gorm:"embedded"`
	Time       *Timer      `gorm:"embedded"`
	Task       *core.Tasks `gorm:"embedded"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
	var existingSession Session
	result := tx.Where("session_id = ?", s.SessionId).First(&existingSession)
	if result.Error == nil {
		return errors.New("exists")
	}
	s.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	s.CreatedAt = time.Now()
	return nil
}

func ConvertToSessionDB(session *core.Session) *Session {
	return &Session{
		SessionId:  session.ID,
		RemoteAddr: session.RemoteAddr,
		ListenerId: session.ListenerId,
		Os:         convertToOsDB(session.Os),
		Process:    convertToProcessDB(session.Process),
		Time:       convertToTimeDB(session.Timer),
		Task:       session.Tasks,
	}
}

func convertToOsDB(os *commonpb.Os) *Os {
	return &Os{
		Name:     os.Name,
		Version:  os.Version,
		Arch:     os.Arch,
		Username: os.Username,
		Hostname: os.Hostname,
		Locale:   os.Locale,
	}
}
func convertToProcessDB(process *commonpb.Process) *Process {
	return &Process{
		Uid:  process.Uid,
		Pid:  process.Pid,
		Gid:  process.Gid,
		Name: process.Name,
		Args: process.Args,
	}
}
func convertToTimeDB(timer *commonpb.Timer) *Timer {
	return &Timer{
		Interval:    timer.Interval,
		Jitter:      timer.Jitter,
		Heartbeat:   timer.Heartbeat,
		LastCheckin: timer.LastCheckin,
	}
}

type Os struct {
	Name     string `gorm:"type:varchar(255)" json:"name"`
	Version  string `gorm:"type:varchar(255)" json:"version"`
	Arch     string `gorm:"type:varchar(255)" json:"arch"`
	Username string `gorm:"type:varchar(255)" json:"username"`
	Hostname string `gorm:"type:varchar(255)" json:"hostname"`
	Locale   string `gorm:"type:varchar(255)" json:"locale"`
}

type Process struct {
	Uid  string `gorm:"type:varchar(255)" json:"uid"`
	Pid  int32  `json:"pid"`
	Gid  string `gorm:"type:varchar(255)" json:"gid"`
	Name string `gorm:"type:varchar(255)" json:"name"`
	Args string `gorm:"type:varchar(255)" json:"args"`
}

type Timer struct {
	Interval    uint64 `json:"interval"`
	Jitter      uint64 `json:"jitter"`
	Heartbeat   uint64 `json:"heartbeat"`
	LastCheckin uint64 `json:"last_checkin"`
}
