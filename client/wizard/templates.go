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
		Input("name", "Listener Name", "").Field().Require().Desc("Unique identifier for this listener").End().
		Input("host", "Host Address", "0.0.0.0").Field().WithValidate(ValidateHost()).Desc("IP to bind, 0.0.0.0 = all interfaces").End().
		Select("protocol", "Protocol", []string{"tcp", "http", "https"}).Field().Require().Desc("Communication protocol type").End().
		Number("port", "Port", 443).Field().WithValidate(ValidatePort()).Desc("Port number (443=HTTPS, 80=HTTP, etc.)").End().
		Confirm("tls", "Enable TLS?", true).Field().Desc("Enable TLS encryption for secure communication").End()
}

// NewTCPPipelineWizard creates a wizard for TCP pipeline setup
func NewTCPPipelineWizard() *Wizard {
	return NewWizard("tcp_pipeline", "TCP Pipeline Setup").
		WithDescription("Configure a new TCP pipeline").
		Input("name", "Pipeline Name", "").Field().Desc("Unique identifier for this pipeline").End().
		Select("listener_id", "Listener ID", []string{""}).Field().Require().Desc("Parent listener to attach this pipeline").End().
		Input("host", "Host Address", "0.0.0.0").Field().WithValidate(ValidateHost()).Desc("IP to bind for implant connections").End().
		Number("port", "Port", 5001).Field().WithValidate(ValidatePort()).Desc("TCP port for implant callbacks").End().
		Confirm("tls", "Enable TLS?", false).Field().Desc("Enable TLS encryption on TCP connection").End()
}

// NewHTTPPipelineWizard creates a wizard for HTTP pipeline setup
func NewHTTPPipelineWizard() *Wizard {
	return NewWizard("http_pipeline", "HTTP Pipeline Setup").
		WithDescription("Configure a new HTTP pipeline").
		Input("name", "Pipeline Name", "").Field().Desc("Unique identifier for this pipeline").End().
		Select("listener_id", "Listener ID", []string{""}).Field().Require().Desc("Parent listener to attach this pipeline").End().
		Input("host", "Host Address", "0.0.0.0").Field().WithValidate(ValidateHost()).Desc("IP to bind for HTTP requests").End().
		Number("port", "Port", 443).Field().WithValidate(ValidatePort()).Desc("HTTP port (443 for HTTPS, 80 for HTTP)").End().
		Confirm("tls", "Enable TLS?", true).Field().Desc("Enable HTTPS (recommended for production)").End()
}

// NewProfileCreateWizard creates a wizard for profile creation
func NewProfileCreateWizard() *Wizard {
	return NewWizard("profile_create", "Create Profile").
		WithDescription("Create a new implant profile").
		Input("name", "Profile Name", "").Field().Require().Desc("Unique identifier for this profile").End().
		Select("pipeline", "Pipeline ID", []string{""}).Field().Require().Desc("Pipeline for C2 communication").End().
		Select("type", "Implant Type", []string{"beacon", "bind", "prelude"}).Field().Require().Desc("beacon=persistent, bind=reverse, prelude=staged").End().
		MultiSelect("modules", "Modules", []string{
			"base",
			"sys_full",
			"execute_exe",
			"execute_dll",
			"execute_bof",
			"execute_shellcode",
			"execute_assembly",
		}).Field().Desc("Select modules to include in the implant").End()
}

