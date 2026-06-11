package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverTaggedPackages(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, root, "server/client_server_integration_test.go", "//go:build integration\n\npackage server\n")
	writeTestFile(t, root, "client/command/listener/pipeline_integration_test.go", "//go:build integration\n\npackage listener\n")
	writeTestFile(t, root, "server/mock_implant_runtime_e2e_test.go", "//go:build mockimplant\n\npackage server\n")
	writeTestFile(t, root, "external/rem/runner/e2e_tunnel_dns_test.go", "//go:build dns\n\npackage runner\n")
	writeTestFile(t, root, "helper/cryptography/age_e2e_test.go", "package cryptography\n")

	got, err := discoverTaggedPackages(root, "integration")
	if err != nil {
		t.Fatalf("discoverTaggedPackages returned error: %v", err)
	}

	want := []string{
		"./client/command/listener",
		"./server",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverTaggedPackages() = %#v, want %#v", got, want)
	}
}

func TestDiscoverTaggedPackagesReturnsRootPackage(t *testing.T) {
	root := t.TempDir()

	writeTestFile(t, root, "root_integration_test.go", "//go:build integration\n\npackage main\n")

	got, err := discoverTaggedPackages(root, "integration")
	if err != nil {
		t.Fatalf("discoverTaggedPackages returned error: %v", err)
	}

	want := []string{"."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("discoverTaggedPackages() = %#v, want %#v", got, want)
	}
}

func TestBuildExprContains(t *testing.T) {
	if !buildExprContains("integration && linux", "integration") {
		t.Fatal("expected integration token match")
	}
	if buildExprContains("realintegration", "integration") {
		t.Fatal("did not expect partial token match")
	}
}

func writeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
