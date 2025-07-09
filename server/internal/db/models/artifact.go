package models

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"gorm.io/gorm"
	"os"
	"time"
)

type Artifact struct {
	ID          uint32 `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"unique"`
	ProfileName string `gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ProfileName;references:Name"`

	CreatedAt time.Time `gorm:"->;<-:create;"`
	Target    string    // build target, like win64, win32, linux64
	Type      string    // build type, pe, dll, shellcode
	Source    string    // resource file
	//CA            string // ca file , ca file content
	Path        string
	Profile     Profile `gorm:"foreignKey:ProfileName;references:Name;"`
	Os          string
	Arch        string
	Log         string
	Status      string
	ParamsData  string
	Params      *types.ProfileParams `gorm:"-"`
	ProfileByte []byte
}

func (a *Artifact) AfterFind(tx *gorm.DB) (err error) {
	if a.ParamsData == "" {
		return nil
	}

	// 如果知道具体类型，可以直接反序列化
	var params types.ProfileParams
	if err := json.Unmarshal([]byte(a.ParamsData), &params); err != nil {
		return err
	}
	a.Params = &params
	return nil
}

// BeforeSave GORM 钩子 - 保存前将 Params 序列化
func (a *Artifact) BeforeSave(tx *gorm.DB) error {
	if a.Params != nil {
		data, err := json.Marshal(a.Params)
		if err != nil {
			return err
		}
		a.ParamsData = string(data)
	}
	return nil
}

func (a *Artifact) BeforeCreate(tx *gorm.DB) (err error) {
	a.CreatedAt = time.Now()
	return nil
}

func (a *Artifact) ToProtobuf(bin []byte) *clientpb.Artifact {
	return &clientpb.Artifact{
		Id:           a.ID,
		Bin:          bin,
		Name:         a.Name,
		Target:       a.Target,
		Type:         a.Type,
		Platform:     a.Os,
		Arch:         a.Arch,
		Profile:      a.ProfileName,
		Pipeline:     a.Profile.PipelineID,
		CreatedAt:    a.CreatedAt.Unix(),
		Status:       a.Status,
		ProfileBytes: a.ProfileByte,
		ParamsBytes:  []byte(a.ParamsData),
		Source:       a.Source,
	}
}

func (a *Artifact) ToArtifact() (*clientpb.Artifact, error) {
	bin, err := os.ReadFile(a.Path)
	if err != nil {
		return nil, err
	}

	return &clientpb.Artifact{
		Id:           a.ID,
		Bin:          bin,
		Name:         a.Name,
		Target:       a.Target,
		Type:         a.Type,
		Platform:     a.Os,
		Arch:         a.Arch,
		Profile:      a.ProfileName,
		Pipeline:     a.Profile.PipelineID,
		CreatedAt:    a.CreatedAt.Unix(),
		Status:       a.Status,
		ProfileBytes: a.ProfileByte,
		ParamsBytes:  []byte(a.ParamsData),
		Source:       a.Source,
	}, nil
}
