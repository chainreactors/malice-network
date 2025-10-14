package types

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"gopkg.in/yaml.v3"
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

type ProxyProfile struct {
	UseEnvProxy bool   `yaml:"use_env_proxy" json:"use_env_proxy"`
	URL         string `yaml:"url" json:"url"`
}

type SecureProfile struct {
	Enable            bool   `yaml:"enable" json:"enable"`
	ImplantPrivateKey string `yaml:"private_key" json:"private_key"`
	ServerPublicKey   string `yaml:"public_key" json:"public_key"`
}

type DGAProfile struct {
	Enable        bool   `yaml:"enable" json:"enable"`
	Key           string `yaml:"key" json:"key"`
	IntervalHours int    `yaml:"interval_hours" json:"interval_hours"`
}

type GuardrailProfile struct {
	Enable      bool     `yaml:"enable" json:"enable"`
	RequireAll  bool     `yaml:"require_all" json:"require_all"`
	IPAddresses []string `yaml:"ip_addresses" json:"ip_addresses"`
	Usernames   []string `yaml:"usernames" json:"usernames"`
	ServerNames []string `yaml:"server_names" json:"server_names"`
	Domains     []string `yaml:"domains" json:"domains"`
}

type Target struct {
	Address      string       `yaml:"address" json:"address"`
	DomainSuffix string       `yaml:"domain_suffix,omitempty" json:"domain_suffix,omitempty"`
	Http         *HttpProfile `yaml:"http,omitempty" json:"http,omitempty"`
	TLS          *TLSProfile  `yaml:"tls,omitempty" json:"tls,omitempty"`
	TCP          *TCPProfile  `yaml:"tcp,omitempty" json:"tcp,omitempty"`
	REM          *REMProfile  `yaml:"rem,omitempty" json:"rem,omitempty"`
}

type TCPProfile struct {
	// TCP specific configurations can be added here
}

type BasicProfile struct {
	Name        string                 `yaml:"name" json:"name"`
	Proxy       *ProxyProfile          `yaml:"proxy" json:"proxy"`
	Cron        string                 `yaml:"cron" json:"cron"`
	Jitter      float64                `yaml:"jitter" json:"jitter"`
	InitRetry   int                    `yaml:"init_retry" json:"init_retry"`
	ServerRetry int                    `yaml:"server_retry" json:"server_retry"`
	GlobalRetry int                    `yaml:"global_retry" json:"global_retry"`
	Encryption  string                 `yaml:"encryption" json:"encryption"`
	Key         string                 `yaml:"key" json:"key"`
	Secure      *SecureProfile         `yaml:"secure" json:"secure"`
	DGA         *DGAProfile            `yaml:"dga" json:"dga"`
	Guardrail   *GuardrailProfile      `yaml:"guardrail" json:"guardrail"`
	Targets     []Target               `yaml:"targets" json:"targets"`
	Extras      map[string]interface{} `yaml:",inline" json:",inline"`
}

type REMProfile struct {
	Link string `yaml:"link" json:"link"`
}

type TLSProfile struct {
	Enable           bool                   `yaml:"enable" json:"enable"`
	SNI              string                 `yaml:"sni" json:"sni"`
	SkipVerification bool                   `yaml:"skip_verification" json:"skip_verification"`
	Extras           map[string]interface{} `yaml:",inline" json:",inline"`
}

type HttpProfile struct {
	Method  string                 `yaml:"method" json:"method"`
	Path    string                 `yaml:"path" json:"path"`
	Host    string                 `yaml:"host" json:"host"`
	Version string                 `yaml:"version" json:"version"`
	Headers map[string]string      `yaml:"headers" json:"headers"`
	Extras  map[string]interface{} `yaml:",inline" json:",inline"`
}

type PulseFlags struct {
	Start      uint32                 `yaml:"start" json:"start"`
	End        uint32                 `yaml:"end" json:"end"`
	Magic      string                 `yaml:"magic" json:"magic"`
	ArtifactID uint32                 `yaml:"artifact_id" json:"artifact_id"`
	Extras     map[string]interface{} `yaml:",inline" json:",inline"`
}

