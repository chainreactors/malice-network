package models

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Operator - Colletions of content to serve from HTTP(S)
type Operator struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Name      string    `gorm:"uniqueIndex"`
	Token     string    `gorm:"uniqueIndex"`
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

func CreateOperator(dbSession *gorm.DB, name, token string) error {
	var operator Operator
	result := dbSession.Where("name = ?", name).Delete(&operator)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
	}
	operator.Name = name
	operator.Token = token
	err := dbSession.Create(&operator).Error
	return err

}
