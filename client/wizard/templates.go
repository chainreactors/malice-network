package wizard

import (
	"sort"
	"sync"
)

// WizardCategory represents a group of related wizards
type WizardCategory struct {
	Name        string            // Category command name (e.g., "build")
	Title       string            // Display title
	Description string            // Category description
	Wizards     []WizardEntry     // Wizards in this category
}

// WizardEntry represents a wizard within a category
type WizardEntry struct {
	ID          string // Short ID within category (e.g., "beacon")
	FullID      string // Full template ID (e.g., "build_beacon")
	Description string // Description for selection menu
}

// Categories defines the wizard groupings
var Categories = []WizardCategory{
	{
		Name:        "build",
		Title:       "Build",
		Description: "Build implants and payloads",
		Wizards: []WizardEntry{
			{ID: "beacon", FullID: "build_beacon", Description: "Build a beacon implant with full options"},
			{ID: "pulse", FullID: "build_pulse", Description: "Build stage-0 shellcode"},
			{ID: "prelude", FullID: "build_prelude", Description: "Build multi-stage payload"},
			{ID: "module", FullID: "build_module", Description: "Build custom module DLL"},
		},
	},
	{
		Name:        "pipeline",
		Title:       "Pipeline",
		Description: "Configure communication pipelines",
		Wizards: []WizardEntry{
			{ID: "tcp", FullID: "tcp_pipeline", Description: "Configure a TCP pipeline"},
			{ID: "http", FullID: "http_pipeline", Description: "Configure an HTTP pipeline"},
			{ID: "bind", FullID: "bind_pipeline", Description: "Configure a bind pipeline"},
			{ID: "rem", FullID: "rem_pipeline", Description: "Configure a REM pipeline"},
		},
	},
	{
		Name:        "cert",
		Title:       "Certificate",
		Description: "Manage TLS certificates",
		Wizards: []WizardEntry{
			{ID: "generate", FullID: "cert_generate", Description: "Generate a self-signed certificate"},
			{ID: "import", FullID: "cert_import", Description: "Import an existing certificate"},
		},
	},
	{
		Name:        "config",
		Title:       "Config",
		Description: "Configure external services",
		Wizards: []WizardEntry{
			{ID: "github", FullID: "github_config", Description: "Configure GitHub Actions build"},
			{ID: "notify", FullID: "notify_config", Description: "Configure notification channels"},
		},
	},
}

// StandaloneWizards are wizards that don't belong to a category
var StandaloneWizards = []WizardEntry{
	{ID: "listener", FullID: "listener_setup", Description: "Configure a new listener"},
	{ID: "profile", FullID: "profile_create", Description: "Create a new implant profile"},
	{ID: "infra", FullID: "infrastructure_setup", Description: "One-stop C2 infrastructure setup"},
}

// Templates is a registry of predefined wizard templates
var Templates = map[string]func() *Wizard{
	// Existing
	"listener_setup": NewListenerSetupWizard,
	"tcp_pipeline":   NewTCPPipelineWizard,
	"http_pipeline":  NewHTTPPipelineWizard,
	"profile_create": NewProfileCreateWizard,
	// Build
	"build_beacon":  NewBuildBeaconWizard,
	"build_pulse":   NewBuildPulseWizard,
	"build_prelude": NewBuildPreludeWizard,
	"build_module":  NewBuildModuleWizard,
	// Pipeline
	"bind_pipeline": NewBindPipelineWizard,
	"rem_pipeline":  NewRemPipelineWizard,
	// Certificate
	"cert_generate": NewCertGenerateWizard,
	"cert_import":   NewCertImportWizard,
	// Config
	"github_config": NewGithubConfigWizard,
	"notify_config": NewNotifyConfigWizard,
	// Composite
	"infrastructure_setup": NewInfrastructureSetupWizard,
}

// GetCategory returns a category by name
func GetCategory(name string) *WizardCategory {
	for i := range Categories {
		if Categories[i].Name == name {
			return &Categories[i]
		}
	}
	return nil
}

// GetStandaloneWizard returns a standalone wizard by ID
func GetStandaloneWizard(id string) *WizardEntry {
	for i := range StandaloneWizards {
		if StandaloneWizards[i].ID == id {
			return &StandaloneWizards[i]
		}
	}
	return nil
}

var templatesMu sync.RWMutex

