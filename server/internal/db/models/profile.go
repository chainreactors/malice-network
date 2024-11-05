package models

import (
	"encoding/json"
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
	// pe,dll,shellcode,elf
	Type string

	// shellcode prelude beacon bind
	Stager string

	Proxy     string // not impl
	Obfuscate string // not impl, obf llvm plug ,

	Modules string // default modules, comma split, e.g. "execute_exe,execute_dll"
	CA      string // ca file , ca file content

	// params
	//interval int    // default 10
	//jitter   int    // default 5
	Params     map[string]interface{} `gorm:"-"`         // Ignored by GORM
	ParamsJson string                 `gorm:"type:text"` // Used for storing serialized params

	PipelineID    string `gorm:"type:string;index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	implantConfig string // raw implant config

	Pipeline Pipeline `gorm:"foreignKey:PipelineID;references:Name;"`

	CreatedAt time.Time `gorm:"->;<-:create;"`
}

type Params struct {
	Interval string
	Jitter   string
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

// Serialize implantConfig (raw implant config) to JSON
func (p *Profile) SerializeImplantConfig(config interface{}) error {
	configJson, err := json.Marshal(config)
	if err != nil {
		return err
	}
	p.implantConfig = string(configJson)
	return nil
}

// Deserialize implantConfig (JSON string) to a struct or map
func (p *Profile) DeserializeImplantConfig(config interface{}) error {
	if p.implantConfig != "" {
		err := json.Unmarshal([]byte(p.implantConfig), config)
		if err != nil {
			return err
		}
	}
	return nil
}
