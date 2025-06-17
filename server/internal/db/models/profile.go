package models

import (
	"encoding/json"
	"time"

	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type Profile struct {
	ID uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`

	// build
	Name string `gorm:"unique"` // Ensuring Name is unique

	// build type
	Type string

	Modules string // default modules, comma split, e.g. "execute_exe,execute_dll"
	CA      string // ca file , ca file content
	Raw     []byte
	// params
	Params     *types.ProfileParams `gorm:"-"`             // 使用 interface{} 使其更灵活
	ParamsData string               `gorm:"column:params"` // 改用更简洁的数据库字段名

	// BasicPipeline 和 PulsePipeline
	PipelineID      string `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PulsePipelineID string `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	implantConfig string // raw implant config

	// BasicPipeline 和 PulsePipeline
	Pipeline      *Pipeline `gorm:"foreignKey:PipelineID;references:Name;"`
	PulsePipeline *Pipeline `gorm:"foreignKey:PulsePipelineID;references:Name;"`

	CreatedAt time.Time `gorm:"->;<-:create;"`
}

func (p *Profile) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
	p.CreatedAt = time.Now()
	return nil
}

func (p *Profile) AfterFind(tx *gorm.DB) (err error) {
	if p.ParamsData == "" {
		return nil
	}

	// 如果知道具体类型，可以直接反序列化
	var params types.ProfileParams
	if err := json.Unmarshal([]byte(p.ParamsData), &params); err != nil {
		return err
	}
	p.Params = &params
	return nil
}

// Deserialize implantConfig (JSON string) to a struct or map
func (p *Profile) DeserializeImplantConfig() error {
	if p.ParamsData != "" {
		err := json.Unmarshal([]byte(p.ParamsData), &p.Params)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Profile) ToProtobuf() *clientpb.Profile {
	return &clientpb.Profile{
		Name:            p.Name,
		Type:            p.Type,
		Modules:         p.Modules,
		Ca:              p.CA,
		PipelineId:      p.PipelineID,
		PulsePipelineId: p.PulsePipelineID,
		Content:         p.Raw,
		Params:          p.ParamsData,
	}
}

// BeforeSave GORM 钩子 - 保存前将 Params 序列化
func (p *Profile) BeforeSave(tx *gorm.DB) error {
	if p.Params != nil {
		data, err := json.Marshal(p.Params)
		if err != nil {
			return err
		}
		p.ParamsData = string(data)
	}
	return nil
}
