package models

import (
	"github.com/chainreactors/malice-network/proto/listener/lispb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"time"
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
func (w *Website) ToProtobuf(webContentDir string) *lispb.Website {
	WebContents := map[string]*lispb.WebContent{}
	for _, webcontent := range w.WebContents {
		contents, _ := os.ReadFile(filepath.Join(webContentDir, webcontent.Path))
		WebContents[webcontent.ID.String()] = webcontent.ToProtobuf(&contents)
	}
	return &lispb.Website{
		ID:       w.ID.String(),
		Name:     w.Name,
		Contents: WebContents,
	}
}

// WebContent - One piece of content mapped to a path
type WebContent struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	WebsiteID uuid.UUID `gorm:"type:uuid;"`
	Website   Website   `gorm:"foreignKey:WebsiteID;"`

	Path        string `gorm:"primaryKey"`
	Size        uint64
	ContentType string
}

// BeforeCreate - GORM hook to automatically set values
func (wc *WebContent) BeforeCreate(tx *gorm.DB) (err error) {
	wc.ID, err = uuid.NewV4()
	return err
}

// ToProtobuf - Converts to protobuf object
func (wc *WebContent) ToProtobuf(content *[]byte) *lispb.WebContent {
	return &lispb.WebContent{
		ID:          wc.ID.String(),
		WebsiteID:   wc.WebsiteID.String(),
		Path:        wc.Path,
		Size:        uint64(wc.Size),
		ContentType: wc.ContentType,
		Content:     *content,
	}
}

func WebContentFromProtobuf(pbWebContent *lispb.WebContent) WebContent {
	siteUUID, _ := uuid.FromString(pbWebContent.ID)
	websiteUUID, _ := uuid.FromString(pbWebContent.WebsiteID)

	return WebContent{
		ID:          siteUUID,
		WebsiteID:   websiteUUID,
		Path:        pbWebContent.Path,
		Size:        pbWebContent.Size,
		ContentType: pbWebContent.ContentType,
	}
}
