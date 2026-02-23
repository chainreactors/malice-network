package models

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"path/filepath"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
)

type Profile struct {
	ID uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`

	// build
	Name string `gorm:"unique"` // Ensuring Name is unique

	// params
	Params     *implanttypes.ProfileParams `gorm:"-"`             // 使用 interface{} 使其更灵活
	ParamsData string                      `gorm:"column:params"` // 改用更简洁的数据库字段名

	// BasicPipeline 和 PulsePipeline
	PipelineID string `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`

	// BasicPipeline 和 PulsePipeline
	Pipeline *Pipeline `gorm:"foreignKey:PipelineID;references:Name;"`

	CreatedAt time.Time `gorm:"->;<-:create;"`
}

func (p *Profile) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == (uuid.UUID{}) {
		p.ID, err = uuid.NewV4()
		if err != nil {
			return err
		}
	}
	p.CreatedAt = time.Now()
	return nil
}

func (p *Profile) AfterFind(tx *gorm.DB) (err error) {
	if p.ParamsData == "" {
		return nil
	}

	// 如果知道具体类型，可以直接反序列化
	var params implanttypes.ProfileParams
	if err := json.Unmarshal([]byte(p.ParamsData), &params); err != nil {
		return err
	}
	p.Params = &params
	return nil
}

// DiskPath 返回该 profile 在磁盘上的存储目录
func (p *Profile) DiskPath() string {
	return filepath.Join(configs.ProfilePath, p.ID.String())
}

func (p *Profile) ToProtobuf() *clientpb.Profile {
	return &clientpb.Profile{
		Name:       p.Name,
		PipelineId: p.PipelineID,
		Params:     p.ParamsData,
		CreatedAt:  p.CreatedAt.Unix(),
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
