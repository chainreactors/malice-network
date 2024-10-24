package configs

type GeneratorServerConfig struct {
	Name     string   `yaml:"name" config:"name" default:"malefic"`
	Urls     []string `yaml:"urls" config:"urls" default:"[]"`
	Protocol string   `yaml:"protocol" config:"protocol" default:"tcp"`
	TLS      bool     `yaml:"tls" config:"tls" default:"false"`
	Proxy    string   `yaml:"proxy" config:"proxy" default:""`
	Interval int      `yaml:"interval" config:"interval" default:"1000"`
	Jitter   int      `yaml:"jitter" config:"jitter" default:"10"`
	CA       string   `yaml:"ca" config:"ca" default:""`
}

type ImplantsConfig struct {
	RegisterInfo       bool              `yaml:"register_info" config:"register_info" default:"true"`
	Modules            []string          `yaml:"modules" config:"modules" default:"[]"`
	Metadata           Metadata          `yaml:"metadata" config:"metadata"`
	APIs               APIsConfig        `yaml:"apis" config:"apis"`
	Allocator          Allocator         `yaml:"alloctor" config:"alloctor"`
	SleepMask          bool              `yaml:"sleep_mask" config:"sleep_mask" default:"true"`
	SacrificeProcess   bool              `yaml:"sacrifice_process" config:"sacrifice_process" default:"true"`
	ForkAndRun         bool              `yaml:"fork_and_run" config:"fork_and_run" default:"false"`
	HookExit           bool              `yaml:"hook_exit" config:"hook_exit" default:"true"`
	ThreadStackSpoofer bool              `yaml:"thread_stack_spoofer" config:"thread_stack_spoofer" default:"true"`
	PESignatureModify  PESignatureModify `yaml:"pe_signature_modify" config:"pe_signature_modify"`
}

type Metadata struct {
	RemapPath        string `yaml:"remap_path" config:"remap_path" default:"C:/Windows/Users/Maleficarum"`
	Icon             string `yaml:"icon" config:"icon" default:""`
	CompileTime      string `yaml:"compile_time" config:"compile_time" default:"24 Jun 2024 18:03:01"`
	FileVersion      string `yaml:"file_version" config:"file_version" default:""`
	ProductVersion   string `yaml:"product_version" config:"product_version" default:""`
	CompanyName      string `yaml:"company_name" config:"company_name" default:""`
	ProductName      string `yaml:"product_name" config:"product_name" default:""`
	OriginalFilename string `yaml:"original_filename" config:"original_filename" default:""`
	FileDescription  string `yaml:"file_description" config:"file_description" default:""`
	InternalName     string `yaml:"internal_name" config:"internal_name" default:""`
}

type APIsConfig struct {
	Level    string         `yaml:"level" config:"level" default:"nt_apis"`
	Priority PriorityConfig `yaml:"priority" config:"priority"`
}

type PriorityConfig struct {
	Normal   APIPriority `yaml:"normal" config:"normal"`
	Dynamic  APIPriority `yaml:"dynamic" config:"dynamic"`
	Syscalls APIPriority `yaml:"syscalls" config:"syscalls"`
}

type APIPriority struct {
	Enable bool   `yaml:"enable" config:"enable" default:"false"`
	Type   string `yaml:"type" config:"type" default:"normal"`
}

type Allocator struct {
	InProcess    string `yaml:"inprocess" config:"inprocess" default:"NtAllocateVirtualMemory"`
	CrossProcess string `yaml:"crossprocess" config:"crossprocess" default:"NtAllocateVirtualMemory"`
}

type PESignatureModify struct {
	Feature bool         `yaml:"feature" config:"feature" default:"true"`
	Modify  ModifyConfig `yaml:"modify" config:"modify"`
}

type ModifyConfig struct {
	Magic     string `yaml:"magic" config:"magic" default:"\x00\x00"`
	Signature string `yaml:"signature" config:"signature" default:"\x00\x00"`
}

type GeneratorConfig struct {
	Server   GeneratorServerConfig `yaml:"server" config:"server"`
	Implants ImplantsConfig        `yaml:"implants" config:"implants"`
}