// NewBuildBeaconWizard creates a wizard for building beacon implant
func NewBuildBeaconWizard() *Wizard {
	return NewWizard("build_beacon", "Build Beacon").
		WithDescription("Build a beacon implant with full options").
		// Group 1: Basic Configuration
		NewGroup("basic", "Basic Configuration").
			WithDescription("Core build settings").
			Select("profile", "Profile Name", []string{""}).Field().Desc("Implant profile with pipeline settings").EndGroup().
			Select("target", "Build Target", []string{
				"x86_64-pc-windows-gnu", "i686-pc-windows-gnu",
				"x86_64-pc-windows-msvc", "i686-pc-windows-msvc",
				"x86_64-unknown-linux-musl", "i686-unknown-linux-musl",
				"x86_64-apple-darwin", "aarch64-apple-darwin",
			}).Field().Require().Desc("Target OS and architecture").EndGroup().
			Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().Require().Desc("docker=local, action=GitHub, saas=cloud").EndGroup().
			Confirm("lib", "Build as Library (DLL/SO)?", false).Field().Desc("Build as DLL/SO instead of executable").EndGroup().
			End().
		// Group 2: Network Configuration
		NewGroup("network", "Network Configuration").
			WithDescription("C2 connection settings").
			Select("addresses", "C2 Addresses", []string{""}).Field().Require().Desc("Server addresses for callbacks").EndGroup().
			Input("proxy", "Proxy URL", "").Field().Desc("HTTP proxy URL (optional)").EndGroup().
			Confirm("proxy_use_env", "Use Environment Proxy?", false).Field().Desc("Use system proxy settings").EndGroup().
			End().
		// Group 3: Communication Parameters
		NewGroup("timing", "Communication Parameters").
			WithDescription("Timing and retry settings").
			Input("cron", "Cron Expression", "*/5 * * * * * *").Field().Desc("Callback schedule (*/5 = every 5 sec)").EndGroup().
			Input("jitter", "Jitter (0.0-1.0)", "0.2").Field().WithValidate(ValidateFloat(0, 1)).Desc("Random delay factor (0.0-1.0)").EndGroup().
			Number("init_retry", "Initial Retry Count", 3).Field().WithValidate(ValidateRange(0, 100)).Desc("Retry count on initial connection").EndGroup().
			Number("server_retry", "Server Retry Count", 3).Field().WithValidate(ValidateRange(0, 100)).Desc("Retry count per server address").EndGroup().
			Number("global_retry", "Global Retry Count", 3).Field().WithValidate(ValidateRange(0, 100)).Desc("Total retry count before exit").EndGroup().
			End().
		// Group 4: Encryption Configuration
		NewGroup("crypto", "Encryption Configuration").
			WithDescription("Traffic encryption settings").
			Select("encryption", "Encryption Type", []string{"", "aes", "xor"}).Field().Desc("Traffic encryption method").EndGroup().
			Input("key", "Encryption Key (empty for auto)", "").Field().Desc("Custom key or empty for auto-generate").EndGroup().
			Confirm("secure", "Enable Secure Mode?", false).Field().Desc("Enhanced security features").EndGroup().
			End().
		// Group 5: Module Selection
		NewGroup("modules", "Module Selection").
			WithDescription("Select modules to include").
			MultiSelect("modules", "Modules", []string{
				"nano", "full", "base", "extend",
				"fs_full", "sys_full", "execute_full", "net_full",
			}).Field().Desc("Built-in module packages to include").EndGroup().
			MultiSelect("third_modules", "3rd Party Modules", []string{"rem", "curl"}).Field().Desc("Additional third-party modules").EndGroup().
			End().
		// Group 6: Protection Configuration
		NewGroup("protection", "Protection Configuration").
			WithDescription("Anti-analysis and guardrails").
			Confirm("anti_sandbox", "Enable Anti-Sandbox?", false).Field().Desc("Detect and evade sandbox environments").EndGroup().
			Input("guardrail_ips", "Guardrail IPs (comma-separated)", "").Field().Desc("Only run if target has these IPs").EndGroup().
			Input("guardrail_users", "Guardrail Usernames", "").Field().Desc("Only run for these usernames").EndGroup().
			Input("guardrail_servers", "Guardrail Server Names", "").Field().Desc("Only run on these server names").EndGroup().
			Input("guardrail_domains", "Guardrail Domains", "").Field().Desc("Only run in these domains").EndGroup().
			Confirm("ollvm", "Enable OLLVM Obfuscation?", false).Field().Desc("Apply OLLVM code obfuscation").EndGroup().
			End()
}

// NewBuildPulseWizard creates a wizard for building pulse shellcode
func NewBuildPulseWizard() *Wizard {
	return NewWizard("build_pulse", "Build Pulse").
		WithDescription("Build stage-0 shellcode").
		Select("target", "Build Target", []string{
			"x86_64-pc-windows-gnu", "i686-pc-windows-gnu",
			"x86_64-pc-windows-msvc", "i686-pc-windows-msvc",
		}).Field().Require().Desc("Target OS and architecture").End().
		Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().Require().Desc("docker=local, action=GitHub, saas=cloud").End().
		Select("profile", "Profile Name", []string{""}).Field().Desc("Implant profile with pipeline settings").End().
		Select("address", "C2 Address", []string{""}).Field().Require().Desc("Server address for stage-1 download").End().
		Input("user_agent", "User-Agent", "").Field().Desc("Custom User-Agent header for HTTP requests").End().
		Select("beacon_artifact_id", "Beacon Artifact ID", []string{""}).Field().Desc("Pre-built beacon artifact to download").End().
		Input("path", "HTTP Path", "/pulse").Field().Desc("HTTP path for stage-1 download").End()
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
		}).Field().Require().Desc("Target OS and architecture").End().
		FilePath("autorun", "Autorun ZIP File").Field().Desc("ZIP file with autorun scripts").End().
		Select("profile", "Profile Name", []string{""}).Field().Desc("Implant profile with pipeline settings").End().
		Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().Require().Desc("docker=local, action=GitHub, saas=cloud").End()
}

