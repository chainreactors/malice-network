package models

import (
	"github.com/chainreactors/malice-network/helper/proto/listener/lispb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"time"
)

// WebsiteContent - Single table that combines Website and WebContent
type WebsiteContent struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	Name        string `gorm:"unique;"`
	Path        string `gorm:""`
	Size        uint64 `gorm:""`
	ContentType string `gorm:""`
}

// BeforeCreate - GORM hook to automatically set values
func (wc *WebsiteContent) BeforeCreate(tx *gorm.DB) (err error) {
	wc.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	wc.CreatedAt = time.Now()
	return nil
}

// ToProtobuf - Converts to protobuf object
func (wc *WebsiteContent) ToProtobuf(webContentDir string) *lispb.Website {
	contents, _ := os.ReadFile(filepath.Join(webContentDir, wc.Path))
	return &lispb.Website{
		ID:   wc.ID.String(),
		Name: wc.Name,
		Contents: map[string]*lispb.WebContent{
			wc.ID.String(): {
				ID:          wc.ID.String(),
				WebsiteID:   wc.ID.String(),
				Path:        wc.Path,
				Size:        wc.Size,
				ContentType: wc.ContentType,
				Content:     contents,
			},
		},
	}
}

// FromProtobuf - Converts from protobuf object to WebsiteContent
func WebsiteContentFromProtobuf(pbWebContent *lispb.WebContent) WebsiteContent {
	siteUUID, _ := uuid.FromString(pbWebContent.ID)
	return WebsiteContent{
		ID:          siteUUID,
		Name:        pbWebContent.Name,
		Path:        pbWebContent.Path,
		Size:        pbWebContent.Size,
		ContentType: pbWebContent.ContentType,
	}
}