// NewListenerSetupWizard creates a wizard for listener configuration
func NewListenerSetupWizard() *Wizard {
	return NewWizard("listener_setup", "Listener Setup").
		WithDescription("Configure a new listener").
		Input("name", "Listener Name", "").Field().SetRequired().
		Input("host", "Host Address", "0.0.0.0").Field().SetValidate(ValidateHost()).
		Select("protocol", "Protocol", []string{"tcp", "http", "https"}).Field().SetRequired().
		Number("port", "Port", 443).Field().SetValidate(ValidatePort()).
		Confirm("tls", "Enable TLS?", true)
}

// NewTCPPipelineWizard creates a wizard for TCP pipeline setup
func NewTCPPipelineWizard() *Wizard {
	return NewWizard("tcp_pipeline", "TCP Pipeline Setup").
		WithDescription("Configure a new TCP pipeline").
		Input("name", "Pipeline Name", "").
		Select("listener_id", "Listener ID", []string{""}).Field().SetRequired().
		Input("host", "Host Address", "0.0.0.0").Field().SetValidate(ValidateHost()).
		Number("port", "Port", 5001).Field().SetValidate(ValidatePort()).
		Confirm("tls", "Enable TLS?", false)
}

// NewHTTPPipelineWizard creates a wizard for HTTP pipeline setup
func NewHTTPPipelineWizard() *Wizard {
	return NewWizard("http_pipeline", "HTTP Pipeline Setup").
		WithDescription("Configure a new HTTP pipeline").
		Input("name", "Pipeline Name", "").
		Select("listener_id", "Listener ID", []string{""}).Field().SetRequired().
		Input("host", "Host Address", "0.0.0.0").Field().SetValidate(ValidateHost()).
		Number("port", "Port", 443).Field().SetValidate(ValidatePort()).
		Confirm("tls", "Enable TLS?", true)
}

// NewProfileCreateWizard creates a wizard for profile creation
func NewProfileCreateWizard() *Wizard {
	return NewWizard("profile_create", "Create Profile").
		WithDescription("Create a new implant profile").
		Input("name", "Profile Name", "").Field().SetRequired().
		Select("pipeline", "Pipeline ID", []string{""}).Field().SetRequired().
		Select("type", "Implant Type", []string{"beacon", "bind", "prelude"}).Field().SetRequired().
		MultiSelect("modules", "Modules", []string{
			"base",
			"sys_full",
			"execute_exe",
			"execute_dll",
			"execute_bof",
			"execute_shellcode",
			"execute_assembly",
		})
}

// NewBuildBeaconWizard creates a wizard for building beacon implant
func NewBuildBeaconWizard() *Wizard {
	return NewWizard("build_beacon", "Build Beacon").
		WithDescription("Build a beacon implant with full options").
		// Basic configuration
		Select("profile", "Profile Name", []string{""}).
		Select("target", "Build Target", []string{
			"x86_64-pc-windows-gnu", "i686-pc-windows-gnu",
			"x86_64-pc-windows-msvc", "i686-pc-windows-msvc",
			"x86_64-unknown-linux-musl", "i686-unknown-linux-musl",
			"x86_64-apple-darwin", "aarch64-apple-darwin",
		}).Field().SetRequired().
		Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().SetRequired().
		Confirm("lib", "Build as Library (DLL/SO)?", false).
		// Network configuration
		Select("addresses", "C2 Addresses", []string{""}).Field().SetRequired().
		Input("proxy", "Proxy URL", "").
		Confirm("proxy_use_env", "Use Environment Proxy?", false).
		// Communication parameters
		Input("cron", "Cron Expression", "*/5 * * * * * *").
		Input("jitter", "Jitter (0.0-1.0)", "0.2").Field().SetValidate(ValidateFloat(0, 1)).
		Number("init_retry", "Initial Retry Count", 3).Field().SetValidate(ValidateRange(0, 100)).
		Number("server_retry", "Server Retry Count", 3).Field().SetValidate(ValidateRange(0, 100)).
		Number("global_retry", "Global Retry Count", 3).Field().SetValidate(ValidateRange(0, 100)).
		// Encryption configuration
		Select("encryption", "Encryption Type", []string{"", "aes", "xor"}).
		Input("key", "Encryption Key (empty for auto)", "").
		Confirm("secure", "Enable Secure Mode?", false).
		// Module selection
		MultiSelect("modules", "Modules", []string{
			"nano", "full", "base", "extend",
			"fs_full", "sys_full", "execute_full", "net_full",
		}).
		MultiSelect("third_modules", "3rd Party Modules", []string{"rem", "curl"}).
		// Protection configuration
		Confirm("anti_sandbox", "Enable Anti-Sandbox?", false).
		Input("guardrail_ips", "Guardrail IPs (comma-separated)", "").
		Input("guardrail_users", "Guardrail Usernames", "").
		Input("guardrail_servers", "Guardrail Server Names", "").
		Input("guardrail_domains", "Guardrail Domains", "").
		Confirm("ollvm", "Enable OLLVM Obfuscation?", false)
}