// NewBuildModuleWizard creates a wizard for building custom module DLL
func NewBuildModuleWizard() *Wizard {
	return NewWizard("build_module", "Build Module").
		WithDescription("Build custom module DLL").
		Select("target", "Build Target", []string{
			"x86_64-pc-windows-gnu", "i686-pc-windows-gnu",
			"x86_64-pc-windows-msvc", "i686-pc-windows-msvc",
		}).Field().Require().Desc("Target OS and architecture").End().
		MultiSelect("modules", "Modules", []string{
			"nano", "full", "base", "extend",
			"fs_full", "sys_full", "execute_full", "net_full",
		}).Field().Desc("Built-in module packages to include").End().
		MultiSelect("third_modules", "3rd Party Modules", []string{"rem", "curl"}).Field().Desc("Additional third-party modules").End().
		Select("profile", "Profile Name", []string{""}).Field().Desc("Implant profile with pipeline settings").End().
		Select("source", "Build Source", []string{"docker", "action", "saas"}).Field().Require().Desc("docker=local, action=GitHub, saas=cloud").End()
}

// NewBindPipelineWizard creates a wizard for bind pipeline setup
func NewBindPipelineWizard() *Wizard {
	return NewWizard("bind_pipeline", "Bind Pipeline Setup").
		WithDescription("Configure a bind pipeline").
		Select("listener_id", "Listener ID", []string{""}).Field().Require().Desc("Parent listener for bind connections").End()
}

// NewRemPipelineWizard creates a wizard for REM pipeline setup
func NewRemPipelineWizard() *Wizard {
	return NewWizard("rem_pipeline", "REM Pipeline Setup").
		WithDescription("Configure a REM pipeline").
		Input("name", "Pipeline Name", "").Field().Desc("Unique identifier for this pipeline").End().
		Select("listener_id", "Listener ID", []string{""}).Field().Require().Desc("Parent listener to attach").End().
		Input("console", "Console URL (tcp://host:port)", "tcp://0.0.0.0:19966").Field().Desc("Remote console URL for REM module").End().
		Confirm("secure", "Enable Secure Mode?", false).Field().Desc("Enable encryption for REM traffic").End()
}

// NewCertGenerateWizard creates a wizard for certificate generation
func NewCertGenerateWizard() *Wizard {
	return NewWizard("cert_generate", "Generate Certificate").
		WithDescription("Generate a self-signed certificate").
		Input("cn", "Common Name (CN)", "").Field().Require().Desc("Domain name for the certificate").End().
		Input("o", "Organization (O)", "").Field().Desc("Organization name (optional)").End().
		Input("c", "Country (C)", "").Field().Desc("Two-letter country code (e.g., US, CN)").End().
		Input("l", "Locality (L)", "").Field().Desc("City or locality name").End().
		Input("ou", "Organizational Unit (OU)", "").Field().Desc("Department or division name").End().
		Input("st", "State/Province (ST)", "").Field().Desc("State or province name").End().
		Number("validity", "Validity (Days)", 365).Field().WithValidate(ValidateRange(1, 3650)).Desc("Certificate validity period in days").End()
}

// NewCertImportWizard creates a wizard for certificate import
func NewCertImportWizard() *Wizard {
	return NewWizard("cert_import", "Import Certificate").
		WithDescription("Import an existing certificate").
		FilePath("cert", "Certificate File").Field().Require().Desc("Path to certificate file (.crt, .pem)").End().
		FilePath("key", "Private Key File").Field().Require().Desc("Path to private key file (.key, .pem)").End().
		FilePath("ca_cert", "CA Certificate (optional)").Field().Desc("Path to CA certificate if needed").End()
}

