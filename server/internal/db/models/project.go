package models

import (
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type Project struct {
	ID          uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`
	Name        string    `gorm:"uniqueIndex;not null"`
	Description string
	Note        string
	IsDeleted   bool      `gorm:"default:false"`
	CreatedAt   time.Time `gorm:"->;<-:create;"`
	UpdatedAt   time.Time
}

func (p *Project) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == (uuid.UUID{}) {
		p.ID, err = uuid.NewV4()
		if err != nil {
			return err
		}
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	return nil
}

func (p *Project) ToProtobuf() *clientpb.Project {
	return &clientpb.Project{
		Id:          p.ID.String(),
		Name:        p.Name,
		Description: p.Description,
		Note:        p.Note,
		IsDeleted:   p.IsDeleted,
		CreatedAt:   p.CreatedAt.Unix(),
		UpdatedAt:   p.UpdatedAt.Unix(),
	}
}
