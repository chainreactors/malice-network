package wizard

// Templates is a registry of predefined wizard templates
var Templates = map[string]func() *Wizard{
	"listener_setup": NewListenerSetupWizard,
	"tcp_pipeline":   NewTCPPipelineWizard,
	"http_pipeline":  NewHTTPPipelineWizard,
	"profile_create": NewProfileCreateWizard,
}

// NewListenerSetupWizard creates a wizard for listener configuration
func NewListenerSetupWizard() *Wizard {
	return NewWizard("listener_setup", "Listener Setup").
		WithDescription("Configure a new listener").
		Input("name", "Listener Name", "").
		Input("host", "Host Address", "0.0.0.0").
		Select("protocol", "Protocol", []string{"tcp", "http", "https"}).
		Number("port", "Port", 443).
		Confirm("tls", "Enable TLS?", true)
}

// NewTCPPipelineWizard creates a wizard for TCP pipeline setup
func NewTCPPipelineWizard() *Wizard {
	return NewWizard("tcp_pipeline", "TCP Pipeline Setup").
		WithDescription("Configure a new TCP pipeline").
		Input("name", "Pipeline Name", "").
		Input("listener_id", "Listener ID", "").
		Input("host", "Host Address", "0.0.0.0").
		Number("port", "Port", 5001).
		Confirm("tls", "Enable TLS?", false)
}

// NewHTTPPipelineWizard creates a wizard for HTTP pipeline setup
func NewHTTPPipelineWizard() *Wizard {
	return NewWizard("http_pipeline", "HTTP Pipeline Setup").
		WithDescription("Configure a new HTTP pipeline").
		Input("name", "Pipeline Name", "").
		Input("listener_id", "Listener ID", "").
		Input("host", "Host Address", "0.0.0.0").
		Number("port", "Port", 443).
		Confirm("tls", "Enable TLS?", true)
}

// NewProfileCreateWizard creates a wizard for profile creation
func NewProfileCreateWizard() *Wizard {
	return NewWizard("profile_create", "Create Profile").
		WithDescription("Create a new implant profile").
		Input("name", "Profile Name", "").
		Input("pipeline", "Pipeline ID", "").
		Select("type", "Implant Type", []string{"beacon", "bind", "prelude"}).
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

// GetTemplate returns a wizard template by name
func GetTemplate(name string) (*Wizard, bool) {
	if fn, ok := Templates[name]; ok {
		return fn(), true
	}
	return nil, false
}

// ListTemplates returns all available template names
func ListTemplates() []string {
	names := make([]string, 0, len(Templates))
	for name := range Templates {
		names = append(names, name)
	}
	return names
}

// RegisterTemplate registers a new wizard template
func RegisterTemplate(name string, factory func() *Wizard) {
	Templates[name] = factory
}
