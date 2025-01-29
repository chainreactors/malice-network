package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/helper/types"
	"gorm.io/gorm"
)

type Session struct {
	SessionID   string `gorm:"primaryKey;->;<-:create;type:uuid;"`
	RawID       uint32
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	Note        string
	GroupName   string
	Target      string
	Initialized bool
	Type        string
	PipelineID  string
	ListenerID  string
	IsAlive     bool
	LastCheckin int64
	IsRemoved   bool                  `gorm:"default:false"`
	Data        *types.SessionContext `gorm:"-"`
	DataString  string                `gorm:"column:data"`
	ProfileName string                `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ProfileName;references:Name"`
	Profile     Profile               `gorm:"foreignKey:ProfileName;references:Name;"`
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

func (s *Session) BeforeSave(tx *gorm.DB) error {
	if s.Data != nil {
		data, err := json.Marshal(s.Data)
		if err != nil {
			return err
		}
		s.DataString = string(data)
	}
	return nil
}

func (s *Session) AfterFind(tx *gorm.DB) error {
	if s.DataString == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(s.DataString), &s.Data); err != nil {
		return err
	}
	return nil
}

func (s *Session) ToProtobuf() *clientpb.Session {
	if s == nil {
		return nil
	}

	// 将整个 Data 序列化为 JSON 字符串
	var dataString string
	if s.Data != nil {
		if jsonBytes, err := json.Marshal(s.Data.Data()); err == nil {
			dataString = string(jsonBytes)
		}
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
		IsPrivilege:   s.Data.IsPrivilege,
		LastCheckin:   s.LastCheckin,
		Filepath:      s.Data.Filepath,
		Workdir:       s.Data.WorkDir,
		Locate:        s.Data.Locale,
		Proxy:         s.Data.ProxyURL,
		Os:            s.Data.Os,
		Process:       s.Data.Process,
		Timer:         &implantpb.Timer{Interval: s.Data.Interval, Jitter: s.Data.Jitter},
		Modules:       s.Data.Modules,
		Timediff:      time.Now().Unix() - s.LastCheckin,
		Addons:        s.Data.Addons,
		Name:          s.ProfileName,
		Data:          dataString,
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
