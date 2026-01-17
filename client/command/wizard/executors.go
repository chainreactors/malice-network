package wizard

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	wizardfw "github.com/chainreactors/malice-network/client/wizard"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	serverbuild "github.com/chainreactors/malice-network/server/build"
	"github.com/corpix/uarand"
)

// ExecutorFunc is a function that executes wizard results
type ExecutorFunc func(con *core.Console, result *wizardfw.WizardResult) error

var (
	executors   = make(map[string]ExecutorFunc)
	executorsMu sync.RWMutex
)

// RegisterExecutor registers an executor for a wizard template
func RegisterExecutor(templateID string, fn ExecutorFunc) {
	executorsMu.Lock()
	defer executorsMu.Unlock()
	executors[templateID] = fn
}

// GetExecutor returns the executor for a wizard template
func GetExecutor(templateID string) (ExecutorFunc, bool) {
	executorsMu.RLock()
	defer executorsMu.RUnlock()
	fn, ok := executors[templateID]
	return fn, ok
}

// HasExecutor checks if an executor is registered
func HasExecutor(templateID string) bool {
	executorsMu.RLock()
	defer executorsMu.RUnlock()
	_, ok := executors[templateID]
	return ok
}

func init() {
	// Register all built-in executors
	// Pipeline executors
	RegisterExecutor("tcp_pipeline", executeTCPPipeline)
	RegisterExecutor("http_pipeline", executeHTTPPipeline)
	RegisterExecutor("bind_pipeline", executeBindPipeline)
	RegisterExecutor("rem_pipeline", executeREMPipeline)
	// Build executors
	RegisterExecutor("build_beacon", executeBuildBeacon)
	RegisterExecutor("build_pulse", executeBuildPulse)
	RegisterExecutor("build_prelude", executeBuildPrelude)
	RegisterExecutor("build_module", executeBuildModule)
	// Profile executor
	RegisterExecutor("profile_create", executeProfileCreate)
	// Listener executor
	RegisterExecutor("listener_setup", executeListenerSetup)
	// Infrastructure executor (composite)
	RegisterExecutor("infrastructure_setup", executeInfrastructureSetup)
	// Certificate executors
	RegisterExecutor("cert_generate", executeCertGenerate)
	RegisterExecutor("cert_import", executeCertImport)
	// Config executors
	RegisterExecutor("github_config", executeGithubConfig)
	RegisterExecutor("notify_config", executeNotifyConfig)
}

// Helper functions for getting values from wizard result

func derefString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case *string:
		if val != nil {
			return *val, true
		}
	}
	return "", false
}

func getString(result *wizardfw.WizardResult, key string) string {
	if v, ok := result.Values[key]; ok {
		if s, ok := derefString(v); ok {
			return s
		}
	}
	return ""
}

func getInt(result *wizardfw.WizardResult, key string) int {
	v, ok := result.Values[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		if s, ok := derefString(v); ok {
			if i, err := strconv.Atoi(s); err == nil {
				return i
			}
		}
	}
	return 0
}

func getUint32(result *wizardfw.WizardResult, key string) uint32 {
	return uint32(getInt(result, key))
}

func getBool(result *wizardfw.WizardResult, key string) bool {
	v, ok := result.Values[key]
	if !ok {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case *bool:
		return val != nil && *val
	default:
		if s, ok := derefString(v); ok {
			return s == "true" || s == "yes" || s == "1"
		}
	}
	return false
}

func getFloat64(result *wizardfw.WizardResult, key string) float64 {
	v, ok := result.Values[key]
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		if s, ok := derefString(v); ok {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return f
			}
		}
	}
	return 0
}

func splitCommaSeparated(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func getStringSlice(result *wizardfw.WizardResult, key string) []string {
	v, ok := result.Values[key]
	if !ok {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return normalizeStringSlice(val)
	case *[]string:
		if val != nil {
			return normalizeStringSlice(*val)
		}
	case []interface{}:
		out := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					out = append(out, s)
				}
			}
		}
		return normalizeStringSlice(out)
	default:
		if s, ok := derefString(v); ok && s != "" {
			return splitCommaSeparated(s)
		}
	}
	return nil
}

// pipelineParams holds common parameters for pipeline creation
type pipelineParams struct {
	name       string
	listenerID string
	host       string
	port       uint32
	tls        *clientpb.TLS
}

// checkPortAvailable checks if a TCP port is available for binding
func checkPortAvailable(host string, port uint32) error {
	addr := fmt.Sprintf("%s:%d", host, port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("port %d is already in use on %s", port, host)
	}
	ln.Close()
	return nil
}

// extractPipelineParams extracts common pipeline parameters from wizard result
func extractPipelineParams(result *wizardfw.WizardResult, prefix string) (*pipelineParams, error) {
	listenerID := getString(result, "listener_id")
	if listenerID == "" {
		return nil, fmt.Errorf("listener_id is required")
	}

	host := getString(result, "host")
	if host == "" {
		host = "0.0.0.0"
	}

	port := getUint32(result, "port")
	if port == 0 {
		port = uint32(cryptography.RandomInRange(10240, 65535))
	}

	name := getString(result, "name")
	if name == "" {
		name = fmt.Sprintf("%s_%s_%d", prefix, listenerID, port)
	}

	var tls *clientpb.TLS
	if getBool(result, "tls") {
		tls = &clientpb.TLS{Enable: true}
	}

	return &pipelineParams{
		name:       name,
		listenerID: listenerID,
		host:       host,
		port:       port,
		tls:        tls,
	}, nil
}