// NewGithubConfigWizard creates a wizard for GitHub Actions configuration
func NewGithubConfigWizard() *Wizard {
	return NewWizard("github_config", "GitHub Configuration").
		WithDescription("Configure GitHub Actions build").
		Input("owner", "GitHub Owner/Org", "").Field().Require().Desc("GitHub username or organization").End().
		Input("repo", "Repository Name", "").Field().Require().Desc("Repository name for Actions builds").End().
		Input("token", "GitHub Token", "").Field().Require().Desc("Personal access token with repo scope").End().
		Input("workflow_file", "Workflow File", "").Field().Desc("Custom workflow filename (optional)").End()
}

// NewNotifyConfigWizard creates a wizard for notification configuration
func NewNotifyConfigWizard() *Wizard {
	return NewWizard("notify_config", "Notification Configuration").
		WithDescription("Configure notification channels").
		// Telegram
		Confirm("telegram_enable", "Enable Telegram?", false).Field().Desc("Enable Telegram bot notifications").End().
		Input("telegram_token", "Telegram Bot Token", "").Field().Desc("Bot token from @BotFather").End().
		Input("telegram_chat_id", "Telegram Chat ID", "").Field().Desc("Chat/Channel ID to send messages").End().
		// DingTalk
		Confirm("dingtalk_enable", "Enable DingTalk?", false).Field().Desc("Enable DingTalk robot notifications").End().
		Input("dingtalk_token", "DingTalk Token", "").Field().Desc("Robot access token").End().
		Input("dingtalk_secret", "DingTalk Secret", "").Field().Desc("Robot signing secret").End().
		// Lark
		Confirm("lark_enable", "Enable Lark?", false).Field().Desc("Enable Lark/Feishu notifications").End().
		Input("lark_webhook", "Lark Webhook URL", "").Field().Desc("Webhook URL from Lark bot").End().
		// ServerChan
		Confirm("serverchan_enable", "Enable ServerChan?", false).Field().Desc("Enable ServerChan push notifications").End().
		Input("serverchan_url", "ServerChan URL", "").Field().Desc("ServerChan send key URL").End().
		// PushPlus
		Confirm("pushplus_enable", "Enable PushPlus?", false).Field().Desc("Enable PushPlus notifications").End().
		Input("pushplus_token", "PushPlus Token", "").Field().Desc("PushPlus user token").End().
		Input("pushplus_topic", "PushPlus Topic", "").Field().Desc("Topic name for group messaging").End()
}

// NewInfrastructureSetupWizard creates a wizard for one-stop infrastructure setup
func NewInfrastructureSetupWizard() *Wizard {
	return NewWizard("infrastructure_setup", "Infrastructure Setup").
		WithDescription("One-stop C2 infrastructure setup").
		// Listener
		Input("listener_name", "Listener Name", "").Field().Require().Desc("Unique name for the listener").End().
		Input("listener_host", "Listener Host", "0.0.0.0").Field().WithValidate(ValidateHost()).Desc("IP to bind, 0.0.0.0 = all interfaces").End().
		Select("listener_protocol", "Protocol", []string{"tcp", "http", "https"}).Field().Require().Desc("Communication protocol type").End().
		Number("listener_port", "Listener Port", 443).Field().WithValidate(ValidatePort()).Desc("Port number (443=HTTPS, 80=HTTP)").End().
		Confirm("listener_tls", "Enable TLS?", true).Field().Desc("Enable TLS encryption").End().
		// Pipeline
		Select("pipeline_type", "Pipeline Type", []string{"tcp", "http"}).Field().Require().Desc("Pipeline protocol type").End().
		Input("pipeline_name", "Pipeline Name", "").Field().Desc("Unique name for the pipeline").End().
		Input("pipeline_host", "Pipeline Host", "0.0.0.0").Field().WithValidate(ValidateHost()).Desc("IP to bind for implant connections").End().
		Number("pipeline_port", "Pipeline Port", 5001).Field().WithValidate(ValidatePort()).Desc("Port for implant callbacks").End().
		Confirm("pipeline_tls", "Enable Pipeline TLS?", false).Field().Desc("Enable TLS on pipeline connection").End().
		// Profile
		Input("profile_name", "Profile Name", "").Field().Require().Desc("Unique name for the profile").End().
		Select("implant_type", "Implant Type", []string{"beacon", "bind", "prelude"}).Field().Require().Desc("beacon=persistent, bind=reverse, prelude=staged").End().
		MultiSelect("modules", "Modules", []string{
			"base", "sys_full", "execute_full", "net_full",
		}).Field().Desc("Select modules to include").End()
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
