package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

type History struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	LicenseID uuid.UUID `gorm:"type:char(36);index"`
	BuildName string    `gorm:"index"`
	CreatedAt time.Time

	License License `gorm:"foreignKey:LicenseID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Build   Builder `gorm:"foreignKey:BuildName;references:Name;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

// BeforeCreate - GORM hook to automatically set values
func (h *History) BeforeCreate(tx *gorm.DB) (err error) {
	h.CreatedAt = time.Now()
	return nil
}
