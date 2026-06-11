package implanttypes

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"path/filepath"

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
	Name       string                 `yaml:"name" json:"name"`
	Proxy      *ProxyProfile          `yaml:"proxy" json:"proxy"`
	Cron       string                 `yaml:"cron" json:"cron"`
	Jitter     float64                `yaml:"jitter" json:"jitter"`
	Keepalive  bool                   `yaml:"keepalive" json:"keepalive"`
	Retry      int                    `yaml:"retry" json:"retry"`
	MaxCycles  int                    `yaml:"max_cycles" json:"max_cycles"`
	Encryption string                 `yaml:"encryption" json:"encryption"`
	Key        string                 `yaml:"key" json:"key"`
	Secure     *SecureProfile         `yaml:"secure" json:"secure"`
	DGA        *DGAProfile            `yaml:"dga" json:"dga"`
	Guardrail  *GuardrailProfile      `yaml:"guardrail" json:"guardrail"`
	Targets    []Target               `yaml:"targets" json:"targets"`
	Extras     map[string]interface{} `yaml:",inline" json:",inline"`
}

type REMProfile struct {
	Link string `yaml:"link" json:"link"`
}

type TLSProfile struct {
	Enable           bool                   `yaml:"enable" json:"enable"`
	SNI              string                 `yaml:"sni" json:"sni"`
	SkipVerification bool                   `yaml:"skip_verification" json:"skip_verification"`
	ServerCA         string                 `yaml:"server_ca,omitempty" json:"server_ca,omitempty"`
	MTLS             *MTLSProfile           `yaml:"mtls,omitempty" json:"mtls,omitempty"`
	Extras           map[string]interface{} `yaml:",inline" json:",inline"`
}

type MTLSProfile struct {
	Enable     bool   `yaml:"enable" json:"enable"`
	ClientCert string `yaml:"client_cert" json:"client_cert"`
	ClientKey  string `yaml:"client_key" json:"client_key"`
	ServerCA   string `yaml:"server_ca,omitempty" json:"server_ca,omitempty"`
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

type EvaderProfile struct {
	AntiEmu      bool `yaml:"anti_emu" json:"anti_emu"`
	EtwPass      bool `yaml:"etw_pass" json:"etw_pass"`
	GodSpeed     bool `yaml:"god_speed" json:"god_speed"`
	SleepEncrypt bool `yaml:"sleep_encrypt" json:"sleep_encrypt"`
	AntiForensic bool `yaml:"anti_forensic" json:"anti_forensic"`
	CfgPatch     bool `yaml:"cfg_patch" json:"cfg_patch"`
	ApiUntangle  bool `yaml:"api_untangle" json:"api_untangle"`
	NormalApi    bool `yaml:"normal_api" json:"normal_api"`
}

type ProxyDllProfile struct {
	ProxyFunc     string `yaml:"proxyfunc" json:"proxyfunc"`
	RawDll        string `yaml:"raw_dll" json:"raw_dll"`
	ProxiedDll    string `yaml:"proxied_dll" json:"proxied_dll"`
	ProxyDll      string `yaml:"proxy_dll" json:"proxy_dll"`
	PackResources bool   `yaml:"pack_resources" json:"pack_resources"`
	Block         bool   `yaml:"block" json:"block"`
	HijackDllmain bool   `yaml:"hijack_dllmain" json:"hijack_dllmain"`
}

type LoaderProfile struct {
	Evader   *EvaderProfile   `yaml:"evader" json:"evader"`
	ProxyDll *ProxyDllProfile `yaml:"proxydll" json:"proxydll"`
}

type ProfileConfig struct {
	Basic   *BasicProfile          `yaml:"basic" json:"basic"`
	Pulse   *PulseProfile          `yaml:"pulse" json:"pulse"`
	Implant *ImplantProfile        `yaml:"implants" json:"implants"`
	Build   *BuildProfile          `yaml:"build" json:"build"`
	Loader  *LoaderProfile         `yaml:"loader" json:"loader"`
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
