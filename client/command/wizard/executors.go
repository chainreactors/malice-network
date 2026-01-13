package wizard

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	wizardfw "github.com/chainreactors/malice-network/client/wizard"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/implanttypes"
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

func getString(result *wizardfw.WizardResult, key string) string {
	if v, ok := result.Values[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case *string:
			if val != nil {
				return *val
			}
		}
	}
	return ""
}

func getInt(result *wizardfw.WizardResult, key string) int {
	if v, ok := result.Values[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		case string:
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		case *string:
			if val != nil {
				if i, err := strconv.Atoi(*val); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

func getUint32(result *wizardfw.WizardResult, key string) uint32 {
	return uint32(getInt(result, key))
}

func getBool(result *wizardfw.WizardResult, key string) bool {
	if v, ok := result.Values[key]; ok {
		switch val := v.(type) {
		case bool:
			return val
		case *bool:
			if val != nil {
				return *val
			}
		case string:
			return val == "true" || val == "yes" || val == "1"
		case *string:
			if val != nil {
				s := *val
				return s == "true" || s == "yes" || s == "1"
			}
		}
	}
	return false
}

func getFloat64(result *wizardfw.WizardResult, key string) float64 {
	if v, ok := result.Values[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case float32:
			return float64(val)
		case int:
			return float64(val)
		case int64:
			return float64(val)
		case string:
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		case *string:
			if val != nil {
				if f, err := strconv.ParseFloat(*val, 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

func getStringSlice(result *wizardfw.WizardResult, key string) []string {
	if v, ok := result.Values[key]; ok {
		switch val := v.(type) {
		case []string:
			return val
		case *[]string:
			if val != nil {
				return *val
			}
		case []interface{}:
			result := make([]string, len(val))
			for i, item := range val {
				if s, ok := item.(string); ok {
					result[i] = s
				}
			}
			return result
		case string:
			if val != "" {
				return strings.Split(val, ",")
			}
		case *string:
			if val != nil && *val != "" {
				return strings.Split(*val, ",")
			}
		}
	}
	return nil
}

// executeTCPPipeline executes the TCP pipeline wizard
func executeTCPPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	name := getString(result, "name")
	listenerID := getString(result, "listener_id")
	host := getString(result, "host")
	port := getUint32(result, "port")
	tlsEnabled := getBool(result, "tls")

	// Validate required fields
	if listenerID == "" {
		return fmt.Errorf("listener_id is required")
	}

	// Generate name if not provided
	if name == "" {
		if port == 0 {
			port = uint32(cryptography.RandomInRange(10240, 65535))
		}
		name = fmt.Sprintf("tcp_%s_%d", listenerID, port)
	}

	// Set defaults
	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = uint32(cryptography.RandomInRange(10240, 65535))
	}

	// Build TLS config
	var tls *clientpb.TLS
	if tlsEnabled {
		tls = &clientpb.TLS{Enable: true}
	}

	pipeline := &clientpb.Pipeline{
		Tls:        tls,
		Name:       name,
		ListenerId: listenerID,
		Parser:     consts.ImplantMalefic,
		Enable:     false,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name: name,
				Host: host,
				Port: port,
			},
		},
	}

	// Register pipeline
	_, err := con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return fmt.Errorf("failed to register TCP pipeline: %w", err)
	}

	con.Log.Importantf("TCP Pipeline %s registered\n", name)

	// Start pipeline
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
		Pipeline:   pipeline,
	})
	if err != nil {
		return fmt.Errorf("failed to start TCP pipeline: %w", err)
	}

	con.Log.Importantf("TCP Pipeline %s started successfully\n", name)
	return nil
}

