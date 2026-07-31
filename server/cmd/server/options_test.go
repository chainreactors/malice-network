package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	config "github.com/gookit/config/v2"
	"github.com/jessevdk/go-flags"
)

func TestValidateListenerAuthNameRejectsMismatch(t *testing.T) {
	cfg := &configs.ListenerConfig{Name: "listener"}
	err := validateListenerAuthName(cfg, &mtls.ClientConfig{Operator: "listener-2"})
	if err == nil {
		t.Fatal("validateListenerAuthName should reject mismatched names")
	}
	if !strings.Contains(err.Error(), `listener config name "listener" does not match auth operator "listener-2"`) {
		t.Fatalf("unexpected mismatch error: %v", err)
	}
}

func TestValidateListenerAuthNameUsesOperatorWhenNameIsEmpty(t *testing.T) {
	cfg := &configs.ListenerConfig{}
	if err := validateListenerAuthName(cfg, &mtls.ClientConfig{Operator: "listener-2"}); err != nil {
		t.Fatalf("validateListenerAuthName failed: %v", err)
	}
	if cfg.Name != "listener-2" {
		t.Fatalf("listener name = %q, want listener-2", cfg.Name)
	}
}

func TestResolveListenerAuthPathRelativeToConfig(t *testing.T) {
	oldFilename := configs.CurrentServerConfigFilename
	configs.CurrentServerConfigFilename = filepath.Join(t.TempDir(), "listener-2.yaml")
	t.Cleanup(func() {
		configs.CurrentServerConfigFilename = oldFilename
	})

	got := resolveListenerAuthPath("listener-2.auth")
	want := filepath.Join(filepath.Dir(configs.CurrentServerConfigFilename), "listener-2.auth")
	if got != want {
		t.Fatalf("resolved auth path = %q, want %q", got, want)
	}
}

func TestInitListenerWritesConfiguredAuthPathAndIdentity(t *testing.T) {
	configs.InitTestConfigRuntime(t)
	root := t.TempDir()
	t.Chdir(root)
	configs.UseTestPaths(t, filepath.Join(root, ".malice"))
	if err := configs.InitConfig(); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}

	oldDBClient := db.Client
	t.Cleanup(func() { db.Client = oldDBClient })
	var err error
	db.Client, err = db.NewDBClient(nil)
	if err != nil {
		t.Fatalf("NewDBClient failed: %v", err)
	}

	config.Set("server.ip", "127.0.0.1")
	config.Set("server.grpc_port", 5004)
	configDir := filepath.Join(root, "node")
	oldFilename := configs.CurrentServerConfigFilename
	configs.CurrentServerConfigFilename = filepath.Join(configDir, "config.yaml")
	t.Cleanup(func() { configs.CurrentServerConfigFilename = oldFilename })

	opt := &Options{
		Listeners: &configs.ListenerConfig{
			Name: "listener-2",
			Auth: filepath.Join("credentials", "listener-2.auth"),
		},
	}
	if err := opt.InitListener(); err != nil {
		t.Fatalf("InitListener failed: %v", err)
	}

	authPath := filepath.Join(configDir, "credentials", "listener-2.auth")
	auth, err := mtls.ReadConfig(authPath)
	if err != nil {
		t.Fatalf("ReadConfig(%q) failed: %v", authPath, err)
	}
	if auth.Operator != "listener-2" {
		t.Fatalf("auth operator = %q, want listener-2", auth.Operator)
	}
	if opt.Listeners.Auth != authPath {
		t.Fatalf("listener auth path = %q, want %q", opt.Listeners.Auth, authPath)
	}
}

func TestValidateAllowsListenerOnlyWithoutServerConfig(t *testing.T) {
	opt := &Options{
		ListenerOnly: true,
		Listeners: &configs.ListenerConfig{
			Enable: true,
		},
	}

	if err := opt.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestPrepareListenerOnlyDoesNotRequireServerConfig(t *testing.T) {
	opt := &Options{
		ListenerOnly: true,
		Listeners: &configs.ListenerConfig{
			Enable: true,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PrepareListener() panicked with nil server config: %v", r)
		}
	}()

	if err := opt.PrepareListener(); err == nil {
		t.Fatal("PrepareListener() error = nil, want listener startup error")
	}
}

func TestPrepareConfigAllowsListenerOnlyFileWithoutServerSection(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "listener.yaml")
	content := []byte(`listeners:
  enable: true
  name: listener-a
  auth: listener-a.auth
  transport: forward
  forward:
    listen_host: 0.0.0.0
    listen_port: 5005
`)
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	opt := &Options{
		Config:       configPath,
		ListenerOnly: true,
	}
	if err := opt.PrepareConfig(nil); err != nil {
		t.Fatalf("PrepareConfig() error = %v, want nil", err)
	}
	if opt.Listeners == nil {
		t.Fatal("PrepareConfig() did not load listeners config")
	}
	if opt.Server != nil {
		t.Fatalf("PrepareConfig() server config = %#v, want nil", opt.Server)
	}
}

func TestExecuteReturnsErrorWithoutServerConfig(t *testing.T) {
	opt := &Options{
		ListenerOnly: true,
		Listeners: &configs.ListenerConfig{
			Enable: true,
		},
	}
	parser := flags.NewParser(opt, flags.Default)
	parser.Command.Active = &flags.Command{
		Name: opt.UserCmd.Name(),
		Active: &flags.Command{
			Name: "list",
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Execute() panicked with nil server config: %v", r)
		}
	}()

	if err := opt.Execute(nil, parser); err == nil {
		t.Fatal("Execute() error = nil, want missing server config error")
	}
}
