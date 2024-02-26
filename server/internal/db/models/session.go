package models

import (
	"errors"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"gorm.io/gorm"
	"time"
)

type Session struct {
	SessionID  string    `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt  time.Time `gorm:"->;<-:create;"`
	RemoteAddr string
	ListenerId string
	IsAlive    bool
	Os         *Os      `gorm:"embedded"`
	Process    *Process `gorm:"embedded"`
	Time       *Timer   `gorm:"embedded"`
	Last       time.Time
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
	var existingSession Session
	result := tx.Where("session_id = ?", s.SessionID).First(&existingSession)
	if result.Error == nil {
		return errors.New("exists")
	}
	s.CreatedAt = time.Now()
	return nil
}

func ConvertToSessionDB(session *core.Session) *Session {
	currentTime := time.Now()
	return &Session{
		SessionID:  session.ID,
		RemoteAddr: session.RemoteAddr,
		ListenerId: session.ListenerId,
		Os:         convertToOsDB(session.Os),
		Process:    convertToProcessDB(session.Process),
		Time:       convertToTimeDB(session.Timer),
		Last:       currentTime,
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

func (s *Session) ToProtobuf() *lispb.RegisterSession {
	return &lispb.RegisterSession{
		SessionId:  s.SessionID,
		ListenerId: s.ListenerId,
		RemoteAddr: s.RemoteAddr,
		RegisterData: &commonpb.Register{
			Os:      s.Os.toProtobuf(),
			Process: s.Process.toProtobuf(),
			Timer:   s.Time.toProtobuf(),
		},
	}
}

func (o *Os) toProtobuf() *commonpb.Os {
	return &commonpb.Os{
		Name:     o.Name,
		Version:  o.Version,
		Arch:     o.Arch,
		Username: o.Username,
		Hostname: o.Name,
		Locale:   o.Locale,
	}
}
func (p *Process) toProtobuf() *commonpb.Process {
	return &commonpb.Process{
		Uid:  p.Uid,
		Pid:  p.Pid,
		Gid:  p.Gid,
		Name: p.Name,
		Args: p.Args,
	}
}
func (t *Timer) toProtobuf() *commonpb.Timer {
	return &commonpb.Timer{
		Interval:    t.Interval,
		Jitter:      t.Jitter,
		Heartbeat:   t.Heartbeat,
		LastCheckin: t.LastCheckin,
	}
}
func FindActiveSessions(dbSession *gorm.DB) ([]Session, error) {
	var activeSessions []Session
	result := dbSession.Where("is_alive = ?", true).Find(&activeSessions)
	if result.Error != nil {
		return nil, result.Error
	}
	return activeSessions, nil
}

func UpdateLast(dbSession *gorm.DB, sessionID string) error {
	var session Session
	result := dbSession.Where("session_id = ?", sessionID).First(&session)
	loc := time.Now().Location()
	if result.Error != nil {
		return result.Error
	}
	session.Last = time.Now().In(loc)
	result = dbSession.Save(&session)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