type PulseProfile struct {
	Flags      *PulseFlags            `yaml:"flags" json:"flags"`
	Encryption string                 `yaml:"encryption" json:"encryption"`
	Key        string                 `yaml:"key" json:"key"`
	Target     string                 `yaml:"target" json:"target"`
	Protocol   string                 `yaml:"protocol" json:"protocol"`
	Http       *HttpProfile           `yaml:"http" json:"http"`
	Extras     map[string]interface{} `yaml:",inline" json:",inline"`
}

type PackItem struct {
	Src string `yaml:"src" json:"src"`
	Dst string `yaml:"dst" json:"dst"`
}

type AntiProfile struct {
	Sandbox  bool `yaml:"sandbox" json:"sandbox"`
	VM       bool `yaml:"vm" json:"vm"`
	Debug    bool `yaml:"debug" json:"debug"`
	Disasm   bool `yaml:"disasm" json:"disasm"`
	Emulator bool `yaml:"emulator" json:"emulator"`
	Forensic bool `yaml:"forensic" json:"forensic"`
}

type APIPriorityConfig struct {
	Enable bool   `yaml:"enable" json:"enable"`
	Type   string `yaml:"type" json:"type"`
}

type APIsProfile struct {
	Level    string                        `yaml:"level" json:"level"`
	Priority map[string]*APIPriorityConfig `yaml:"priority" json:"priority"`
}

type AllocatorProfile struct {
	InProcess    string `yaml:"inprocess" json:"inprocess"`
	CrossProcess string `yaml:"crossprocess" json:"crossprocess"`
}

type ImplantFlags struct {
	Start      uint32                 `yaml:"start" json:"start"`
	End        uint32                 `yaml:"end" json:"end"`
	Magic      string                 `yaml:"magic" json:"magic"`
	ArtifactID uint32                 `yaml:"artifact_id" json:"artifact_id"`
	Extras     map[string]interface{} `yaml:",inline" json:",inline"`
}

type ImplantProfile struct {
	Runtime            string                 `yaml:"runtime" json:"runtime"`
	Mod                string                 `yaml:"mod" json:"mod"`
	RegisterInfo       bool                   `yaml:"register_info" json:"register_info"`
	HotLoad            bool                   `yaml:"hot_load" json:"hot_load"`
	Modules            []string               `yaml:"modules" json:"modules"`
	Enable3rd          bool                   `yaml:"enable_3rd" json:"enable_3rd"`
	ThirdModules       []string               `yaml:"3rd_modules" json:"3rd_modules"`
	Prelude            string                 `yaml:"prelude" json:"prelude"`
	Pack               []PackItem             `yaml:"pack" json:"pack"`
	Flags              *ImplantFlags          `yaml:"flags" json:"flags"`
	Anti               *AntiProfile           `yaml:"anti" json:"anti"`
	APIs               *APIsProfile           `yaml:"apis" json:"apis"`
	Allocator          *AllocatorProfile      `yaml:"allocator" json:"allocator"`
	ThreadStackSpoofer bool                   `yaml:"thread_stack_spoofer" json:"thread_stack_spoofer"`
	Extras             map[string]interface{} `yaml:",inline" json:",inline"`
}

type OLLVMProfile struct {
	Enable   bool `yaml:"enable" json:"enable"`
	BCFObf   bool `yaml:"bcfobf" json:"bcfobf"`
	SplitObf bool `yaml:"splitobf" json:"splitobf"`
	SubObf   bool `yaml:"subobf" json:"subobf"`
	FCO      bool `yaml:"fco" json:"fco"`
	ConstEnc bool `yaml:"constenc" json:"constenc"`
}

