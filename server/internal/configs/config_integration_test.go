package configs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gookit/config/v2"
	yamlDriver "github.com/gookit/config/v2/yaml"
)

func TestRepositoryRootConfigParses(t *testing.T) {
	loadTestConfig(t, repoPath(t, "config.yaml"))

	serverCfg := GetServerConfig()
	listenerCfg := GetListenerConfig()

	if serverCfg == nil {
		t.Fatal("expected server config")
	}
	if listenerCfg == nil {
		t.Fatal("expected listener config")
	}
	if serverCfg.GRPCHost != "0.0.0.0" || serverCfg.GRPCPort != 5004 {
		t.Fatalf("unexpected grpc config: %#v", serverCfg)
	}
	if serverCfg.EncryptionKey != "maliceofinternal" {
		t.Fatalf("unexpected encryption key: %q", serverCfg.EncryptionKey)
	}
	if serverCfg.SaasConfig == nil || !serverCfg.SaasConfig.Enable {
		t.Fatalf("unexpected saas config: %#v", serverCfg.SaasConfig)
	}
	if listenerCfg.Name != "listener" || listenerCfg.IP != "127.0.0.1" {
		t.Fatalf("unexpected listener config: %#v", listenerCfg)
	}
	if listenerCfg.AutoBuildConfig == nil || !listenerCfg.AutoBuildConfig.Enable || !listenerCfg.AutoBuildConfig.BuildPulse {
		t.Fatalf("unexpected auto build config: %#v", listenerCfg.AutoBuildConfig)
	}
	assertPipelineSlices(t, listenerCfg)
}

func TestRepositoryServerConfigParses(t *testing.T) {
	loadTestConfig(t, repoPath(t, "server", "config.yaml"))

	serverCfg := GetServerConfig()
	listenerCfg := GetListenerConfig()

	if serverCfg == nil || listenerCfg == nil {
		t.Fatalf("expected both server and listener config, got server=%#v listener=%#v", serverCfg, listenerCfg)
	}
	if serverCfg.GithubConfig == nil || serverCfg.GithubConfig.Workflow != "generate.yml" {
		t.Fatalf("unexpected github config: %#v", serverCfg.GithubConfig)
	}
	if len(listenerCfg.REMs) != 1 || listenerCfg.REMs[0].Name != "rem_default" {
		t.Fatalf("unexpected rem config: %#v", listenerCfg.REMs)
	}
	assertPipelineSlices(t, listenerCfg)
}

func TestLoadMiscConfigParsesCertificateKeys(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("server:\n  config:\n    packet_length: 2048\n    certificate: cert-data\n    certificate_key: key-data\n")
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	initTestConfigRuntime(t)

	prev := ServerConfigFileName
	ServerConfigFileName = configPath
	t.Cleanup(func() {
		ServerConfigFileName = prev
	})

	cert, key, err := LoadMiscConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(cert) != "cert-data" || string(key) != "key-data" {
		t.Fatalf("unexpected misc config payload: cert=%q key=%q", cert, key)
	}
}

func assertPipelineSlices(t *testing.T, listenerCfg *ListenerConfig) {
	t.Helper()

	if len(listenerCfg.TcpPipelines) != 1 {
		t.Fatalf("unexpected tcp pipelines: %#v", listenerCfg.TcpPipelines)
	}
	if len(listenerCfg.HttpPipelines) != 1 {
		t.Fatalf("unexpected http pipelines: %#v", listenerCfg.HttpPipelines)
	}
	if len(listenerCfg.BindPipelineConfig) != 1 {
		t.Fatalf("unexpected bind pipelines: %#v", listenerCfg.BindPipelineConfig)
	}

	tcp := listenerCfg.TcpPipelines[0]
	http := listenerCfg.HttpPipelines[0]
	bind := listenerCfg.BindPipelineConfig[0]

	if tcp.Parser != "auto" || tcp.TlsConfig == nil || !tcp.TlsConfig.Enable || len(tcp.EncryptionConfig) != 2 {
		t.Fatalf("unexpected tcp pipeline: %#v", tcp)
	}
	if http.Parser != "auto" || http.TlsConfig == nil || !http.TlsConfig.Enable || len(http.EncryptionConfig) != 2 {
		t.Fatalf("unexpected http pipeline: %#v", http)
	}
	if bind.Name != "bind_pipelines" || len(bind.EncryptionConfig) != 1 {
		t.Fatalf("unexpected bind pipeline: %#v", bind)
	}
}

func loadTestConfig(t *testing.T, path string) {
	t.Helper()

	initTestConfigRuntime(t)

	if err := config.LoadFiles(path); err != nil {
		t.Fatalf("failed to load config %s: %v", path, err)
	}
}

func repoPath(t *testing.T, parts ...string) string {
	t.Helper()

	items := append([]string{"..", "..", ".."}, parts...)
	return filepath.Join(items...)
}

func initTestConfigRuntime(t *testing.T) {
	t.Helper()

	config.Reset()
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yamlDriver.Driver)
	t.Cleanup(config.Reset)
}