// executeHTTPPipeline executes the HTTP pipeline wizard
func executeHTTPPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	name := getString(result, "name")
	listenerID := getString(result, "listener_id")
	host := getString(result, "host")
	port := getUint32(result, "port")
	tlsEnabled := getBool(result, "tls")

	// Validate required fields
	if listenerID == "" {
		return fmt.Errorf("listener_id is required")
	}

	// Generate name if not provided
	if name == "" {
		if port == 0 {
			port = uint32(cryptography.RandomInRange(10240, 65535))
		}
		name = fmt.Sprintf("http_%s_%d", listenerID, port)
	}

	// Set defaults
	if host == "" {
		host = "0.0.0.0"
	}
	if port == 0 {
		port = uint32(cryptography.RandomInRange(10240, 65535))
	}

	// Build TLS config
	var tls *clientpb.TLS
	if tlsEnabled {
		tls = &clientpb.TLS{Enable: true}
	}

	pipeline := &clientpb.Pipeline{
		Tls:        tls,
		Name:       name,
		ListenerId: listenerID,
		Parser:     consts.ImplantMalefic,
		Enable:     false,
		Body: &clientpb.Pipeline_Http{
			Http: &clientpb.HTTPPipeline{
				Name: name,
				Host: host,
				Port: port,
			},
		},
	}

	// Register pipeline
	_, err := con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return fmt.Errorf("failed to register HTTP pipeline: %w", err)
	}

	con.Log.Importantf("HTTP Pipeline %s registered\n", name)

	// Start pipeline
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
		Pipeline:   pipeline,
	})
	if err != nil {
		return fmt.Errorf("failed to start HTTP pipeline: %w", err)
	}

	con.Log.Importantf("HTTP Pipeline %s started successfully\n", name)
	return nil
}

// executeBindPipeline executes the Bind pipeline wizard
func executeBindPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	listenerID := getString(result, "listener_id")

	// Validate required fields
	if listenerID == "" {
		return fmt.Errorf("listener_id is required")
	}

	name := fmt.Sprintf("bind_%s", listenerID)

	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Parser:     consts.ImplantMalefic,
		Enable:     false,
		Body: &clientpb.Pipeline_Bind{
			Bind: &clientpb.BindPipeline{
				Name: name,
			},
		},
	}

	// Register pipeline
	_, err := con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return fmt.Errorf("failed to register Bind pipeline: %w", err)
	}

	con.Log.Importantf("Bind Pipeline %s registered\n", name)

	// Start pipeline
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
		Pipeline:   pipeline,
	})
	if err != nil {
		return fmt.Errorf("failed to start Bind pipeline: %w", err)
	}

	con.Log.Importantf("Bind Pipeline %s started successfully\n", name)
	return nil
}

// executeREMPipeline executes the REM pipeline wizard
func executeREMPipeline(con *core.Console, result *wizardfw.WizardResult) error {
	name := getString(result, "name")
	listenerID := getString(result, "listener_id")
	console := getString(result, "console")
	secure := getBool(result, "secure")

	// Validate required fields
	if listenerID == "" {
		return fmt.Errorf("listener_id is required")
	}

	// Generate name if not provided
	if name == "" {
		name = fmt.Sprintf("rem_%s", listenerID)
	}

	// Default console URL
	if console == "" {
		console = "tcp://0.0.0.0:19966"
	}

	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Parser:     consts.ImplantMalefic,
		Secure:     &clientpb.Secure{Enable: secure},
		Enable:     false,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:    name,
				Console: console,
			},
		},
	}

	// Register pipeline
	_, err := con.Rpc.RegisterPipeline(con.Context(), pipeline)
	if err != nil {
		return fmt.Errorf("failed to register REM pipeline: %w", err)
	}

	con.Log.Importantf("REM Pipeline %s registered\n", name)

	// Start pipeline
	_, err = con.Rpc.StartPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       name,
		ListenerId: listenerID,
		Pipeline:   pipeline,
	})
	if err != nil {
		return fmt.Errorf("failed to start REM pipeline: %w", err)
	}

	con.Log.Importantf("REM Pipeline %s started successfully\n", name)
	return nil
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
	return executeBuild(con, result, consts.CommandBuildModules)
}

