package models

import (
	"os"
	"path/filepath"
	"time"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

// WebsiteContent - Single table that combines Website and WebContent
type WebsiteContent struct {
	ID        uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt time.Time `gorm:"->;<-:create;"`

	File        string            `gorm:""`
	Path        string            `gorm:""`
	Size        uint64            `gorm:""`
	Type        string            `gorm:""`
	ContentType string            `gorm:""`
	Encryption  *EncryptionConfig `gorm:"embedded;embeddedPrefix:encryption_"`

	Pipeline   *Pipeline `gorm:"foreignKey:PipelineID;references:Name;"`
	PipelineID string    `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
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
func (wc *WebsiteContent) ToProtobuf(read bool) *clientpb.WebContent {
	var data []byte
	if read && wc.Type == "raw" {
		data, _ = os.ReadFile(filepath.Join(configs.WebsitePath, wc.PipelineID, wc.ID.String()))
	}

	return &clientpb.WebContent{
		Id:          wc.ID.String(),
		WebsiteId:   wc.PipelineID,
		Path:        wc.Path,
		Size:        wc.Size,
		Type:        wc.Type,
		ContentType: wc.ContentType,
		Content:     data,
		Encryption:  ToEncryptionProtobuf(wc.Encryption),
		ListenerId:  wc.Pipeline.ListenerID,
	}
}

func FromWebContentPb(content *clientpb.WebContent) *WebsiteContent {
	return &WebsiteContent{
		PipelineID:  content.WebsiteId,
		File:        content.File,
		Path:        content.Path,
		Size:        content.Size,
		Type:        content.Type,
		ContentType: content.ContentType,
		Encryption:  ToEncryptionDB(content.Encryption),
	}
}
