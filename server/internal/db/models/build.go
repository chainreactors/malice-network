package models

import (
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"gorm.io/gorm"
	"sort"
	"strings"
	"time"
)

type Builder struct {
	ID          uint32 `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"unique"`
	ProfileName string `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ProfileName;references:Name"` // 将 ProfileName 设置为外键

	CreatedAt     time.Time `gorm:"->;<-:create;"`
	Target        string    // build target, like win64, win32, linux64
	Type          string    // build type, pe, dll, shellcode
	Stager        string    // shellcode prelude beacon bind
	Modules       string    // default modules, comma split, e.g. "execute_exe,execute_dll"
	Source        string    // resource file
	ParamsJson    string
	CA            string // ca file , ca file content
	Path          string
	IsSRDI        bool
	ShellcodePath string
	Profile       Profile `gorm:"foreignKey:ProfileName;references:Name;"`
	Os            string
	Arch          string
	Log           string
}

func (b *Builder) BeforeCreate(tx *gorm.DB) (err error) {
	b.CreatedAt = time.Now()
	moduleList := strings.Split(b.Modules, ",")
	sort.Strings(moduleList)
	b.Modules = strings.Join(moduleList, ",")
	return nil
}

func (b *Builder) ToArtifact(bin []byte) *clientpb.Artifact {
	var pipeline string
	if b.Type == consts.CommandBuildPulse {
		pipeline = b.Profile.PulsePipelineID
	} else {
		pipeline = b.Profile.PipelineID
	}
	return &clientpb.Artifact{
		Id:       b.ID,
		Bin:      bin,
		Name:     b.Name,
		Target:   b.Target,
		Type:     b.Type,
		Stage:    b.Stager,
		Platform: b.Os,
		Arch:     b.Arch,
		IsSrdi:   b.IsSRDI,
		Profile:  b.ProfileName,
		Pipeline: pipeline,
		Time:     b.CreatedAt.Unix(),
	}
}

func (b *Builder) ToProtobuf() *clientpb.Builder {
	var pipeline string
	if b.Type == consts.CommandBuildPulse {
		pipeline = b.Profile.PulsePipelineID
	} else {
		pipeline = b.Profile.PipelineID
	}
	return &clientpb.Builder{
		Id:          b.ID,
		Name:        b.Name,
		Target:      b.Target,
		Type:        b.Type,
		Stage:       b.Stager,
		Platform:    b.Os,
		Arch:        b.Arch,
		Modules:     b.Modules,
		IsSrdi:      b.IsSRDI,
		ProfileName: b.ProfileName,
		Pipeline:    pipeline,
		Time:        b.CreatedAt.Unix(),
		Resource:    b.Source,
	}
}

func (b *Builder) FromProtobuf(pb *clientpb.Generate) {
	b.Name = pb.Name
	b.Target = pb.Target
	b.Type = pb.Type
	b.Stager = pb.Stager
	b.Modules = strings.Join(pb.Modules, ",")
}
