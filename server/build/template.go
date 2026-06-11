package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

var (
	TemplatePath = filepath.Join(configs.ServerRootPath, "templates")

	validTransports = map[string]bool{
		"tcp":      true,
		"http":     true,
		"rem":      true,
		"tcp_tls":  true,
		"http_tls": true,
	}
)

func InitTemplatePath() {
	os.MkdirAll(TemplatePath, 0700)
}

// FindTemplate locates a pre-compiled template binary by transport and target.
// Naming convention: malefic-{transport}-{target}[.exe]
func FindTemplate(transport, target string) (string, error) {
	if !validTransports[transport] {
		return "", fmt.Errorf("unsupported transport %q, valid: tcp, http, rem, tcp_tls, http_tls", transport)
	}

	prefix := fmt.Sprintf("malefic-%s-%s", transport, target)
	entries, err := os.ReadDir(TemplatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template directory %s: %w", TemplatePath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		nameNoExt := strings.TrimSuffix(name, filepath.Ext(name))
		if name == prefix || nameNoExt == prefix || name == prefix+".exe" {
			return filepath.Join(TemplatePath, name), nil
		}
	}

	return "", fmt.Errorf("template not found for transport=%s target=%s in %s", transport, target, TemplatePath)
}

// ListTemplates returns all available template files grouped by transport.
func ListTemplates() map[string][]string {
	result := make(map[string][]string)
	entries, err := os.ReadDir(TemplatePath)
	if err != nil {
		logs.Log.Warnf("[template] failed to read template dir: %v", err)
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "malefic-") {
			continue
		}

		parts := strings.SplitN(strings.TrimPrefix(name, "malefic-"), "-", 2)
		if len(parts) < 2 {
			continue
		}
		transport := parts[0]
		if validTransports[transport] {
			result[transport] = append(result[transport], name)
		}
	}
	return result
}

// DetectTransport infers the transport type from implant.yaml content.
// It looks for transport-related keywords in the targets section.
func DetectTransport(implantYaml []byte) string {
	content := string(implantYaml)
	hasTLS := strings.Contains(content, "tls:") && !strings.Contains(content, "tls: {}")
	hasHTTP := strings.Contains(content, "http:")
	hasREM := strings.Contains(content, "rem:")

	switch {
	case hasHTTP && hasTLS:
		return "http_tls"
	case hasHTTP:
		return "http"
	case hasREM:
		return "rem"
	case hasTLS:
		return "tcp_tls"
	default:
		return "tcp"
	}
}
