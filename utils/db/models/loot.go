package models

import (
	"time"

	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Loot - Represents a piece of loot
type Loot struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	FileType int
	Name     string
	Size     int64

	OriginHostID uuid.UUID `gorm:"type:uuid;"`
}

func (l *Loot) ToProtobuf() *clientpb.Loot {
	return &clientpb.Loot{
		ID:             l.ID.String(),
		FileType:       clientpb.FileType(l.FileType),
		Name:           l.Name,
		Size:           l.Size,
		OriginHostUUID: l.OriginHostID.String(),
	}
}

// BeforeCreate - GORM hook
func (l *Loot) BeforeCreate(tx *gorm.DB) (err error) {
	l.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	l.CreatedAt = time.Now()
	return nil
}
