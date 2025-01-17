package models

import (
	"encoding/json"
	"errors"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db/content"
	"gorm.io/gorm"
	"time"
)

type Session struct {
	SessionID   string `gorm:"primaryKey;->;<-:create;type:uuid;"`
	RawID       uint32
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	Name        string
	Note        string
	GroupName   string
	Target      string
	Initialized bool
	Type        string
	IsPrivilege bool
	PipelineID  string
	ListenerID  string
	IsAlive     bool
	Context     string
	LastCheckin int64
	Interval    uint64
	Jitter      float64
	IsRemoved   bool     `gorm:"default:false"`
	Os          *Os      `gorm:"embedded;embeddedPrefix:os_"`
	Process     *Process `gorm:"embedded;embeddedPrefix:process_"`

	ProfileName string  `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ProfileName;references:Name"`
	Profile     Profile `gorm:"foreignKey:ProfileName;references:Name;"`
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

func (s *Session) ToProtobuf() *clientpb.Session {
	cont, _ := content.RecoverSessionContext(s.Context)
	if s.Os == nil {
		s.Os = &Os{}
	}
	if s.Process == nil {
		s.Process = &Process{}
	}
	return &clientpb.Session{
		Type:          s.Type,
		SessionId:     s.SessionID,
		RawId:         s.RawID,
		PipelineId:    s.PipelineID,
		ListenerId:    s.ListenerID,
		Note:          s.Note,
		GroupName:     s.GroupName,
		Target:        s.Target,
		IsAlive:       s.IsAlive,
		IsInitialized: s.Initialized,
		IsPrivilege:   s.IsPrivilege,
		LastCheckin:   s.LastCheckin,
		Os:            s.Os.toProtobuf(),
		Process:       s.Process.toProtobuf(),
		Timer:         &implantpb.Timer{Interval: s.Interval, Jitter: s.Jitter},
		Modules:       cont.Modules,
		Timediff:      time.Now().Unix() - s.LastCheckin,
		Addons:        cont.Addons,
		Name:          s.ProfileName,
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

func (o *Os) toProtobuf() *implantpb.Os {
	return &implantpb.Os{
		Name:     o.Name,
		Version:  o.Version,
		Arch:     o.Arch,
		Username: o.Username,
		Hostname: o.Name,
		Locale:   o.Locale,
	}
}

func FromOsPb(os *implantpb.Os) *Os {
	if os == nil {
		return &Os{}
	}
	return &Os{
		Name:     os.Name,
		Version:  os.Version,
		Arch:     os.Arch,
		Username: os.Username,
		Hostname: os.Hostname,
		Locale:   os.Locale,
	}
}

type Timer struct {
	Interval uint64  `json:"interval"`
	Jitter   float64 `json:"jitter"`
}

func (t *Timer) toProtobuf() *implantpb.Timer {
	return &implantpb.Timer{
		Interval: t.Interval,
		Jitter:   t.Jitter,
	}
}

func FromTimePb(timer *implantpb.Timer) *Timer {
	if timer == nil {
		return &Timer{}
	}
	return &Timer{
		Interval: timer.Interval,
		Jitter:   timer.Jitter,
	}
}

type Process struct {
	Name  string `gorm:"type:varchar(255)" json:"name"`
	Pid   int32  `json:"pid"`
	Ppid  int32  `json:"ppid"`
	Owner string `gorm:"type:varchar(255)" json:"owner"`
	Arch  string `gorm:"type:varchar(255)" json:"arch"`
	Path  string `gorm:"type:varchar(255)" json:"path"`
	Args  string `gorm:"type:varchar(255)" json:"args"`
}

func (p *Process) toProtobuf() *implantpb.Process {
	return &implantpb.Process{
		Name:  p.Name,
		Pid:   uint32(p.Pid),
		Ppid:  uint32(p.Ppid),
		Owner: p.Owner,
		Arch:  p.Arch,
		Path:  p.Path,
		Args:  p.Args,
	}
}

func FromProcessPb(process *implantpb.Process) *Process {
	if process == nil {
		return &Process{}
	}
	return &Process{
		Name:  process.Name,
		Pid:   int32(process.Pid),
		Ppid:  int32(process.Ppid),
		Owner: process.Owner,
		Arch:  process.Arch,
		Path:  process.Path,
		Args:  process.Args,
	}
}

// FromRegister - convert session to context json string
func FromRegister(register *implantpb.Register) string {
	content, err := json.Marshal(register)
	if err != nil {
		return ""
	}
	return string(content)
}

func ToRegister(context string) *implantpb.Register {
	var register *implantpb.Register
	err := json.Unmarshal([]byte(context), &register)
	if err != nil {
		return nil
	}
	return register
}