// NewBuildPulseWizard creates a wizard for building pulse shellcode
func NewBuildPulseWizard() *Wizard {
	return NewWizard("build_pulse", "Build Pulse").
		WithDescription("Build stage-0 shellcode").
		Select("target", "Build Target", []string{
			"x86_64-pc-windows-gnu", "i686-pc-windows-gnu",
			"x86_64-pc-windows-msvc", "i686-pc-windows-msvc",
		}).Field().SetRequired().
		Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().SetRequired().
		Select("profile", "Profile Name", []string{""}).
		Select("address", "C2 Address", []string{""}).Field().SetRequired().
		Input("user_agent", "User-Agent", "").
		Select("beacon_artifact_id", "Beacon Artifact ID", []string{""}).
		Input("path", "HTTP Path", "/pulse")
}

// NewBuildPreludeWizard creates a wizard for building multi-stage payload
func NewBuildPreludeWizard() *Wizard {
	return NewWizard("build_prelude", "Build Prelude").
		WithDescription("Build multi-stage payload").
		Select("target", "Build Target", []string{
			"x86_64-pc-windows-gnu", "i686-pc-windows-gnu",
			"x86_64-pc-windows-msvc", "i686-pc-windows-msvc",
			"x86_64-unknown-linux-musl", "i686-unknown-linux-musl",
			"x86_64-apple-darwin", "aarch64-apple-darwin",
		}).Field().SetRequired().
		FilePath("autorun", "Autorun ZIP File").
		Select("profile", "Profile Name", []string{""}).
		Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().SetRequired()
}

// NewBuildModuleWizard creates a wizard for building custom module DLL
func NewBuildModuleWizard() *Wizard {
	return NewWizard("build_module", "Build Module").
		WithDescription("Build custom module DLL").
		Select("target", "Build Target", []string{
			"x86_64-pc-windows-gnu", "i686-pc-windows-gnu",
			"x86_64-pc-windows-msvc", "i686-pc-windows-msvc",
		}).Field().SetRequired().
		MultiSelect("modules", "Modules", []string{
			"nano", "full", "base", "extend",
			"fs_full", "sys_full", "execute_full", "net_full",
		}).
		MultiSelect("third_modules", "3rd Party Modules", []string{"rem", "curl"}).
		Select("profile", "Profile Name", []string{""}).
		Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().SetRequired()
}

// NewBindPipelineWizard creates a wizard for bind pipeline setup
func NewBindPipelineWizard() *Wizard {
	return NewWizard("bind_pipeline", "Bind Pipeline Setup").
		WithDescription("Configure a bind pipeline").
		Select("listener_id", "Listener ID", []string{""}).Field().SetRequired()
}

// NewRemPipelineWizard creates a wizard for REM pipeline setup
func NewRemPipelineWizard() *Wizard {
	return NewWizard("rem_pipeline", "REM Pipeline Setup").
		WithDescription("Configure a REM pipeline").
		Input("name", "Pipeline Name", "").
		Select("listener_id", "Listener ID", []string{""}).Field().SetRequired().
		Input("console", "Console URL (tcp://host:port)", "tcp://0.0.0.0:19966").
		Confirm("secure", "Enable Secure Mode?", false)
}

// NewCertGenerateWizard creates a wizard for certificate generation
func NewCertGenerateWizard() *Wizard {
	return NewWizard("cert_generate", "Generate Certificate").
		WithDescription("Generate a self-signed certificate").
		Input("cn", "Common Name (CN)", "").Field().SetRequired().
		Input("o", "Organization (O)", "").
		Input("c", "Country (C)", "").
		Input("l", "Locality (L)", "").
		Input("ou", "Organizational Unit (OU)", "").
		Input("st", "State/Province (ST)", "").
		Number("validity", "Validity (Days)", 365).Field().SetValidate(ValidateRange(1, 3650))
}

