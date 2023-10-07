package models

import (
	"time"

	"gorm.io/gorm"
)

// KeyExHistory - Represents an implant
type KeyExHistory struct {
	Sha256    string    `gorm:"primaryKey;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
}

// BeforeCreate - GORM hook
func (k *KeyExHistory) BeforeCreate(tx *gorm.DB) (err error) {
	k.CreatedAt = time.Now()
	return nil
}
