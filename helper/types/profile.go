package types

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"gopkg.in/yaml.v3"
	"path/filepath"
)

func LoadProfile(content []byte) (*ProfileConfig, error) {
	if content == nil {
		content = consts.DefaultProfile
	}
	profile := &ProfileConfig{}
	err := yaml.Unmarshal(content, profile)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

type BasicProfile struct {
	Name       string                 `yaml:"name" config:"name" default:"malefic"`
	Targets    []string               `yaml:"targets" config:"targets" default:"[]"`
	Protocol   string                 `yaml:"protocol" config:"protocol" default:"tcp"`
	TLS        *TLSProfile            `yaml:"tls" config:"tls"`
	Proxy      string                 `yaml:"proxy" config:"proxy" default:""`
	Interval   int                    `yaml:"interval" config:"interval" default:"5"`
	Jitter     float64                `yaml:"jitter" config:"jitter" default:"0.2"`
	Encryption string                 `yaml:"encryption" config:"encryption" default:"aes"`
	Key        string                 `yaml:"key" config:"key" default:"maliceofinternal"`
	REM        *REMProfile            `yaml:"rem" config:"rem"`
	Http       *HttpProfile           `yaml:"http" config:"http"`
	Extras     map[string]interface{} `yaml:",inline"`
}

type REMProfile struct {
	Link string `yaml:"link" config:"link" default:""`
}

type TLSProfile struct {
	Enable  bool                   `yaml:"enable" config:"enable" default:"false"`
	Version string                 `yaml:"version" config:"version" default:"auto"`
	SNI     string                 `yaml:"sni" config:"sni" default:"localhost"`
	Extras  map[string]interface{} `yaml:",inline"`
}

type HttpProfile struct {
	Method  string                 `yaml:"method" config:"method" default:"POST"`
	Path    string                 `yaml:"path" config:"path" default:"/jquery.js"`
	Host    string                 `yaml:"host" config:"host" default:"127.0.0.1"`
	Version string                 `yaml:"version" config:"version" default:"1.1"`
	Headers map[string]string      `yaml:"headers" config:"headers"`
	Extras  map[string]interface{} `yaml:",inline"`
}

type PulseProfile struct {
	Target     string `yaml:"target"`
	Encryption string `yaml:"encryption"`
	Key        string `yaml:"key"`
	Protocol   string `yaml:"protocol"`
	Flags      struct {
		ArtifactID uint32                 `yaml:"artifact_id" config:"artifact_id" default:"0"`
		Extras     map[string]interface{} `yaml:",inline"`
	}
	Http   *HttpProfile           `yaml:"http" config:"http"`
	Extras map[string]interface{} `yaml:",inline"`
}

type PackItem struct {
	Src string `yaml:"src" config:"src"`
	Dst string `yaml:"dst" config:"dst"`
}

type ImplantProfile struct {
	Mod          string                 `yaml:"mod" config:"mod" default:""`
	RegisterInfo bool                   `yaml:"register_info" config:"register_info" default:"false"`
	HotLoad      bool                   `yaml:"hot_load" config:"hot_load" default:"false"`
	Modules      []string               `yaml:"modules" config:"modules" default:"[]"`
	Pack         []PackItem             `yaml:"pack" config:"pack"`
	AutoRun      string                 `yaml:"autorun" config:"autorun"`
	Enable3rd    bool                   `yaml:"enable_3rd" config:"enable_3rd"`
	ThirdModules []string               `yaml:"3rd_modules" config:"3rd_modules"`
	Extras       map[string]interface{} `yaml:",inline"`
}

type MetadataProfile struct {
	Icon   string                 `yaml:"icon" config:"icon" default:""`
	Extras map[string]interface{} `yaml:",inline"`
}

type BuildProfile struct {
	Metadata *MetadataProfile       `yaml:"metadata" config:"metadata"`
	Extras   map[string]interface{} `yaml:",inline"`
}

type ProfileConfig struct {
	Basic   *BasicProfile          `yaml:"basic" config:"basic"`
	Pulse   *PulseProfile          `yaml:"pulse" config:"pulse"`
	Implant *ImplantProfile        `yaml:"implants" config:"implants"`
	Build   *BuildProfile          `yaml:"build" config:"build"`
	Extras  map[string]interface{} `yaml:",inline"`
}

type ProfileParams struct {
	Interval int     `json:"interval"`
	Jitter   float64 `json:"jitter"`
	//Obfuscation string `json:"obfuscation"` // not impl, obf llvm plug ,

	Address        string `json:"address"`
	Proxy          string `json:"proxy"`
	OriginBeaconID uint32 `json:"origin_beacon_id"`
	RelinkBeaconID uint32 `json:"relink_beacon_id"`
	REMPipeline    string `json:"rem"`
	Enable3RD      bool   `json:"enable_3_rd"`
	Modules        string `json:"modules"`
	AutoDownload   bool   `json:"auto_download"`

	AutoRunFile string `json:"auto_run_file"`
}

func (p *ProfileParams) String() string {
	content, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(content)
}

func UnmarshalProfileParams(params []byte) (*ProfileParams, error) {
	var p *ProfileParams
	err := json.Unmarshal(params, &p)
	if err != nil {
		return p, err
	}
	return p, nil
}

// ValidateProfileFiles 验证 profile 中引用的文件是否存在于指定目录中
func (p *ProfileConfig) ValidateProfileFiles(baseDir string) error {

	if p.Build != nil && p.Build.Metadata != nil && p.Build.Metadata.Icon != "" {
		iconPath := filepath.Join(baseDir, p.Build.Metadata.Icon)
		if !fileutils.Exist(iconPath) {
			return fmt.Errorf("icon file not found: %s", p.Build.Metadata.Icon)
		}
	}

	if p.Implant != nil && len(p.Implant.Pack) > 0 {
		for i, packItem := range p.Implant.Pack {
			if packItem.Src != "" {
				srcPath := filepath.Join(baseDir, packItem.Src)
				if !fileutils.Exist(srcPath) {
					return fmt.Errorf("pack source file not found: %s (pack item %d)", packItem.Src, i)
				}
			}
		}
	}

	return nil
}
