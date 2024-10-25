package models

import (
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gofrs/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
	"strconv"
	"time"
)

type Profile struct {
	ID uuid.UUID `gorm:"primaryKey;->;<-:create;type:uuid;"`

	// build
	Name   string // Ensuring Name is unique
	Target string // build target win64,win32,linux64

	// build type
	// pe, dll, module dll, stage0, stage1, shellcode
	Type string

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

	Pipeline Pipeline `gorm:"foreignKey:Name;references:Name;"`

	CreatedAt time.Time `gorm:"->;<-:create;"`
}

type params struct {
	interval string
	jitter   string
}

func (p *Profile) BeforeCreate(tx *gorm.DB) (err error) {
	p.ID, err = uuid.NewV4()
	if err != nil {
		return err
	}
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

// UpdateGeneratorConfig
func (p *Profile) UpdateGeneratorConfig(defaultConfig configs.GeneratorConfig, req *clientpb.Generate, path string) error {

	if p.Name != "" {
		defaultConfig.Server.Name = p.Name
	}

	if req.Url != "" {
		defaultConfig.Server.Urls = []string{}
		defaultConfig.Server.Urls = append(defaultConfig.Server.Urls, req.Url)
	} else if p.Name != "" {
		defaultConfig.Server.Urls = []string{}
		defaultConfig.Server.Urls = append(defaultConfig.Server.Urls,
			fmt.Sprintf("%s:%v", p.Pipeline.Host, p.Pipeline.Port))
	}
	var dbParams *params
	err := p.DeserializeImplantConfig(dbParams)
	if err != nil {
		return err
	}
	if val, ok := req.Params["interval"]; ok {
		interval, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		defaultConfig.Server.Interval = interval
	} else if p.Name != "" {
		dbInterval, err := strconv.Atoi(dbParams.interval)
		if err != nil {
			return err
		}
		defaultConfig.Server.Interval = dbInterval
	}

	if val, ok := req.Params["jitter"]; ok {
		jitter, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		defaultConfig.Server.Jitter = jitter
	} else if p.Name != "" {
		dbJitter, err := strconv.Atoi(dbParams.jitter)
		if err != nil {
			return err
		}
		defaultConfig.Server.Jitter = dbJitter
	}

	if val, ok := req.Params["ca"]; ok {
		defaultConfig.Server.CA = val
	} else if p.Pipeline.Tls.Enable {
		defaultConfig.Server.CA = p.Pipeline.Tls.Cert
	}

	//dbModules := strings.Split(profile.Modules, ",")
	//
	//if len(dbModules) > 0 {
	//	defaultConfig.Implants.Modules = []string{}
	//	defaultConfig.Implants.Modules = append(defaultConfig.Implants.Modules, dbModules...)
	//}

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
