package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLayer(t *testing.T) {
	root := t.TempDir()

	tests := []struct {
		name    string
		path    string
		content string
		relDir  string
		want    string
	}{
		{
			name:    "command conformance",
			path:    filepath.Join(root, "client", "command", "agent", "commands_test.go"),
			content: "package agent\n",
			relDir:  "client/command/agent",
			want:    "command_conformance",
		},
		{
			name:    "integration tag",
			path:    filepath.Join(root, "server", "client_server_integration_test.go"),
			content: "//go:build integration\n\npackage server\n",
			relDir:  "server",
			want:    "integration",
		},
		{
			name:    "mock implant tag",
			path:    filepath.Join(root, "server", "mock_implant_test.go"),
			content: "//go:build mockimplant\n\npackage server\n",
			relDir:  "server",
			want:    "mockimplant",
		},
		{
			name:    "unit fallback",
			path:    filepath.Join(root, "helper", "utils", "output", "output_test.go"),
			content: "package output\n",
			relDir:  "helper/utils/output",
			want:    "unit",
		},
		{
			name:    "other build tag",
			path:    filepath.Join(root, "server", "internal", "llm", "provider_test.go"),
			content: "//go:build bridge_agent_proto\n\npackage llm\n",
			relDir:  "server/internal/llm",
			want:    "tagged",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.MkdirAll(filepath.Dir(tt.path), 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			if err := os.WriteFile(tt.path, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("write file: %v", err)
			}

			got, err := detectLayer(tt.path, tt.relDir)
			if err != nil {
				t.Fatalf("detect layer: %v", err)
			}
			if got != tt.want {
				t.Fatalf("detect layer = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMergeComponentCoverage(t *testing.T) {
	component := ComponentSpec{
		ID:             "client-command-agent",
		Name:           "Agent Commands",
		Path:           "client/command/agent",
		Tier:           "tier1",
		ExpectedLayers: []string{"command_conformance", "integration"},
	}

	report := mergeComponentCoverage(componentReport(component), []PackageStats{
		{
			Path:          "client/command/agent",
			GoFiles:       6,
			TestFiles:     1,
			Layers:        []string{"command_conformance"},
			TestFilePaths: []string{"client/command/agent/commands_test.go"},
		},
	})

	if report.Status != "needs_attention" {
		t.Fatalf("status = %q, want needs_attention", report.Status)
	}
	if len(report.MissingLayers) != 1 || report.MissingLayers[0] != "integration" {
		t.Fatalf("missing layers = %#v", report.MissingLayers)
	}
	if report.ObservedTests != 1 {
		t.Fatalf("observed tests = %d, want 1", report.ObservedTests)
	}
}
