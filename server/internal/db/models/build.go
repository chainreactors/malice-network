package models

import (
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"gorm.io/gorm"
	"strings"
	"time"
)

type Builder struct {
	ID          uint32 `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"unique"`
	ProfileName string `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ProfileName;references:Name"` // 将 ProfileName 设置为外键

	RealName   string
	CreatedAt  time.Time `gorm:"->;<-:create;"`
	Target     string    // build target, like win64, win32, linux64
	Type       string    // build type, pe, dll, shellcode
	Stager     string    // shellcode prelude beacon bind
	Modules    string    // default modules, comma split, e.g. "execute_exe,execute_dll"
	ParamsJson string
	CA         string // ca file , ca file content
	Path       string
	Profile    Profile `gorm:"foreignKey:ProfileName;references:Name;"`
}

func (b *Builder) BeforeCreate(tx *gorm.DB) (err error) {
	b.CreatedAt = time.Now()
	return nil
}

func (b *Builder) ToProtobuf() *clientpb.Generate {
	if b.ProfileName != "" {
		return &clientpb.Generate{
			Name:    b.Name,
			Target:  b.Profile.Target,
			Type:    b.Profile.Type,
			Stager:  b.Profile.Stager,
			Modules: strings.Split(b.Profile.Modules, ","),
		}
	}

	return &clientpb.Generate{
		Name:    b.Name,
		Target:  b.Target,
		Type:    b.Type,
		Stager:  b.Stager,
		Modules: strings.Split(b.Modules, ","),
	}
}

func (b *Builder) FromProtobuf(pb *clientpb.Generate) {
	b.Name = pb.Name
	b.Target = pb.Target
	b.Type = pb.Type
	b.Stager = pb.Stager
	b.Modules = strings.Join(pb.Modules, ",")
}