// executeTCPPipeline executes the TCP pipeline wizard
func executeTCPPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	p, err := extractPipelineParams(result, "tcp")
	if err != nil {
		return err
	}
	// Check if port is available before registration
	if err := checkPortAvailable(p.host, p.port); err != nil {
		return fmt.Errorf("cannot create TCP pipeline: %w", err)
	}
	pipeline := &clientpb.Pipeline{
		Tls:        p.tls,
		Name:       p.name,
		ListenerId: p.listenerID,
		Parser:     consts.ImplantMalefic,
		Enable:     false,
		Body:       &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Name: p.name, Host: p.host, Port: p.port}},
	}
	return registerAndStartPipeline(con, pipeline, "TCP")
}

// executeHTTPPipeline executes the HTTP pipeline wizard
func executeHTTPPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	p, err := extractPipelineParams(result, "http")
	if err != nil {
		return err
	}
	// Check if port is available before registration
	if err := checkPortAvailable(p.host, p.port); err != nil {
		return fmt.Errorf("cannot create HTTP pipeline: %w", err)
	}
	pipeline := &clientpb.Pipeline{
		Tls:        p.tls,
		Name:       p.name,
		ListenerId: p.listenerID,
		Parser:     consts.ImplantMalefic,
		Enable:     false,
		Body:       &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Name: p.name, Host: p.host, Port: p.port}},
	}
	return registerAndStartPipeline(con, pipeline, "HTTP")
}

// registerAndStartPipeline handles the common register and start logic for all pipelines
func registerAndStartPipeline(con *core.Console, pipeline *clientpb.Pipeline, pipelineType string) error {
	if _, err := con.Rpc.RegisterPipeline(con.Context(), pipeline); err != nil {
		return fmt.Errorf("failed to register %s pipeline: %w", pipelineType, err)
	}
	con.Log.Importantf("%s Pipeline %s registered\n", pipelineType, pipeline.Name)

	if _, err := con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
		Pipeline:   pipeline,
	}); err != nil {
		return fmt.Errorf("failed to start %s pipeline: %w", pipelineType, err)
	}
	con.Log.Importantf("%s Pipeline %s started successfully\n", pipelineType, pipeline.Name)
	return nil
}

// executeBindPipeline executes the Bind pipeline wizard
func executeBindPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	listenerID := getString(result, "listener_id")
	if listenerID == "" {
		return fmt.Errorf("listener_id is required")
	}

	name := fmt.Sprintf("bind_%s", listenerID)
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Parser:     consts.ImplantMalefic,
		Enable:     false,
		Body:       &clientpb.Pipeline_Bind{Bind: &clientpb.BindPipeline{Name: name}},
	}
	return registerAndStartPipeline(con, pipeline, "Bind")
}

// executeREMPipeline executes the REM pipeline wizard
func executeREMPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	listenerID := getString(result, "listener_id")
	if listenerID == "" {
		return fmt.Errorf("listener_id is required")
	}

	name := getString(result, "name")
	if name == "" {
		name = fmt.Sprintf("rem_%s", listenerID)
	}

	console := getString(result, "console")
	if console == "" {
		console = "tcp://0.0.0.0:19966"
	}

	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Parser:     consts.ImplantMalefic,
		Secure:     &clientpb.Secure{Enable: getBool(result, "secure")},
		Enable:     false,
		Body:       &clientpb.Pipeline_Rem{Rem: &clientpb.REM{Name: name, Console: console}},
	}
	return registerAndStartPipeline(con, pipeline, "REM")
}

// executeBuildBeacon executes the build beacon wizard
func executeBuildBeacon(con *core.Console, result *wizardfw.WizardResult) error {
	return executeBuild(con, result, consts.CommandBuildBeacon)
}

// executeBuildPulse executes the build pulse wizard
func executeBuildPulse(con *core.Console, result *wizardfw.WizardResult) error {
	return executeBuild(con, result, consts.CommandBuildPulse)
}

// executeBuildPrelude executes the build prelude wizard
func executeBuildPrelude(con *core.Console, result *wizardfw.WizardResult) error {
	return executeBuild(con, result, consts.CommandBuildPrelude)
}

// executeBuildModule executes the build module wizard
func executeBuildModule(con *core.Console, result *wizardfw.WizardResult) error {
	modules := getStringSlice(result, "modules")
	thirdModules := getStringSlice(result, "third_modules")

	if len(modules) > 0 && len(thirdModules) > 0 {
		return fmt.Errorf("please choose either modules or third_modules, not both")
	}
	if len(thirdModules) > 0 {
		return executeBuild(con, result, consts.CommandBuild3rdModules)
	}
	return executeBuild(con, result, consts.CommandBuildModules)
}

