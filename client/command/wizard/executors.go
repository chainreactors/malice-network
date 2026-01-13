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
	RegisterExecutor("tcp_pipeline", executeTCPPipeline)
	RegisterExecutor("http_pipeline", executeHTTPPipeline)
	RegisterExecutor("bind_pipeline", executeBindPipeline)
	RegisterExecutor("rem_pipeline", executeREMPipeline)
	// Build executors
	RegisterExecutor("build_beacon", executeBuildBeacon)
	RegisterExecutor("build_pulse", executeBuildPulse)
	RegisterExecutor("build_prelude", executeBuildPrelude)
	RegisterExecutor("build_module", executeBuildModule)
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

	// Execute build asynchronously
	go func() {
		artifact, err := con.Rpc.Build(con.Context(), buildConfig)
		if err != nil {
			con.Log.Errorf("Build %s failed: %v\n", buildConfig.BuildType, err)
			return
		}
		con.Log.Infof("Build started: %s (type: %s, target: %s, source: %s)\n",
			artifact.Name, artifact.Type, artifact.Target, artifact.Source)
	}()

	con.Log.Importantf("Build task submitted (type: %s, target: %s)\n", buildType, target)
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

	host, target, err := normalizeHostPort(address, "80")
	if err != nil {
		return nil, err
	}
	_ = host
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
		target := &implanttypes.Target{}
		var defaultPort string
		for _, candidate := range addressSchemes {
			if strings.TrimSuffix(candidate.prefix, "://") == scheme {
				defaultPort = candidate.defaultPort
				break
			}
		}
		if defaultPort == "" {
			return nil, fmt.Errorf("unsupported address scheme %q", u.Scheme)
		}

		host := strings.TrimSpace(u.Hostname())
		if host == "" {
			return nil, fmt.Errorf("invalid address %q: missing host", address)
		}
		port := strings.TrimSpace(u.Port())
		if port == "" {
			port = defaultPort
		}
		if err := validatePort(port); err != nil {
			return nil, err
		}
		target.Address = net.JoinHostPort(host, port)

		for _, schemeConfig := range addressSchemes {
			if strings.TrimSuffix(schemeConfig.prefix, "://") == scheme {
				schemeConfig.configure(host, target)
				return target, nil
			}
		}
		return nil, fmt.Errorf("unsupported address scheme %q", u.Scheme)
	}

	if strings.ContainsAny(address, "/?#") {
		return nil, fmt.Errorf("invalid address %q: expected host[:port] or scheme://host[:port]", address)
	}

	host, addr, err := normalizeHostPort(address, "5001")
	if err != nil {
		return nil, err
	}
	target := &implanttypes.Target{
		Address: addr,
		TCP:     &implanttypes.TCPProfile{},
	}
	_ = host
	return target, nil
}

func validatePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("invalid port: %q", port)
	}
	return nil
}

func normalizeHostPort(addr string, defaultPort string) (host string, normalized string, err error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", "", fmt.Errorf("address is empty")
	}
	if strings.ContainsAny(addr, "/?#") {
		return "", "", fmt.Errorf("invalid address %q: expected host[:port]", addr)
	}

	port := defaultPort

	switch {
	case strings.HasPrefix(addr, "["):
		if strings.Contains(addr, "]") && !strings.Contains(addr, "]:") {
			addr = addr + ":" + port
		}
		h, p, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			return "", "", fmt.Errorf("invalid address %q: %w", addr, splitErr)
		}
		if strings.TrimSpace(h) == "" {
			return "", "", fmt.Errorf("invalid address %q: missing host", addr)
		}
		if err := validatePort(p); err != nil {
			return "", "", err
		}
		return h, net.JoinHostPort(h, p), nil

	case strings.Count(addr, ":") == 0:
		if err := validatePort(port); err != nil {
			return "", "", err
		}
		return addr, net.JoinHostPort(addr, port), nil

	case strings.Count(addr, ":") == 1:
		h, p, splitErr := net.SplitHostPort(addr)
		if splitErr != nil {
			return "", "", fmt.Errorf("invalid address %q: %w", addr, splitErr)
		}
		if strings.TrimSpace(h) == "" {
			return "", "", fmt.Errorf("invalid address %q: missing host", addr)
		}
		if err := validatePort(p); err != nil {
			return "", "", err
		}
		return h, net.JoinHostPort(h, p), nil

	default:
		host = addr
		ipPart := host
		if i := strings.LastIndex(ipPart, "%"); i != -1 {
			ipPart = ipPart[:i]
		}
		if net.ParseIP(ipPart) == nil {
			return "", "", fmt.Errorf("invalid IPv6 address %q (use [ipv6]:port)", addr)
		}
		if err := validatePort(port); err != nil {
			return "", "", err
		}
		return host, net.JoinHostPort(host, port), nil
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
