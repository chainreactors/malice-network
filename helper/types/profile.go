package types

import (
	_ "embed"
	"encoding/json"
	"gopkg.in/yaml.v3"
)

//go:embed profile.yaml
var DefaultProfile []byte

func LoadProfile(content []byte) (*ProfileConfig, error) {
	if content == nil {
		content = DefaultProfile
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
	Extras     map[string]interface{} `yaml:",inline"`
}

type TLSProfile struct {
	Enable  bool                   `yaml:"enable" config:"enable" default:"false"`
	Version string                 `yaml:"version" config:"version" default:"auto"`
	SNI     string                 `yaml:"sni" config:"sni" default:"localhost"`
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
	Extras map[string]interface{} `yaml:",inline"`
}

type ImplantProfile struct {
	Mod          string                 `yaml:"mod" config:"mod" default:""`
	RegisterInfo bool                   `yaml:"register_info" config:"register_info" default:"false"`
	HotLoad      bool                   `yaml:"hot_load" config:"hot_load" default:"false"`
	Modules      []string               `yaml:"modules" config:"modules" default:"[]"`
	Extras       map[string]interface{} `yaml:",inline"`
}

type ProfileConfig struct {
	Basic   *BasicProfile          `yaml:"basic" config:"basic"`
	Pulse   *PulseProfile          `yaml:"pulse" config:"pulse"`
	Implant *ImplantProfile        `yaml:"implants" config:"implants"`
	Extras  map[string]interface{} `yaml:",inline"`
}

type ProfileParams struct {
	Interval int     `json:"interval"`
	Jitter   float64 `json:"jitter"`
	// shellcode prelude beacon bind
	Obfuscation string `json:"obfuscation"` // not impl, obf llvm plug ,

	Proxy          string `json:"proxy"`
	OriginBeaconID uint32 `json:"origin_beacon_id"`
	RelinkBeaconID uint32 `json:"relink_beacon_id"`

	Module3rd bool   `json:"module_3rd"`
	Modules   string `json:"modules"`
}

func (p *ProfileParams) String() string {
	content, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(content)
}

func Unmarsal(params string) (*ProfileParams, error) {
	var p *ProfileParams
	err := json.Unmarshal([]byte(params), &p)
	if err != nil {
		return p, err
	}
	return p, nil
}
