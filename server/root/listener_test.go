package root

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

type listenerRootRPCStub struct {
	clientrpc.RootRPCClient
	response *rootpb.Response
	calls    int
}

func (s *listenerRootRPCStub) AddListener(context.Context, *rootpb.Operator, ...grpc.CallOption) (*rootpb.Response, error) {
	s.calls++
	return s.response, nil
}

func withWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}

func listenerAuthResponse(t *testing.T, name string) *rootpb.Response {
	t.Helper()
	data, err := yaml.Marshal(&mtls.ClientConfig{
		Operator: name,
		Host:     "192.0.2.10",
		Port:     5004,
		Type:     mtls.Listener,
	})
	if err != nil {
		t.Fatalf("marshal listener auth: %v", err)
	}
	return &rootpb.Response{Response: string(data)}
}

func TestSaveListenerAuthCreatesMatchingConfig(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)

	rpc := &listenerRootRPCStub{response: listenerAuthResponse(t, "listener-2")}
	_, err := saveListenerAuth(rpc, &rootpb.Operator{
		Op:   "add",
		Args: []string{"listener-2"},
	})
	if err != nil {
		t.Fatalf("saveListenerAuth failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "listener-2.auth")); err != nil {
		t.Fatalf("listener auth file missing: %v", err)
	}
	configData, err := os.ReadFile(filepath.Join(dir, "listener-2.yaml"))
	if err != nil {
		t.Fatalf("listener config file missing: %v", err)
	}
	var config struct {
		Listeners struct {
			Enable    bool   `yaml:"enable"`
			Name      string `yaml:"name"`
			Auth      string `yaml:"auth"`
			Transport string `yaml:"transport"`
		} `yaml:"listeners"`
	}
	if err := yaml.Unmarshal(configData, &config); err != nil {
		t.Fatalf("unmarshal generated listener config: %v", err)
	}
	if !config.Listeners.Enable || config.Listeners.Name != "listener-2" ||
		config.Listeners.Auth != "listener-2.auth" || config.Listeners.Transport != "reverse" {
		t.Fatalf("generated listener config = %#v", config.Listeners)
	}
}

func TestSaveListenerAuthDoesNotOverwriteExistingBundle(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)

	authPath := filepath.Join(dir, "listener-2.auth")
	if err := os.WriteFile(authPath, []byte("existing"), 0o600); err != nil {
		t.Fatalf("write existing auth: %v", err)
	}
	rpc := &listenerRootRPCStub{response: listenerAuthResponse(t, "listener-2")}
	_, err := saveListenerAuth(rpc, &rootpb.Operator{
		Op:   "add",
		Args: []string{"listener-2"},
	})
	if err == nil {
		t.Fatal("saveListenerAuth should reject an existing listener bundle")
	}
	if rpc.calls != 0 {
		t.Fatalf("AddListener calls = %d, want 0", rpc.calls)
	}
	data, readErr := os.ReadFile(authPath)
	if readErr != nil {
		t.Fatalf("read existing auth: %v", readErr)
	}
	if string(data) != "existing" {
		t.Fatalf("existing auth was overwritten: %q", data)
	}
}

func TestResetListenerAuthPreservesExistingConfig(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)

	configPath := filepath.Join(dir, "listener-2.yaml")
	const existingConfig = "listeners:\n  name: listener-2\n  tcp:\n    - name: custom\n"
	if err := os.WriteFile(configPath, []byte(existingConfig), 0o600); err != nil {
		t.Fatalf("write existing config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "listener-2.auth"), []byte("old auth"), 0o600); err != nil {
		t.Fatalf("write existing auth: %v", err)
	}

	rpc := &listenerRootRPCStub{response: listenerAuthResponse(t, "listener-2")}
	_, err := saveListenerAuth(rpc, &rootpb.Operator{
		Op:   "reset",
		Args: []string{"listener-2"},
	})
	if err != nil {
		t.Fatalf("saveListenerAuth reset failed: %v", err)
	}
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}
	if string(configData) != existingConfig {
		t.Fatalf("reset overwrote listener config:\n%s", configData)
	}
	authData, err := os.ReadFile(filepath.Join(dir, "listener-2.auth"))
	if err != nil {
		t.Fatalf("read reset auth: %v", err)
	}
	if string(authData) == "old auth" {
		t.Fatal("reset did not update listener auth")
	}
}

func TestSaveListenerAuthRejectsMismatchedOperator(t *testing.T) {
	dir := t.TempDir()
	withWorkingDirectory(t, dir)

	rpc := &listenerRootRPCStub{response: listenerAuthResponse(t, "listener-other")}
	_, err := saveListenerAuth(rpc, &rootpb.Operator{
		Op:   "add",
		Args: []string{"listener-2"},
	})
	if err == nil {
		t.Fatal("saveListenerAuth should reject a mismatched auth operator")
	}
	for _, filename := range []string{"listener-2.auth", "listener-2.yaml"} {
		if _, statErr := os.Stat(filepath.Join(dir, filename)); !os.IsNotExist(statErr) {
			t.Fatalf("%s should not be created, stat error = %v", filename, statErr)
		}
	}
}
