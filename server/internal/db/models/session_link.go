package models

import (
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
)

const SessionLinkSourceManual = "manual"

// SessionLink stores the current parent of a session. ChildSessionID is the
// primary key so a child can have at most one parent.
type SessionLink struct {
	ChildSessionID  string `gorm:"column:child_session_id;primaryKey;type:uuid;not null"`
	ParentSessionID string `gorm:"column:parent_session_id;type:uuid;not null;index"`
	Source          string `gorm:"not null;default:manual"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (link *SessionLink) ToProtobuf() *clientpb.SessionLink {
	if link == nil {
		return nil
	}
	return &clientpb.SessionLink{
		ParentSessionId: link.ParentSessionID,
		ChildSessionId:  link.ChildSessionID,
		Source:          link.Source,
		CreatedAt:       link.CreatedAt.Unix(),
		UpdatedAt:       link.UpdatedAt.Unix(),
	}
}
