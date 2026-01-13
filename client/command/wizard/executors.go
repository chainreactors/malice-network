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
	// Note: listener_setup, build_*, cert_*, etc. can be added as needed
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
