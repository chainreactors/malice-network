package models

import (
	"time"

	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// Website - Colletions of content to serve from HTTP(S)
type Website struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name string `gorm:"unique;"` // Website Name

	WebContents []WebContent
}

// BeforeCreate - GORM hook
func (w *Website) BeforeCreate(tx *gorm.DB) (err error) {
	w.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	w.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Converts to protobuf object
func (w *Website) ToProtobuf() *clientpb.Website {
	return &clientpb.Website{
		Name:     w.Name,
		Contents: map[string]*clientpb.WebContent{},
	}
}

// WebContent - One piece of content mapped to a path
type WebContent struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	WebsiteID uuid.UUID `gorm:"type:uuid;"`

	Path        string `gorm:"primaryKey"`
	Size        int
	ContentType string
}

// BeforeCreate - GORM hook to automatically set values
func (wc *WebContent) BeforeCreate(tx *gorm.DB) (err error) {
	wc.ID, err = uuid.NewV4()
	return err
}

// ToProtobuf - Converts to protobuf object
func (wc *WebContent) ToProtobuf(content []byte) *clientpb.WebContent {
	return &clientpb.WebContent{
		Path:        wc.Path,
		Size:        uint64(wc.Size),
		ContentType: wc.ContentType,
		Content:     content,
	}
}
