package models

import (
	"errors"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Operator - Colletions of content to serve from HTTP(S)
type Operator struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`
	Name      string    `gorm:"uniqueIndex"`
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

func CreateOperator(dbSession *gorm.DB, name string) error {
	var operator Operator
	result := dbSession.Where("name = ?", name).Delete(&operator)
	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
	}
	operator.Name = name
	err := dbSession.Create(&operator).Error
	return err

}

func ListOperators(dbSession *gorm.DB) (*clientpb.Clients, error) {
	var operators []Operator
	err := dbSession.Find(&operators).Error
	if err != nil {
		return nil, err
	}

	var clients []*clientpb.Client
	for _, op := range operators {
		client := &clientpb.Client{
			Name: op.Name,
		}
		clients = append(clients, client)
	}
	pbClients := &clientpb.Clients{
		Clients: clients,
	}
	return pbClients, nil
}
