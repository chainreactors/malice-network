package models

import (
	"time"

	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// DNSCanary - Colletions of content to serve from HTTP(S)
type DNSCanary struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	ImplantName   string
	Domain        string
	Triggered     bool
	FirstTrigger  time.Time
	LatestTrigger time.Time
	Count         uint32
}

// BeforeCreate - GORM hook
func (c *DNSCanary) BeforeCreate(tx *gorm.DB) (err error) {
	c.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	c.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Converts to protobuf object
func (c *DNSCanary) ToProtobuf() *clientpb.DNSCanary {
	return &clientpb.DNSCanary{
		ImplantName:    c.ImplantName,
		Domain:         c.Domain,
		Triggered:      c.Triggered,
		FirstTriggered: c.FirstTrigger.Format(time.RFC1123),
		LatestTrigger:  c.LatestTrigger.Format(time.RFC1123),
		Count:          c.Count,
	}
}
