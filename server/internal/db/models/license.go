package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

type License struct {
	ID         uuid.UUID `gorm:"primaryKey;autoIncrement"`
	Username   string    `gorm:"unique"`
	Email      string
	Token      string
	ExpireAt   time.Time
	BuildCount int
	MaxBuilds  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// BeforeCreate - GORM hook to automatically set values
func (l *License) BeforeCreate(tx *gorm.DB) (err error) {
	l.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	l.CreatedAt = time.Now()
	return nil
}
