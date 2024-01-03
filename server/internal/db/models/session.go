package models

import (
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/server/core"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

type Session struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	SessionId  string
	RemoteAddr string
	ListenerId string
	Os         *commonpb.Os
	Process    *commonpb.Process
	Time       *commonpb.Timer
	Task       *core.Tasks
}

func (s *Session) BeforeCreate(tx *gorm.DB) (err error) {
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
		Os:         session.Os,
		Process:    session.Process,
		Time:       session.Timer,
		Task:       session.Tasks,
	}
}