// executeBuild is the common build execution logic
func executeBuild(con *core.Console, result *wizardfw.WizardResult, buildType string) error {
	target := getString(result, "target")
	if target == "" {
		return fmt.Errorf("target is required")
	}

	// Set source
	source := getString(result, "source")
	if source == "" {
		source = consts.ArtifactFromDocker
	}

	var buildConfig *clientpb.BuildConfig
	if buildType == consts.CommandBuildPrelude {
		autorunZipPath := getString(result, "autorun")
		if autorunZipPath == "" {
			return fmt.Errorf("autorun is required")
		}
		zipData, err := os.ReadFile(autorunZipPath)
		if err != nil {
			return fmt.Errorf("failed to read autorun zip: %w", err)
		}
		buildConfig, err = serverbuild.ProcessAutorunZipFromBytes(zipData)
		if err != nil {
			return fmt.Errorf("failed to process autorun zip: %w", err)
		}
		buildConfig.ProfileName = getString(result, "profile")
		buildConfig.Target = target
		buildConfig.BuildType = buildType
		buildConfig.Lib = getBool(result, "lib")
		buildConfig.Source = source
	} else {
		buildConfig = &clientpb.BuildConfig{
			ProfileName: getString(result, "profile"),
			Target:      target,
			BuildType:   buildType,
			Lib:         getBool(result, "lib"),
			Source:      source,
		}
	}

	// Build profile from wizard results if no implant.yaml provided by bundle (e.g., prelude builds without implant.yaml).
	if len(buildConfig.MaleficConfig) == 0 {
		profile, err := buildProfileFromWizard(con, result, buildType)
		if err != nil {
			return fmt.Errorf("failed to build profile: %w", err)
		}
		buildConfig.MaleficConfig, err = profile.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to encode profile: %w", err)
		}
	}

	// Check source availability
	resp, err := con.Rpc.CheckSource(con.Context(), buildConfig)
	if err != nil {
		return fmt.Errorf("failed to check source: %w", err)
	}
	buildConfig.Source = resp.Source

	// Handle artifact ID for pulse builds
	if buildType == consts.CommandBuildPulse {
		artifactID := getUint32(result, "beacon_artifact_id")
		if artifactID > 0 {
			buildConfig.ArtifactId = artifactID
		}
	}

	// Validate lib flag
	if err := validateLibFlag(buildConfig); err != nil {
		return err
	}

	artifact, err := con.Rpc.Build(con.Context(), buildConfig)
	if err != nil {
		return fmt.Errorf("build %s failed: %w", buildConfig.BuildType, err)
	}
	con.Log.Infof("Build started: %s (type: %s, target: %s, source: %s)\n",
		artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	return nil
}

// buildProfileFromWizard creates a ProfileConfig from wizard results
func buildProfileFromWizard(con *core.Console, result *wizardfw.WizardResult, buildType string) (*implanttypes.ProfileConfig, error) {
	profileName := getString(result, "profile")
	var profile *implanttypes.ProfileConfig
	var err error

	if profileName != "" {
		profilePB, err := con.Rpc.GetProfileByName(con.Context(), &clientpb.Profile{Name: profileName})
		if err != nil {
			return nil, fmt.Errorf("failed to get profile %q: %w", profileName, err)
		}
		profile, err = implanttypes.LoadProfile(profilePB.Content)
	} else {
		profile, err = implanttypes.LoadProfile(consts.DefaultProfile)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}

	ensureProfileSections(profile)

	// Set implant mode
	if buildType == consts.CommandBuildBeacon || buildType == consts.CommandBuildBind {
		profile.Implant.Mod = buildType
	}

	// Apply build-specific settings
	if buildType == consts.CommandBuildPulse {
		if err := applyPulseSettings(result, profile); err != nil {
			return nil, err
		}
		return profile, nil
	}

	// Apply basic settings
	applyBasicSettings(result, profile)

	// Parse and apply targets
	if addresses := getString(result, "addresses"); addresses != "" {
		targets, err := parseTargets(addresses)
		if err != nil {
			return nil, err
		}
		profile.Basic.Targets = targets
	}

	// Apply modules
	applyModuleSettings(result, profile)

	// Apply build settings
	applyBuildSettings(result, profile)

	return profile, nil
}

func ensureProfileSections(profile *implanttypes.ProfileConfig) {
	if profile.Basic == nil {
		profile.Basic = &implanttypes.BasicProfile{}
	}
	if profile.Pulse == nil {
		profile.Pulse = &implanttypes.PulseProfile{}
	}
	if profile.Implant == nil {
		profile.Implant = &implanttypes.ImplantProfile{}
	}
	if profile.Build == nil {
		profile.Build = &implanttypes.BuildProfile{}
	}
}

// applyBasicSettings applies basic profile settings from wizard result
func applyBasicSettings(result *wizardfw.WizardResult, profile *implanttypes.ProfileConfig) {
	// String settings
	if v := getString(result, "cron"); v != "" {
		profile.Basic.Cron = v
	}
	if v := getString(result, "encryption"); v != "" {
		profile.Basic.Encryption = v
	}
	if v := getString(result, "key"); v != "" {
		profile.Basic.Key = v
	}

	// Numeric settings
	if v := getFloat64(result, "jitter"); v > 0 {
		profile.Basic.Jitter = v
	}
	if v := getInt(result, "init_retry"); v > 0 {
		profile.Basic.InitRetry = v
	}
	if v := getInt(result, "server_retry"); v > 0 {
		profile.Basic.ServerRetry = v
	}
	if v := getInt(result, "global_retry"); v > 0 {
		profile.Basic.GlobalRetry = v
	}

	// Secure mode
	if getBool(result, "secure") {
		if profile.Basic.Secure == nil {
			profile.Basic.Secure = &implanttypes.SecureProfile{}
		}
		profile.Basic.Secure.Enable = true
	}

	// Proxy settings
	proxy, proxyUseEnv := getString(result, "proxy"), getBool(result, "proxy_use_env")
	if proxy != "" || proxyUseEnv {
		profile.Basic.Proxy = &implanttypes.ProxyProfile{URL: proxy, UseEnvProxy: proxyUseEnv}
	}

	// Guardrail settings
	applyGuardrailSettings(result, profile)
}

