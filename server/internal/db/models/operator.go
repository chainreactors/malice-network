package models

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Operator - Colletions of content to serve from HTTP(S)
type Operator struct {
	ID               uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt        time.Time `gorm:"->;<-:create;"`
	Name             string    `gorm:"uniqueIndex"`
	Remote           string    `gorm:"type:string;"`
	Type             string    `gorm:"type:string;"`
	CAType           int
	KeyType          string
	CaCertificatePEM string
	CertificatePEM   string
	PrivateKeyPEM    string
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

func (o *Operator) ToProtobuf() *clientpb.Client {
	return &clientpb.Client{
		Name: o.Name,
		Type: o.Type,
	}
}

func (o *Operator) ToListener() *clientpb.Listener {
	if o == nil {
		return nil
	}
	return &clientpb.Listener{
		Id: o.Name,
		Ip: o.Remote,
	}
}
