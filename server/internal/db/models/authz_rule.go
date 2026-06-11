package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// AuthzRule maps a role to allowed (or denied) gRPC method patterns.
// Pattern supports:
//   - Exact match: "/clientrpc.MaliceRPC/GetSessions"
//   - Service wildcard: "/clientrpc.MaliceRPC/*"
//   - Package prefix: "/listenerrpc.*"
type AuthzRule struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Role      string    `gorm:"index;type:string;"`
	Method    string    `gorm:"type:string;"`
	Allow     bool      `gorm:"default:true"`
}

// BeforeCreate - GORM hook
func (a *AuthzRule) BeforeCreate(tx *gorm.DB) (err error) {
	a.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	a.CreatedAt = time.Now()
	return nil
}
