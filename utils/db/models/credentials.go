package models

import (
	"time"

	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Credential - Represents a piece of loot
type Credential struct {
	ID             uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt      time.Time `gorm:"->;<-:create;"`
	OriginHostUUID uuid.UUID `gorm:"type:uuid;"`

	Collection string
	Username   string
	Plaintext  string
	Hash       string // https://hashcat.net/wiki/doku.php?id=example_hashes
	HashType   int32
	IsCracked  bool
}

func (c *Credential) ToProtobuf() *clientpb.Credential {
	return &clientpb.Credential{
		ID:             c.ID.String(),
		Username:       c.Username,
		Plaintext:      c.Plaintext,
		Hash:           c.Hash,
		HashType:       clientpb.HashType(c.HashType),
		IsCracked:      c.IsCracked,
		OriginHostUUID: c.OriginHostUUID.String(),
		Collection:     c.Collection,
	}
}

// BeforeCreate - GORM hook
func (c *Credential) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}
