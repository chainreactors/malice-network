package models

import (
	"gorm.io/gorm"
	"time"
)

type Profile struct {
	ID            string    `gorm:"primaryKey;->;<-:create;type:uuid;"`
	CreatedAt     time.Time `gorm:"->;<-:create;"`
	Name          string
	ServerName    string
	Proxy         string
	Interval      int
	Jitter        int
	ListenerID    string `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PipelineID    string `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	implantConfig string

	Pipeline Pipeline `gorm:"foreignKey:ListenerID,Name;references:ListenerID,Name;"`
}

func (p *Profile) BeforeCreate(tx *gorm.DB) (err error) {
	if err != nil {
		return err
	}
	p.CreatedAt = time.Now()
	return nil
}