type MetadataProfile struct {
	RemapPath        string                 `yaml:"remap_path" json:"remap_path"`
	Icon             string                 `yaml:"icon" json:"icon"`
	CompileTime      string                 `yaml:"compile_time" json:"compile_time"`
	FileVersion      string                 `yaml:"file_version" json:"file_version"`
	ProductVersion   string                 `yaml:"product_version" json:"product_version"`
	CompanyName      string                 `yaml:"company_name" json:"company_name"`
	ProductName      string                 `yaml:"product_name" json:"product_name"`
	OriginalFilename string                 `yaml:"original_filename" json:"original_filename"`
	FileDescription  string                 `yaml:"file_description" json:"file_description"`
	InternalName     string                 `yaml:"internal_name" json:"internal_name"`
	RequireAdmin     bool                   `yaml:"require_admin" json:"require_admin"`
	RequireUAC       bool                   `yaml:"require_uac" json:"require_uac"`
	Extras           map[string]interface{} `yaml:",inline" json:",inline"`
}

type BuildProfile struct {
	ZigBuild  bool                   `yaml:"zigbuild" json:"zigbuild"`
	Remap     bool                   `yaml:"remap" json:"remap"`
	Toolchain string                 `yaml:"toolchain" json:"toolchain"`
	OLLVM     *OLLVMProfile          `yaml:"ollvm" json:"ollvm"`
	Metadata  *MetadataProfile       `yaml:"metadata" json:"metadata"`
	Extras    map[string]interface{} `yaml:",inline" json:",inline"`
}

type ProfileConfig struct {
	Basic   *BasicProfile          `yaml:"basic" json:"basic"`
	Pulse   *PulseProfile          `yaml:"pulse" json:"pulse"`
	Implant *ImplantProfile        `yaml:"implants" json:"implants"`
	Build   *BuildProfile          `yaml:"build" json:"build"`
	Extras  map[string]interface{} `yaml:",inline" json:",inline"`
}

type ProfileParams struct {
	Cron   string  `json:"cron"`
	Jitter float64 `json:"jitter"`
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

// LoadProfileFromFile 从文件加载Profile配置
func LoadProfileFromFile(filename string) (*ProfileConfig, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile file %s: %w", filename, err)
	}

	return LoadProfile(content)
}

// LoadProfileFromContent 从文件加载Profile配置
func LoadProfileFromContent(content []byte) (*ProfileConfig, error) {
	return LoadProfile(content)
}

// ToYAML 将Profile配置转换为YAML格式
func (p *ProfileConfig) ToYAML() ([]byte, error) {
	return yaml.Marshal(p)
}

// ToJSON 将Profile配置转换为JSON格式
func (p *ProfileConfig) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// SetDefaults 设置默认值
func (p *ProfileConfig) SetDefaults() {
	defaultProfile, err := LoadProfile(consts.DefaultProfile)
	if err == nil {
		*p = *defaultProfile
	}
}

// Clone 深拷贝ProfileConfig
func (p *ProfileConfig) Clone() (*ProfileConfig, error) {
	data, err := p.ToJSON()
	if err != nil {
		return nil, err
	}

	var cloned ProfileConfig
	err = json.Unmarshal(data, &cloned)
	if err != nil {
		return nil, err
	}

	return &cloned, nil
}

// Merge 合并另一个ProfileConfig的非空值
func (p *ProfileConfig) Merge(other *ProfileConfig) {
	if other == nil {
		return
	}

	if other.Basic != nil {
		if p.Basic == nil {
			p.Basic = &BasicProfile{}
		}
		p.mergeBasicProfile(other.Basic)
	}

	if other.Pulse != nil {
		if p.Pulse == nil {
			p.Pulse = &PulseProfile{}
		}
		p.mergePulseProfile(other.Pulse)
	}

	if other.Implant != nil {
		if p.Implant == nil {
			p.Implant = &ImplantProfile{}
		}
		p.mergeImplantProfile(other.Implant)
	}

	if other.Build != nil {
		if p.Build == nil {
			p.Build = &BuildProfile{}
		}
		p.mergeBuildProfile(other.Build)
	}
}

