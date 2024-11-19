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

	CreatedAt  time.Time `gorm:"->;<-:create;"`
	Target     string    // build target, like win64, win32, linux64
	Type       string    // build type, pe, dll, shellcode
	Stager     string    // shellcode prelude beacon bind
	Modules    string    // default modules, comma split, e.g. "execute_exe,execute_dll"
	ParamsJson string
	CA         string // ca file , ca file content
	Path       string
	Profile    Profile `gorm:"foreignKey:ProfileName;references:Name;"`
	Os         string
	Arch       string
}

func (b *Builder) BeforeCreate(tx *gorm.DB) (err error) {
	b.CreatedAt = time.Now()
	return nil
}

func (b *Builder) ToProtobuf(bin []byte) *clientpb.Builder {
	if b.ProfileName != "" {
		return &clientpb.Builder{
			Bin:         bin,
			Name:        b.Name,
			Target:      b.Target,
			Type:        b.Type,
			Stage:       b.Stager,
			Platform:    b.Os,
			Arch:        b.Arch,
			Modules:     b.Modules,
			ProfileName: b.ProfileName,
			PipelineId:  b.Profile.PipelineID,
		}
	}

	return &clientpb.Builder{
		Bin:         bin,
		Name:        b.Name,
		Target:      b.Target,
		Type:        b.Type,
		Stage:       b.Stager,
		Platform:    b.Os,
		Modules:     b.Modules,
		Arch:        b.Arch,
		ProfileName: "",
		PipelineId:  "",
	}
}

func (b *Builder) FromProtobuf(pb *clientpb.Generate) {
	b.Name = pb.Name
	b.Target = pb.Target
	b.Type = pb.Type
	b.Stager = pb.Stager
	b.Modules = strings.Join(pb.Modules, ",")
}