type pulseAddress struct {
	protocol string
	target   string
}

func parsePulseAddress(address string) (*pulseAddress, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("address is required")
	}

	if strings.Contains(address, "://") {
		u, err := url.Parse(address)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", address, err)
		}
		if u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return nil, fmt.Errorf("invalid address %q: only scheme://host[:port] is supported", address)
		}

		scheme := strings.ToLower(strings.TrimSpace(u.Scheme))
		switch scheme {
		case "http", "tcp":
		case "https":
			return nil, fmt.Errorf("pulse build only supports http:// or tcp:// addresses")
		default:
			return nil, fmt.Errorf("unsupported address scheme %q", u.Scheme)
		}

		if strings.Count(u.Host, ":") > 1 && !strings.HasPrefix(u.Host, "[") {
			return nil, fmt.Errorf("invalid address %q: IPv6 hosts must be in brackets, e.g. http://[::1]:80", address)
		}

		host := strings.TrimSpace(u.Hostname())
		if host == "" {
			return nil, fmt.Errorf("invalid address %q: missing host", address)
		}
		port := strings.TrimSpace(u.Port())
		if port == "" {
			if scheme == "tcp" {
				port = "5001"
			} else {
				port = "80"
			}
		}
		if err := validatePort(port); err != nil {
			return nil, err
		}

		target := net.JoinHostPort(host, port)
		if scheme == "tcp" {
			return &pulseAddress{protocol: consts.TCPPipeline, target: target}, nil
		}
		return &pulseAddress{protocol: consts.HTTPPipeline, target: target}, nil
	}

	if strings.ContainsAny(address, "/?#") {
		return nil, fmt.Errorf("invalid address %q: expected host[:port], http://host[:port], or tcp://host[:port]", address)
	}

	target, err := normalizeHostPort(address, "80")
	if err != nil {
		return nil, err
	}
	return &pulseAddress{protocol: consts.HTTPPipeline, target: target}, nil
}

// applyPulseSettings applies pulse profile settings from wizard result
func applyPulseSettings(result *wizardfw.WizardResult, profile *implanttypes.ProfileConfig) error {
	if profile.Pulse == nil {
		profile.Pulse = &implanttypes.PulseProfile{}
	}

	parsed, err := parsePulseAddress(getString(result, "address"))
	if err != nil {
		return err
	}
	profile.Pulse.Protocol = parsed.protocol
	profile.Pulse.Target = parsed.target

	if profile.Pulse.Protocol == consts.HTTPPipeline {
		if profile.Pulse.Http == nil {
			profile.Pulse.Http = &implanttypes.HttpProfile{}
		}
		profile.Pulse.Http.Method = "POST"
		profile.Pulse.Http.Version = "1.1"
		profile.Pulse.Http.Host = parsed.target
		if profile.Pulse.Http.Headers == nil {
			profile.Pulse.Http.Headers = map[string]string{}
		}
		profile.Pulse.Http.Headers["Host"] = parsed.target
	}

	if profile.Pulse.Http != nil {
		if v := strings.TrimSpace(getString(result, "path")); v != "" {
			profile.Pulse.Http.Path = v
		}
		if v := strings.TrimSpace(getString(result, "user_agent")); v != "" {
			if profile.Pulse.Http.Headers == nil {
				profile.Pulse.Http.Headers = map[string]string{}
			}
			profile.Pulse.Http.Headers["User-Agent"] = v
		}
	}

	if artifactID := getUint32(result, "beacon_artifact_id"); artifactID != 0 {
		if profile.Pulse.Flags == nil {
			profile.Pulse.Flags = &implanttypes.PulseFlags{}
		}
		profile.Pulse.Flags.ArtifactID = artifactID
	}

	return nil
}

// applyGuardrailSettings applies guardrail settings from wizard result
func applyGuardrailSettings(result *wizardfw.WizardResult, profile *implanttypes.ProfileConfig) {
	ips := splitCommaSeparated(getString(result, "guardrail_ips"))
	users := splitCommaSeparated(getString(result, "guardrail_users"))
	servers := splitCommaSeparated(getString(result, "guardrail_servers"))
	domains := splitCommaSeparated(getString(result, "guardrail_domains"))
	if len(ips) == 0 && len(users) == 0 && len(servers) == 0 && len(domains) == 0 {
		return
	}

	if profile.Basic.Guardrail == nil {
		profile.Basic.Guardrail = &implanttypes.GuardrailProfile{}
	}
	profile.Basic.Guardrail.Enable = true
	profile.Basic.Guardrail.RequireAll = true

	if len(ips) > 0 {
		profile.Basic.Guardrail.IPAddresses = ips
	}
	if len(users) > 0 {
		profile.Basic.Guardrail.Usernames = users
	}
	if len(servers) > 0 {
		profile.Basic.Guardrail.ServerNames = servers
	}
	if len(domains) > 0 {
		profile.Basic.Guardrail.Domains = domains
	}
}

