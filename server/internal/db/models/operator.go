package models

import (
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"time"

	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Role constants for operator authorization
const (
	RoleAdmin    = "admin"    // Full access, can manage operators and rules
	RoleOperator = "operator" // Access to MaliceRPC (implant operations)
	RoleListener = "listener" // Access to ListenerRPC only
)

// ValidRoles is the set of recognized role values.
var ValidRoles = []string{RoleAdmin, RoleOperator, RoleListener}

// Operator represents a registered client or listener identity.
type Operator struct {
	ID               uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt        time.Time `gorm:"->;<-:create;"`
	Name             string    `gorm:"uniqueIndex"`
	Remote           string    `gorm:"type:string;"`
	Type             string    `gorm:"type:string;"`
	Role             string    `gorm:"type:string;default:'operator'"`
	Fingerprint      string    `gorm:"uniqueIndex;type:string;"`
	Revoked          bool      `gorm:"default:false"`
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
