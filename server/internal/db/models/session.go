package models

import (
	"encoding/json"
	"errors"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"time"

	"gorm.io/gorm"
)

type Session struct {
	SessionID   string `gorm:"primaryKey;->;<-:create;type:uuid;index:idx_sessions_list_default,priority:4,sort:asc;index:idx_sessions_list_group,priority:5,sort:asc"`
	RawID       uint32
	CreatedAt   time.Time `gorm:"->;<-:create;index:idx_sessions_list_default,priority:3,sort:desc;index:idx_sessions_list_group,priority:4,sort:desc;index:idx_sessions_trend,priority:2,sort:asc"`
	Note        string
	GroupName   string `gorm:"index:idx_sessions_list_group,priority:2,sort:asc"`
	Target      string
	Initialized bool
	Type        string
	PipelineID  string
	ListenerID  string
	IsAlive     bool `gorm:"index:idx_sessions_list_default,priority:2,sort:desc;index:idx_sessions_list_group,priority:3,sort:desc"`
	LastCheckin int64
	IsRemoved   bool                   `gorm:"default:false;index:idx_sessions_list_default,priority:1,sort:asc;index:idx_sessions_list_group,priority:1,sort:asc;index:idx_sessions_trend,priority:1,sort:asc"`
	Data        *client.SessionContext `gorm:"-"`
	DataString  string                 `gorm:"column:data"`

	ProfileName string  `gorm:"index;"`
	Profile     Profile `gorm:"foreignKey:ProfileName;references:Name;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
	// Note: The CreateOrRecoverSession helper function handles checking for
	// existing sessions (including soft-deleted ones) before creation,
	// so this check is primarily a safety net for direct Create() calls
	var existingSession Session
	result := tx.Unscoped().Where("session_id = ?", s.SessionID).First(&existingSession)
	if result.Error == nil {
		return errors.New("session exists - use CreateOrRecoverSession helper instead")
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
	data := s.Data
	if data == nil {
		data = &client.SessionContext{}
	}
	info := data.SessionInfo
	if info == nil {
		info = &client.SessionInfo{}
	}

	// 将整个 Data 序列化为 JSON 字符串
	var dataString string
	if s.Data != nil {
		if jsonBytes, err := json.Marshal(s.Data); err == nil {
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
		IsPrivilege:   info.IsPrivilege,
		LastCheckin:   s.LastCheckin,
		Filepath:      info.Filepath,
		Workdir:       info.WorkDir,
		Locate:        info.Locale,
		Proxy:         info.ProxyURL,
		Os:            info.Os,
		Process:       info.Process,
		Timer:         &implantpb.Timer{Expression: info.Expression, Jitter: info.Jitter},
		Modules:       data.Modules,
		CreatedAt:     s.CreatedAt.Unix(),
		Addons:        data.Addons,
		Name:          s.ProfileName,
		KeyPair:       data.KeyPair, // 添加密钥对
		Data:          dataString,
	}
}

type Timer struct {
	Expression string  `json:"expression"`
	Jitter     float64 `json:"jitter"`
}

func (t *Timer) toProtobuf() *implantpb.Timer {
	return &implantpb.Timer{
		Expression: t.Expression,
		Jitter:     t.Jitter,
	}
}

func FromTimePb(timer *implantpb.Timer) *Timer {
	if timer == nil {
		return &Timer{}
	}
	return &Timer{
		Expression: timer.Expression,
		Jitter:     timer.Jitter,
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