// addressScheme defines how to parse a URL scheme into a Target
type addressScheme struct {
	prefix      string
	defaultPort string
	configure   func(host string, target *implanttypes.Target)
}

var addressSchemes = []addressScheme{
	{"http://", "80", configureHTTP},
	{"https://", "443", configureHTTPS},
	{"tcp+tls://", "5001", configureTCPTLS},
	{"tcp://", "5001", configureTCP},
}

func configureHTTP(host string, target *implanttypes.Target) {
	target.Http = defaultHTTPProfile()
}

func configureHTTPS(host string, target *implanttypes.Target) {
	target.Http = defaultHTTPProfile()
	target.TLS = &implanttypes.TLSProfile{
		Enable:           true,
		SNI:              host,
		SkipVerification: true,
	}
}

func configureTCP(host string, target *implanttypes.Target) {
	target.TCP = &implanttypes.TCPProfile{}
}

func configureTCPTLS(host string, target *implanttypes.Target) {
	target.TCP = &implanttypes.TCPProfile{}
	target.TLS = &implanttypes.TLSProfile{
		Enable:           true,
		SNI:              host,
		SkipVerification: true,
	}
}

func defaultHTTPProfile() *implanttypes.HttpProfile {
	return &implanttypes.HttpProfile{
		Method:  "POST",
		Path:    "/",
		Version: "1.1",
		Headers: map[string]string{
			"User-Agent":   uarand.GetRandom(),
			"Content-Type": "application/octet-stream",
		},
	}
}

// parseTargets parses comma-separated addresses into Target slice
func parseTargets(addresses string) ([]implanttypes.Target, error) {
	var targets []implanttypes.Target
	for _, raw := range strings.Split(addresses, ",") {
		addr := strings.TrimSpace(raw)
		if addr == "" {
			continue
		}
		target, err := parseAddress(addr)
		if err != nil {
			return nil, err
		}
		targets = append(targets, *target)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("no valid targets found in addresses")
	}
	return targets, nil
}

// parseAddress parses a single address into a Target
func parseAddress(address string) (*implanttypes.Target, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("address is empty")
	}

	if strings.Contains(address, "://") {
		u, err := url.Parse(address)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", address, err)
		}
		if u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return nil, fmt.Errorf("invalid address %q: only scheme://host[:port] is supported", address)
		}
		if strings.Count(u.Host, ":") > 1 && !strings.HasPrefix(u.Host, "[") {
			return nil, fmt.Errorf("invalid address %q: IPv6 hosts must be in brackets, e.g. tcp://[::1]:5001", address)
		}

		scheme := strings.ToLower(strings.TrimSpace(u.Scheme))

		// Find matching scheme config in a single pass
		var matched *addressScheme
		for i := range addressSchemes {
			if strings.TrimSuffix(addressSchemes[i].prefix, "://") == scheme {
				matched = &addressSchemes[i]
				break
			}
		}
		if matched == nil {
			return nil, fmt.Errorf("unsupported address scheme %q", u.Scheme)
		}

		host := strings.TrimSpace(u.Hostname())
		if host == "" {
			return nil, fmt.Errorf("invalid address %q: missing host", address)
		}
		port := strings.TrimSpace(u.Port())
		if port == "" {
			port = matched.defaultPort
		}
		if err := validatePort(port); err != nil {
			return nil, err
		}

		target := &implanttypes.Target{Address: net.JoinHostPort(host, port)}
		matched.configure(host, target)
		return target, nil
	}

	if strings.ContainsAny(address, "/?#") {
		return nil, fmt.Errorf("invalid address %q: expected host[:port] or scheme://host[:port]", address)
	}

	addr, err := normalizeHostPort(address, "5001")
	if err != nil {
		return nil, err
	}
	target := &implanttypes.Target{
		Address: addr,
		TCP:     &implanttypes.TCPProfile{},
	}
	return target, nil
}

func validatePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port: %q", port)
	}
	return nil
}