// NewCertImportWizard creates a wizard for certificate import
func NewCertImportWizard() *Wizard {
	return NewWizard("cert_import", "Import Certificate").
		WithDescription("Import an existing certificate").
		FilePath("cert", "Certificate File").Field().SetRequired().
		FilePath("key", "Private Key File").Field().SetRequired().
		FilePath("ca_cert", "CA Certificate (optional)")
}

// NewGithubConfigWizard creates a wizard for GitHub Actions configuration
func NewGithubConfigWizard() *Wizard {
	return NewWizard("github_config", "GitHub Configuration").
		WithDescription("Configure GitHub Actions build").
		Input("owner", "GitHub Owner/Org", "").Field().SetRequired().
		Input("repo", "Repository Name", "").Field().SetRequired().
		Input("token", "GitHub Token", "").Field().SetRequired().
		Input("workflow_file", "Workflow File", "")
}

// NewNotifyConfigWizard creates a wizard for notification configuration
func NewNotifyConfigWizard() *Wizard {
	return NewWizard("notify_config", "Notification Configuration").
		WithDescription("Configure notification channels").
		// Telegram
		Confirm("telegram_enable", "Enable Telegram?", false).
		Input("telegram_token", "Telegram Bot Token", "").
		Input("telegram_chat_id", "Telegram Chat ID", "").
		// DingTalk
		Confirm("dingtalk_enable", "Enable DingTalk?", false).
		Input("dingtalk_token", "DingTalk Token", "").
		Input("dingtalk_secret", "DingTalk Secret", "").
		// Lark
		Confirm("lark_enable", "Enable Lark?", false).
		Input("lark_webhook", "Lark Webhook URL", "").
		// ServerChan
		Confirm("serverchan_enable", "Enable ServerChan?", false).
		Input("serverchan_url", "ServerChan URL", "").
		// PushPlus
		Confirm("pushplus_enable", "Enable PushPlus?", false).
		Input("pushplus_token", "PushPlus Token", "").
		Input("pushplus_topic", "PushPlus Topic", "")
}

// NewInfrastructureSetupWizard creates a wizard for one-stop infrastructure setup
func NewInfrastructureSetupWizard() *Wizard {
	return NewWizard("infrastructure_setup", "Infrastructure Setup").
		WithDescription("One-stop C2 infrastructure setup").
		// Listener
		Input("listener_name", "Listener Name", "").Field().SetRequired().
		Input("listener_host", "Listener Host", "0.0.0.0").Field().SetValidate(ValidateHost()).
		Select("listener_protocol", "Protocol", []string{"tcp", "http", "https"}).Field().SetRequired().
		Number("listener_port", "Listener Port", 443).Field().SetValidate(ValidatePort()).
		Confirm("listener_tls", "Enable TLS?", true).
		// Pipeline
		Select("pipeline_type", "Pipeline Type", []string{"tcp", "http"}).Field().SetRequired().
		Input("pipeline_name", "Pipeline Name", "").
		Input("pipeline_host", "Pipeline Host", "0.0.0.0").Field().SetValidate(ValidateHost()).
		Number("pipeline_port", "Pipeline Port", 5001).Field().SetValidate(ValidatePort()).
		Confirm("pipeline_tls", "Enable Pipeline TLS?", false).
		// Profile
		Input("profile_name", "Profile Name", "").Field().SetRequired().
		Select("implant_type", "Implant Type", []string{"beacon", "bind", "prelude"}).Field().SetRequired().
		MultiSelect("modules", "Modules", []string{
			"base", "sys_full", "execute_full", "net_full",
		})
}

// GetTemplate returns a wizard template by name
func GetTemplate(name string) (*Wizard, bool) {
	templatesMu.RLock()
	fn, ok := Templates[name]
	templatesMu.RUnlock()
	if ok {
		return fn(), true
	}
	return nil, false
}

// ListTemplates returns all available template names
func ListTemplates() []string {
	templatesMu.RLock()
	names := make([]string, 0, len(Templates))
	for name := range Templates {
		names = append(names, name)
	}
	templatesMu.RUnlock()
	sort.Strings(names)
	return names
}

// RegisterTemplate registers a new wizard template
func RegisterTemplate(name string, factory func() *Wizard) {
	templatesMu.Lock()
	Templates[name] = factory
	templatesMu.Unlock()
}