func (p *ProfileConfig) mergeBasicProfile(other *BasicProfile) {
	if other.Name != "" {
		p.Basic.Name = other.Name
	}
	if other.Cron != "" {
		p.Basic.Cron = other.Cron
	}
	if other.Jitter != 0 {
		p.Basic.Jitter = other.Jitter
	}
	if other.InitRetry != 0 {
		p.Basic.InitRetry = other.InitRetry
	}
	if other.ServerRetry != 0 {
		p.Basic.ServerRetry = other.ServerRetry
	}
	if other.GlobalRetry != 0 {
		p.Basic.GlobalRetry = other.GlobalRetry
	}
	if other.Encryption != "" {
		p.Basic.Encryption = other.Encryption
	}
	if other.Key != "" {
		p.Basic.Key = other.Key
	}
	if other.Proxy != nil {
		p.Basic.Proxy = other.Proxy
	}
	if other.Secure != nil {
		p.Basic.Secure = other.Secure
	}
	if other.DGA != nil {
		p.Basic.DGA = other.DGA
	}
	if other.Guardrail != nil {
		p.Basic.Guardrail = other.Guardrail
	}
	if len(other.Targets) > 0 {
		p.Basic.Targets = other.Targets
	}
}

func (p *ProfileConfig) mergePulseProfile(other *PulseProfile) {
	if other.Encryption != "" {
		p.Pulse.Encryption = other.Encryption
	}
	if other.Key != "" {
		p.Pulse.Key = other.Key
	}
	if other.Target != "" {
		p.Pulse.Target = other.Target
	}
	if other.Protocol != "" {
		p.Pulse.Protocol = other.Protocol
	}
	if other.Flags != nil {
		p.Pulse.Flags = other.Flags
	}
	if other.Http != nil {
		p.Pulse.Http = other.Http
	}
}

func (p *ProfileConfig) mergeImplantProfile(other *ImplantProfile) {
	if other.Runtime != "" {
		p.Implant.Runtime = other.Runtime
	}
	if other.Mod != "" {
		p.Implant.Mod = other.Mod
	}
	if other.Prelude != "" {
		p.Implant.Prelude = other.Prelude
	}
	if len(other.Modules) > 0 {
		p.Implant.Modules = other.Modules
	}
	if len(other.ThirdModules) > 0 {
		p.Implant.ThirdModules = other.ThirdModules
	}
	if len(other.Pack) > 0 {
		p.Implant.Pack = other.Pack
	}
	if other.Flags != nil {
		p.Implant.Flags = other.Flags
	}
	if other.Anti != nil {
		p.Implant.Anti = other.Anti
	}
	if other.APIs != nil {
		p.Implant.APIs = other.APIs
	}
	if other.Allocator != nil {
		p.Implant.Allocator = other.Allocator
	}
	// 布尔值需要特殊处理
	p.Implant.RegisterInfo = other.RegisterInfo
	p.Implant.HotLoad = other.HotLoad
	p.Implant.Enable3rd = other.Enable3rd
	p.Implant.ThreadStackSpoofer = other.ThreadStackSpoofer
}

func (p *ProfileConfig) mergeBuildProfile(other *BuildProfile) {
	if other.Toolchain != "" {
		p.Build.Toolchain = other.Toolchain
	}
	if other.OLLVM != nil {
		p.Build.OLLVM = other.OLLVM
	}
	if other.Metadata != nil {
		p.Build.Metadata = other.Metadata
	}
	// 布尔值需要特殊处理
	p.Build.ZigBuild = other.ZigBuild
	p.Build.Remap = other.Remap
}

// Validate 验证配置的有效性
func (p *ProfileConfig) Validate() error {
	if p.Basic == nil {
		return fmt.Errorf("basic profile is required")
	}

	if p.Basic.Name == "" {
		return fmt.Errorf("profile name is required")
	}

	if len(p.Basic.Targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}

	for i, target := range p.Basic.Targets {
		if target.Address == "" {
			return fmt.Errorf("target[%d] address is required", i)
		}
	}

	return nil
}