func normalizeHostPort(addr string, defaultPort string) (string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", fmt.Errorf("address is empty")
	}
	if strings.ContainsAny(addr, "/?#") {
		return "", fmt.Errorf("invalid address %q: expected host[:port]", addr)
	}

	port := defaultPort

	switch {
	case strings.HasPrefix(addr, "["):
		if strings.Contains(addr, "]") && !strings.Contains(addr, "]:") {
			addr = addr + ":" + port
		}
		h, p, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			return "", fmt.Errorf("invalid address %q: %w", addr, splitErr)
		}
		if strings.TrimSpace(h) == "" {
			return "", fmt.Errorf("invalid address %q: missing host", addr)
		}
		if err := validatePort(p); err != nil {
			return "", err
		}
		return net.JoinHostPort(h, p), nil

	case strings.Count(addr, ":") == 0:
		if err := validatePort(port); err != nil {
			return "", err
		}
		return net.JoinHostPort(addr, port), nil

	case strings.Count(addr, ":") == 1:
		h, p, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			return "", fmt.Errorf("invalid address %q: %w", addr, splitErr)
		}
		if strings.TrimSpace(h) == "" {
			return "", fmt.Errorf("invalid address %q: missing host", addr)
		}
		if err := validatePort(p); err != nil {
			return "", err
		}
		return net.JoinHostPort(h, p), nil

	default:
		// Bare IPv6 address without brackets
		ipPart := addr
		if i := strings.LastIndex(ipPart, "%"); i != -1 {
			ipPart = ipPart[:i]
		}
		if net.ParseIP(ipPart) == nil {
			return "", fmt.Errorf("invalid IPv6 address %q (use [ipv6]:port)", addr)
		}
		if err := validatePort(port); err != nil {
			return "", err
		}
		return net.JoinHostPort(addr, port), nil
	}
}

// applyModuleSettings applies module settings from wizard result
func applyModuleSettings(result *wizardfw.WizardResult, profile *implanttypes.ProfileConfig) {
	if modules := getStringSlice(result, "modules"); len(modules) > 0 {
		profile.Implant.Modules = modules
	}
	if thirdModules := getStringSlice(result, "third_modules"); len(thirdModules) > 0 {
		profile.Implant.ThirdModules = thirdModules
		profile.Implant.Enable3rd = true
	}
}

// applyBuildSettings applies build settings from wizard result
func applyBuildSettings(result *wizardfw.WizardResult, profile *implanttypes.ProfileConfig) {
	if getBool(result, "ollvm") {
		if profile.Build == nil {
			profile.Build = &implanttypes.BuildProfile{}
		}
		profile.Build.OLLVM = &implanttypes.OLLVMProfile{
			Enable: true, BCFObf: true, SplitObf: true, SubObf: true, FCO: true, ConstEnc: true,
		}
	}
	if getBool(result, "anti_sandbox") {
		if profile.Implant == nil {
			profile.Implant = &implanttypes.ImplantProfile{}
		}
		if profile.Implant.Anti == nil {
			profile.Implant.Anti = &implanttypes.AntiProfile{}
		}
		profile.Implant.Anti.Sandbox = true
	}
}

// validateLibFlag validates the lib flag based on build type and target
func validateLibFlag(buildConfig *clientpb.BuildConfig) error {
	target, ok := consts.GetBuildTarget(buildConfig.Target)
	if !ok {
		return fmt.Errorf("invalid target: %s", buildConfig.Target)
	}

	switch buildConfig.BuildType {
	case consts.CommandBuildModules, consts.CommandBuild3rdModules:
		if target.OS != consts.Windows {
			return fmt.Errorf("modules build only supports Windows targets")
		}
		buildConfig.Lib = true
	case consts.CommandBuildPrelude:
		buildConfig.Lib = false
	case consts.CommandBuildPulse:
		if target.OS != consts.Windows {
			return fmt.Errorf("pulse build only supports Windows targets")
		}
		buildConfig.Lib = false
	}
	return nil
}

// executeProfileCreate creates a new profile from wizard results
func executeProfileCreate(con *core.Console, result *wizardfw.WizardResult) error {
	name := getString(result, "name")
	if name == "" {
		return fmt.Errorf("profile name is required")
	}

	pipelineID := getString(result, "pipeline")
	if pipelineID == "" {
		return fmt.Errorf("pipeline is required")
	}

	implantType := getString(result, "type")
	modules := getStringSlice(result, "modules")

	// Build profile params
	var params implanttypes.ProfileParams
	if len(modules) > 0 {
		params.Modules = strings.Join(modules, ",")
	}

	profile := &clientpb.Profile{
		Name:       name,
		PipelineId: pipelineID,
		Params:     params.String(),
	}

	// Note: implantType is stored in profile content, not params
	_ = implantType

	_, err := con.Rpc.NewProfile(con.Context(), profile)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	con.Log.Importantf("Profile '%s' created successfully for pipeline '%s'\n", name, pipelineID)
	return nil
}

// executeListenerSetup displays listener setup instructions
// Note: Listeners are typically configured via server config file or listener binary
func executeListenerSetup(con *core.Console, result *wizardfw.WizardResult) error {
	name := getString(result, "name")
	host := getString(result, "host")
	protocol := getString(result, "protocol")
	port := getInt(result, "port")
	tls := getBool(result, "tls")

	con.Log.Importantf("Listener Configuration:\n")
	con.Log.Infof("  Name:     %s\n", name)
	con.Log.Infof("  Host:     %s\n", host)
	con.Log.Infof("  Protocol: %s\n", protocol)
	con.Log.Infof("  Port:     %d\n", port)
	con.Log.Infof("  TLS:      %v\n", tls)
	con.Log.Warnf("\nNote: Listeners must be started separately using the listener binary.\n")
	con.Log.Infof("Example: ./listener --config listener.yaml\n")

	return nil
}

