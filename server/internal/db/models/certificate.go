package models

import (
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

// Certificate - Certificate database model
type Certificate struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt      time.Time `gorm:"->;<-:create;"`
	CommonName     string    `gorm:"uniqueIndex"`
	CAType         string
	KeyType        string
	CertificatePEM string
	PrivateKeyPEM  string
}

// BeforeCreate - GORM hook to automatically set values
func (c *Certificate) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}