// executeBuild is the common build execution logic
func executeBuild(con *core.Console, result *wizardfw.WizardResult, buildType string) error {
	target := getString(result, "target")
	if target == "" {
		return fmt.Errorf("target is required")
	}

	// Build profile from wizard results
	profile, err := buildProfileFromWizard(result, buildType)
	if err != nil {
		return fmt.Errorf("failed to build profile: %w", err)
	}

	// Create build config
	buildConfig := &clientpb.BuildConfig{
		ProfileName: getString(result, "profile"),
		Target:      target,
		BuildType:   buildType,
		Lib:         getBool(result, "lib"),
	}

	// Set source
	source := getString(result, "source")
	if source == "" {
		source = consts.ArtifactFromDocker
	}
	buildConfig.Source = source

	// Check source availability
	resp, err := con.Rpc.CheckSource(con.Context(), buildConfig)
	if err != nil {
		return fmt.Errorf("failed to check source: %w", err)
	}
	buildConfig.Source = resp.Source

	// Set profile config
	buildConfig.MaleficConfig, _ = profile.ToYAML()

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
func buildProfileFromWizard(result *wizardfw.WizardResult, buildType string) (*implanttypes.ProfileConfig, error) {
	profileName := getString(result, "profile")

	var profile *implanttypes.ProfileConfig
	var err error

	if profileName != "" {
		// Load existing profile - this would need RPC call, for now use default
		profile, err = implanttypes.LoadProfile(consts.DefaultProfile)
	} else {
		profile, err = implanttypes.LoadProfile(consts.DefaultProfile)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load profile: %w", err)
	}

	// Set implant mode
	if profile.Implant != nil && (buildType == consts.CommandBuildBeacon || buildType == consts.CommandBuildBind) {
		profile.Implant.Mod = buildType
	}

	// Basic profile settings
	cron := getString(result, "cron")
	if cron != "" {
		profile.Basic.Cron = cron
	}

	jitter := getFloat64(result, "jitter")
	if jitter > 0 {
		profile.Basic.Jitter = jitter
	}

	initRetry := getInt(result, "init_retry")
	if initRetry > 0 {
		profile.Basic.InitRetry = initRetry
	}

	serverRetry := getInt(result, "server_retry")
	if serverRetry > 0 {
		profile.Basic.ServerRetry = serverRetry
	}

	globalRetry := getInt(result, "global_retry")
	if globalRetry > 0 {
		profile.Basic.GlobalRetry = globalRetry
	}

	encryption := getString(result, "encryption")
	if encryption != "" {
		profile.Basic.Encryption = encryption
	}

	key := getString(result, "key")
	if key != "" {
		profile.Basic.Key = key
	}

	// Secure mode
	secure := getBool(result, "secure")
	if secure {
		profile.Basic.Secure = &implanttypes.SecureProfile{
			Enable: true,
		}
	}

	// Proxy settings
	proxy := getString(result, "proxy")
	proxyUseEnv := getBool(result, "proxy_use_env")
	if proxy != "" || proxyUseEnv {
		profile.Basic.Proxy = &implanttypes.ProxyProfile{
			URL:         proxy,
			UseEnvProxy: proxyUseEnv,
		}
	}

	// Guardrail settings
	guardrailIPs := getString(result, "guardrail_ips")
	guardrailUsers := getString(result, "guardrail_users")
	guardrailServers := getString(result, "guardrail_servers")
	guardrailDomains := getString(result, "guardrail_domains")

	if guardrailIPs != "" || guardrailUsers != "" || guardrailServers != "" || guardrailDomains != "" {
		if profile.Basic.Guardrail == nil {
			profile.Basic.Guardrail = &implanttypes.GuardrailProfile{}
		}
		profile.Basic.Guardrail.Enable = true
		profile.Basic.Guardrail.RequireAll = true

		if guardrailIPs != "" {
			profile.Basic.Guardrail.IPAddresses = strings.Split(guardrailIPs, ",")
		}
		if guardrailUsers != "" {
			profile.Basic.Guardrail.Usernames = strings.Split(guardrailUsers, ",")
		}
		if guardrailServers != "" {
			profile.Basic.Guardrail.ServerNames = strings.Split(guardrailServers, ",")
		}
		if guardrailDomains != "" {
			profile.Basic.Guardrail.Domains = strings.Split(guardrailDomains, ",")
		}
	}

	// Parse addresses
	addresses := getString(result, "addresses")
	if addresses != "" {
		for _, address := range strings.Split(addresses, ",") {
			address = strings.TrimSpace(address)
			if address == "" {
				continue
			}
			target := implanttypes.Target{}

			if strings.HasPrefix(address, "http://") {
				address = strings.TrimPrefix(address, "http://")
				if !strings.Contains(address, ":") {
					address = address + ":80"
				}
				target.Address = address
				target.Http = &implanttypes.HttpProfile{
					Method:  "POST",
					Path:    "/",
					Version: "1.1",
					Headers: map[string]string{
						"User-Agent":   uarand.GetRandom(),
						"Content-Type": "application/octet-stream",
					},
				}
			} else if strings.HasPrefix(address, "https://") {
				address = strings.TrimPrefix(address, "https://")
				if !strings.Contains(address, ":") {
					address = address + ":443"
				}
				target.Address = address
				target.Http = &implanttypes.HttpProfile{
					Method:  "POST",
					Path:    "/",
					Version: "1.1",
					Headers: map[string]string{
						"User-Agent":   uarand.GetRandom(),
						"Content-Type": "application/octet-stream",
					},
				}
				target.TLS = &implanttypes.TLSProfile{
					Enable:           true,
					SNI:              strings.Split(address, ":")[0],
					SkipVerification: true,
				}
			} else if strings.HasPrefix(address, "tcp://") {
				address = strings.TrimPrefix(address, "tcp://")
				if !strings.Contains(address, ":") {
					address = address + ":5001"
				}
				target.Address = address
				target.TCP = &implanttypes.TCPProfile{}
			} else if strings.HasPrefix(address, "tcp+tls://") {
				address = strings.TrimPrefix(address, "tcp+tls://")
				if !strings.Contains(address, ":") {
					address = address + ":5001"
				}
				target.Address = address
				target.TCP = &implanttypes.TCPProfile{}
				target.TLS = &implanttypes.TLSProfile{
					Enable:           true,
					SNI:              strings.Split(address, ":")[0],
					SkipVerification: true,
				}
			} else {
				// Default to TCP
				if !strings.Contains(address, ":") {
					address = address + ":5001"
				}
				target.Address = address
				target.TCP = &implanttypes.TCPProfile{}
			}

			profile.Basic.Targets = append(profile.Basic.Targets, target)
		}
	}

	// Modules
	modules := getStringSlice(result, "modules")
	if len(modules) > 0 {
		profile.Implant.Modules = modules
	}

	thirdModules := getStringSlice(result, "third_modules")
	if len(thirdModules) > 0 {
		profile.Implant.ThirdModules = thirdModules
		profile.Implant.Enable3rd = true
	}

	// OLLVM
	ollvm := getBool(result, "ollvm")
	if ollvm {
		profile.Build.OLLVM = &implanttypes.OLLVMProfile{
			Enable:   true,
			BCFObf:   true,
			SplitObf: true,
			SubObf:   true,
			FCO:      true,
			ConstEnc: true,
		}
	}

	// Anti-sandbox
	antiSandbox := getBool(result, "anti_sandbox")
	if antiSandbox {
		profile.Implant.Anti = &implanttypes.AntiProfile{
			Sandbox: true,
		}
	}

	return profile, nil
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