// executeInfrastructureSetup creates listener config, pipeline, and profile
func executeInfrastructureSetup(con *core.Console, result *wizardfw.WizardResult) error {
	// Step 1: Display listener configuration (listeners need to be started separately)
	listenerName := getString(result, "listener_name")
	listenerHost := getString(result, "listener_host")
	listenerProtocol := getString(result, "listener_protocol")
	listenerPort := getInt(result, "listener_port")
	listenerTLS := getBool(result, "listener_tls")

	con.Log.Importantf("=== Infrastructure Setup ===\n\n")
	con.Log.Infof("[1/3] Listener Configuration:\n")
	con.Log.Infof("  Name:     %s\n", listenerName)
	con.Log.Infof("  Host:     %s\n", listenerHost)
	con.Log.Infof("  Protocol: %s\n", listenerProtocol)
	con.Log.Infof("  Port:     %d\n", listenerPort)
	con.Log.Infof("  TLS:      %v\n", listenerTLS)
	con.Log.Warnf("  Note: Start listener separately with: ./listener --config listener.yaml\n\n")

	// Step 2: Create pipeline
	pipelineType := getString(result, "pipeline_type")
	pipelineName := getString(result, "pipeline_name")
	pipelineHost := getString(result, "pipeline_host")
	pipelinePort := getUint32(result, "pipeline_port")
	pipelineTLS := getBool(result, "pipeline_tls")

	if pipelineName == "" {
		pipelineName = fmt.Sprintf("%s_%s_%d", pipelineType, listenerName, pipelinePort)
	}

	con.Log.Infof("[2/3] Creating Pipeline '%s'...\n", pipelineName)

	// Check port availability
	if err := checkPortAvailable(pipelineHost, pipelinePort); err != nil {
		return fmt.Errorf("cannot create pipeline: %w", err)
	}

	var tls *clientpb.TLS
	if pipelineTLS {
		tls = &clientpb.TLS{Enable: true}
	}

	var pipeline *clientpb.Pipeline
	if pipelineType == "http" {
		pipeline = &clientpb.Pipeline{
			Tls:        tls,
			Name:       pipelineName,
			ListenerId: listenerName,
			Parser:     consts.ImplantMalefic,
			Enable:     false,
			Body:       &clientpb.Pipeline_Http{Http: &clientpb.HTTPPipeline{Name: pipelineName, Host: pipelineHost, Port: pipelinePort}},
		}
	} else {
		pipeline = &clientpb.Pipeline{
			Tls:        tls,
			Name:       pipelineName,
			ListenerId: listenerName,
			Parser:     consts.ImplantMalefic,
			Enable:     false,
			Body:       &clientpb.Pipeline_Tcp{Tcp: &clientpb.TCPPipeline{Name: pipelineName, Host: pipelineHost, Port: pipelinePort}},
		}
	}

	if _, err := con.Rpc.RegisterPipeline(con.Context(), pipeline); err != nil {
		return fmt.Errorf("failed to register pipeline: %w", err)
	}
	con.Log.Importantf("  Pipeline '%s' registered\n", pipelineName)

	if _, err := con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipeline.Name,
		ListenerId: pipeline.ListenerId,
		Pipeline:   pipeline,
	}); err != nil {
		return fmt.Errorf("failed to start pipeline: %w", err)
	}
	con.Log.Importantf("  Pipeline '%s' started\n\n", pipelineName)

	// Step 3: Create profile
	profileName := getString(result, "profile_name")
	implantType := getString(result, "implant_type")
	modules := getStringSlice(result, "modules")

	con.Log.Infof("[3/3] Creating Profile '%s'...\n", profileName)

	var params implanttypes.ProfileParams
	if len(modules) > 0 {
		params.Modules = strings.Join(modules, ",")
	}

	// Note: implantType is stored in profile content, not params
	_ = implantType

	profile := &clientpb.Profile{
		Name:       profileName,
		PipelineId: pipelineName,
		Params:     params.String(),
	}

	if _, err := con.Rpc.NewProfile(con.Context(), profile); err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}
	con.Log.Importantf("  Profile '%s' created for pipeline '%s'\n\n", profileName, pipelineName)

	con.Log.Importantf("=== Infrastructure Setup Complete ===\n")
	con.Log.Infof("Next steps:\n")
	con.Log.Infof("  1. Start listener: ./listener --config listener.yaml\n")
	con.Log.Infof("  2. Build implant:  wizard build beacon\n")

	return nil
}

