package models

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Operator - Colletions of content to serve from HTTP(S)
type Operator struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Name      string
	Token     string `gorm:"uniqueIndex"`
}

// BeforeCreate - GORM hook
func (o *Operator) BeforeCreate(tx *gorm.DB) (err error) {
	o.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	o.CreatedAt = time.Now()
	return nil
}

// GenerateOperatorToken - Generate a new operator auth token
func GenerateOperatorToken() string {
	buf := make([]byte, 32)
	n, err := rand.Read(buf)
	if err != nil || n != len(buf) {
		panic(errors.New("failed to read from secure rand"))
	}
	return hex.EncodeToString(buf)
}
