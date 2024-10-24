package models

import (
	"encoding/json"
	"errors"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"github.com/chainreactors/malice-network/server/internal/core"
	"gorm.io/gorm"
	"time"
)

type Session struct {
	SessionID   string    `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	Note        string
	GroupName   string
	RemoteAddr  string
	IsPrivilege bool
	ListenerId  string
	IsAlive     bool
	Context     string
	IsRemoved   bool     `gorm:"default:false"`
	Os          *Os      `gorm:"embedded"`
	Process     *Process `gorm:"embedded"`
	Time        *Timer   `gorm:"embedded"`
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

func (s *Session) ToClientProtobuf() *clientpb.Session {
	ctx := recoverFromContext(s.Context)

	return &clientpb.Session{
		SessionId:  s.SessionID,
		ListenerId: s.ListenerId,
		Note:       s.Note,
		RemoteAddr: s.RemoteAddr,
		IsAlive:    s.IsAlive,
		GroupName:  s.GroupName,
		Os:         s.Os.toProtobuf(),
		Process:    s.Process.toProtobuf(),
		Timer:      s.Time.toProtobuf(),
		Modules:    ctx.Modules,
		Addons:     ctx.Addons,
	}
}

func (s *Session) ToRegisterProtobuf() *lispb.RegisterSession {
	return &lispb.RegisterSession{
		SessionId:  s.SessionID,
		ListenerId: s.ListenerId,
		RemoteAddr: s.RemoteAddr,
		RegisterData: &implantpb.Register{
			Name:  s.Note,
			Timer: s.Time.toProtobuf(),
			Sysinfo: &implantpb.SysInfo{
				Os:      s.Os.toProtobuf(),
				Process: s.Process.toProtobuf(),
			},
		},
	}
}

func ConvertToSessionDB(session *core.Session) *Session {
	return &Session{
		SessionID:   session.ID,
		GroupName:   "default",
		RemoteAddr:  session.RemoteAddr,
		ListenerId:  session.PipelineID,
		IsPrivilege: session.IsPrivilege,
		Context:     convertToContext(session.SessionContext),
		Os:          convertToOsDB(session.Os),
		Process:     convertToProcessDB(session.Process),
		Time:        convertToTimeDB(session.Timer),
	}
}

func convertToContext(context *core.SessionContext) string {
	content, err := json.Marshal(context)
	if err != nil {
		return ""
	}
	return string(content)
}

func recoverFromContext(context string) *core.SessionContext {
	var ctx *core.SessionContext
	err := json.Unmarshal([]byte(context), &ctx)
	if err != nil {
		return nil
	}
	return ctx
}

func convertToOsDB(os *implantpb.Os) *Os {
	return &Os{
		Name:     os.Name,
		Version:  os.Version,
		Arch:     os.Arch,
		Username: os.Username,
		Hostname: os.Hostname,
		Locale:   os.Locale,
	}
}
func convertToProcessDB(process *implantpb.Process) *Process {
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
func convertToTimeDB(timer *implantpb.Timer) *Timer {
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

type Timer struct {
	Interval    uint64 `json:"interval"`
	Jitter      uint64 `json:"jitter"`
	Heartbeat   uint64 `json:"heartbeat"`
	LastCheckin uint64 `json:"last_checkin"`
}

func (t *Timer) toProtobuf() *implantpb.Timer {
	return &implantpb.Timer{
		Interval:    t.Interval,
		Jitter:      t.Jitter,
		Heartbeat:   t.Heartbeat,
		LastCheckin: t.LastCheckin,
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
