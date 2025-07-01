package models

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"gorm.io/gorm"
	"sort"
	"strings"
	"time"
)

type Builder struct {
	ID          uint32 `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"unique"`
	ProfileName string `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ProfileName;references:Name"`

	CreatedAt time.Time `gorm:"->;<-:create;"`
	Target    string    // build target, like win64, win32, linux64
	Type      string    // build type, pe, dll, shellcode
	Stager    string    // shellcode prelude beacon bind
	Modules   string    // default modules, comma split, e.g. "execute_exe,execute_dll"
	Source    string    // resource file
	//CA            string // ca file , ca file content
	Path          string
	IsSRDI        bool
	ShellcodePath string
	Profile       Profile `gorm:"foreignKey:ProfileName;references:Name;"`
	Os            string
	Arch          string
	Log           string
	Status        string
	ParamsData    string
	Params        *types.ProfileParams `gorm:"-"`
	ProfileByte   []byte
}

func (b *Builder) AfterFind(tx *gorm.DB) (err error) {
	if b.ParamsData == "" {
		return nil
	}

	// 如果知道具体类型，可以直接反序列化
	var params types.ProfileParams
	if err := json.Unmarshal([]byte(b.ParamsData), &params); err != nil {
		return err
	}
	b.Params = &params
	return nil
}

// BeforeSave GORM 钩子 - 保存前将 Params 序列化
func (b *Builder) BeforeSave(tx *gorm.DB) error {
	if b.Params != nil {
		data, err := json.Marshal(b.Params)
		if err != nil {
			return err
		}
		b.ParamsData = string(data)
	}
	return nil
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
		Id:           b.ID,
		Bin:          bin,
		Name:         b.Name,
		Target:       b.Target,
		Type:         b.Type,
		Stage:        b.Stager,
		Platform:     b.Os,
		Arch:         b.Arch,
		IsSrdi:       b.IsSRDI,
		Profile:      b.ProfileName,
		Pipeline:     pipeline,
		CreatedAt:    b.CreatedAt.Unix(),
		Status:       b.Status,
		ProfileBytes: b.ProfileByte,
		ParamsBytes:  []byte(b.ParamsData),
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
		Id:           b.ID,
		Name:         b.Name,
		Target:       b.Target,
		Type:         b.Type,
		Stage:        b.Stager,
		Platform:     b.Os,
		Arch:         b.Arch,
		Modules:      b.Modules,
		IsSrdi:       b.IsSRDI,
		ProfileName:  b.ProfileName,
		Pipeline:     pipeline,
		CreatedAt:    b.CreatedAt.Unix(),
		Source:       b.Source,
		Status:       b.Status,
		ProfileBytes: b.ProfileByte,
		ParamsBytes:  []byte(b.ParamsData),
	}
}
