package models

import (
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// KeyValue - Represents an implant
type KeyValue struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Key   string `gorm:"unique;"`
	Value string
}

// BeforeCreate - GORM hook
func (k *KeyValue) BeforeCreate(tx *gorm.DB) (err error) {
	k.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	k.CreatedAt = time.Now()
	return nil
}
