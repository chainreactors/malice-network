package models

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
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

	Name        string `gorm:""`
	Path        string `gorm:""`
	Size        uint64 `gorm:""`
	ContentType string `gorm:""`
	Type        string `gorm:""`
	Parser      string `gorm:""`
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
func (wc *WebsiteContent) ToProtobuf(webContentDir string) *clientpb.Website {
	contents, _ := os.ReadFile(filepath.Join(webContentDir, wc.Path))
	return &clientpb.Website{
		ID:   wc.ID.String(),
		Root: wc.Path,
		Contents: map[string]*clientpb.WebContent{
			wc.ID.String(): {
				Id:          wc.ID.String(),
				WebsiteID:   wc.ID.String(),
				Path:        wc.Path,
				Size:        wc.Size,
				ContentType: wc.ContentType,
				Content:     contents,
				Type:        wc.Type,
				Parser:      wc.Parser,
			},
		},
	}
}

// FromProtobuf - Converts from protobuf object to WebsiteContent
func WebsiteContentFromProtobuf(pbWebContent *clientpb.WebContent) WebsiteContent {
	return WebsiteContent{
		Name:        pbWebContent.Name,
		Path:        pbWebContent.Path,
		Size:        pbWebContent.Size,
		ContentType: pbWebContent.ContentType,
		Type:        pbWebContent.Type,
		Parser:      pbWebContent.Parser,
	}
}
