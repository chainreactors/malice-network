package models

import (
	"errors"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

// Listener
type Listener struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Name      string    `gorm:"uniqueIndex"`
}

// BeforeCreate - GORM hook
func (l *Listener) BeforeCreate(tx *gorm.DB) (err error) {
	l.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	l.CreatedAt = time.Now()
	return nil
}

func CreateListener(dbSession *gorm.DB, name string) error {
	var listener Listener
	result := dbSession.Where("name = ?", name).Delete(&listener)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
	}
	listener.Name = name
	err := dbSession.Create(&listener).Error
	return err
}

func ListListeners(dbSession *gorm.DB) ([]Listener, error) {
	var listeners []Listener
	err := dbSession.Find(&listeners).Error
	return listeners, err
}
