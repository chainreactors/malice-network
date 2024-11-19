package models

import (
	"encoding/json"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/gofrs/uuid"
	"gorm.io/gorm"
	"time"
)

type Profile struct {
	ID uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`

	// build
	Name   string `gorm:"unique"` // Ensuring Name is unique
	Target string // build target win64,win32,linux64

	// build type
	Type string

	// shellcode prelude beacon bind
	Stager string

	Proxy     string // not impl
	Obfuscate string // not impl, obf llvm plug ,

	Modules string // default modules, comma split, e.g. "execute_exe,execute_dll"
	CA      string // ca file , ca file content
	Raw     []byte
	// params
	Params     *types.ProfileParams `gorm:"-"`         // Ignored by GORM
	ParamsJson string               `gorm:"type:text"` // Used for storing serialized params

	PipelineID    string `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	implantConfig string // raw implant config

	Pipeline *Pipeline `gorm:"foreignKey:PipelineID;references:Name;"`

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
	if p.ParamsJson == "" {
		return nil
	}
	err = json.Unmarshal([]byte(p.ParamsJson), &p.Params)
	if err != nil {
		return err
	}
	return nil
}

// Deserialize implantConfig (JSON string) to a struct or map
func (p *Profile) DeserializeImplantConfig() error {
	var params *types.ProfileParams
	if p.implantConfig != "" {
		err := json.Unmarshal([]byte(p.implantConfig), params)
		if err != nil {
			return err
		}
	}
	p.Params = params
	return nil
}

func (p *Profile) ToProtobuf() *clientpb.Profile {
	return &clientpb.Profile{
		Name:       p.Name,
		Target:     p.Target,
		Type:       p.Type,
		Modules:    p.Modules,
		Ca:         p.CA,
		PipelineId: p.PipelineID,
		Content:    p.Raw,
		Params:     p.ParamsJson,
	}
}