// executeCertGenerate generates a self-signed certificate
func executeCertGenerate(con *core.Console, result *wizardfw.WizardResult) error {
	cn := getString(result, "cn")
	if cn == "" {
		return fmt.Errorf("Common Name (CN) is required")
	}

	certSubject := &clientpb.CertificateSubject{
		Cn:       cn,
		O:        getString(result, "o"),
		C:        getString(result, "c"),
		L:        getString(result, "l"),
		Ou:       getString(result, "ou"),
		St:       getString(result, "st"),
		Validity: fmt.Sprintf("%d", getInt(result, "validity")),
	}

	_, err := con.Rpc.GenerateSelfCert(con.Context(), &clientpb.Pipeline{
		Tls: &clientpb.TLS{
			CertSubject: certSubject,
			Acme:        false,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to generate certificate: %w", err)
	}

	con.Log.Importantf("Self-signed certificate generated successfully\n")
	con.Log.Infof("  CN: %s\n", cn)
	if certSubject.O != "" {
		con.Log.Infof("  O:  %s\n", certSubject.O)
	}
	if certSubject.Validity != "" && certSubject.Validity != "0" {
		con.Log.Infof("  Validity: %s days\n", certSubject.Validity)
	}

	return nil
}

// executeCertImport imports an existing certificate
func executeCertImport(con *core.Console, result *wizardfw.WizardResult) error {
	certPath := getString(result, "cert")
	keyPath := getString(result, "key")

	if certPath == "" || keyPath == "" {
		return fmt.Errorf("certificate and key files are required")
	}

	// Read certificate file
	certData, err := cryptography.ProcessPEM(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Read key file
	keyData, err := cryptography.ProcessPEM(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Read CA certificate if provided
	var caCert *clientpb.Cert
	caPath := getString(result, "ca_cert")
	if caPath != "" {
		caData, err := cryptography.ProcessPEM(caPath)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate file: %w", err)
		}
		caCert = &clientpb.Cert{
			Cert: caData,
		}
	}

	tls := &clientpb.TLS{
		Cert: &clientpb.Cert{
			Cert: certData,
			Key:  keyData,
		},
		Ca: caCert,
	}

	_, err = con.Rpc.GenerateSelfCert(con.Context(), &clientpb.Pipeline{
		Tls: tls,
	})
	if err != nil {
		return fmt.Errorf("failed to import certificate: %w", err)
	}

	con.Log.Importantf("Certificate imported successfully\n")
	con.Log.Infof("  Certificate: %s\n", certPath)
	con.Log.Infof("  Key:         %s\n", keyPath)
	if caPath != "" {
		con.Log.Infof("  CA Cert:     %s\n", caPath)
	}

	return nil
}

// executeGithubConfig configures GitHub Actions build
func executeGithubConfig(con *core.Console, result *wizardfw.WizardResult) error {
	owner := getString(result, "owner")
	repo := getString(result, "repo")
	token := getString(result, "token")

	if owner == "" || repo == "" || token == "" {
		return fmt.Errorf("owner, repo, and token are required")
	}

	workflowFile := getString(result, "workflow_file")

	githubConfig := &clientpb.GithubActionBuildConfig{
		Owner:      owner,
		Repo:       repo,
		Token:      token,
		WorkflowId: workflowFile,
	}

	_, err := con.Rpc.UpdateGithubConfig(con.Context(), githubConfig)
	if err != nil {
		return fmt.Errorf("failed to update GitHub config: %w", err)
	}

	con.Log.Importantf("GitHub Actions configuration updated successfully\n")
	con.Log.Infof("  Owner:    %s\n", owner)
	con.Log.Infof("  Repo:     %s\n", repo)
	con.Log.Infof("  Token:    %s***\n", token[:minInt(4, len(token))])
	if workflowFile != "" {
		con.Log.Infof("  Workflow: %s\n", workflowFile)
	}

	return nil
}

// executeNotifyConfig configures notification channels
func executeNotifyConfig(con *core.Console, result *wizardfw.WizardResult) error {
	notify := &clientpb.Notify{}
	hasConfig := false

	// Telegram
	if getBool(result, "telegram_enable") {
		notify.TelegramEnable = true
		notify.TelegramApiKey = getString(result, "telegram_token")
		if chatID := getString(result, "telegram_chat_id"); chatID != "" {
			// Parse chat ID as int64
			var id int64
			fmt.Sscanf(chatID, "%d", &id)
			notify.TelegramChatId = id
		}
		hasConfig = true
	}

	// DingTalk
	if getBool(result, "dingtalk_enable") {
		notify.DingtalkEnable = true
		notify.DingtalkToken = getString(result, "dingtalk_token")
		notify.DingtalkSecret = getString(result, "dingtalk_secret")
		hasConfig = true
	}

	// Lark
	if getBool(result, "lark_enable") {
		notify.LarkEnable = true
		notify.LarkWebhookUrl = getString(result, "lark_webhook")
		hasConfig = true
	}

	// ServerChan
	if getBool(result, "serverchan_enable") {
		notify.ServerchanEnable = true
		notify.ServerchanUrl = getString(result, "serverchan_url")
		hasConfig = true
	}

	// PushPlus
	if getBool(result, "pushplus_enable") {
		notify.PushplusEnable = true
		notify.PushplusToken = getString(result, "pushplus_token")
		notify.PushplusTopic = getString(result, "pushplus_topic")
		hasConfig = true
	}

	if !hasConfig {
		con.Log.Warnf("No notification channels enabled\n")
		return nil
	}

	_, err := con.Rpc.UpdateNotifyConfig(con.Context(), notify)
	if err != nil {
		return fmt.Errorf("failed to update notification config: %w", err)
	}

	con.Log.Importantf("Notification configuration updated successfully\n")
	if notify.TelegramEnable {
		con.Log.Infof("  Telegram: enabled\n")
	}
	if notify.DingtalkEnable {
		con.Log.Infof("  DingTalk: enabled\n")
	}
	if notify.LarkEnable {
		con.Log.Infof("  Lark:     enabled\n")
	}
	if notify.ServerchanEnable {
		con.Log.Infof("  ServerChan: enabled\n")
	}
	if notify.PushplusEnable {
		con.Log.Infof("  PushPlus: enabled\n")
	}

	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
