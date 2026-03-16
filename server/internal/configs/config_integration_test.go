package configs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/helper/implanttypes"
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
	if got := serverCfg.Address(); got != "127.0.0.1:5004" {
		t.Fatalf("unexpected server address: %q", got)
	}
	if serverCfg.EncryptionKey != "maliceofinternal" {
		t.Fatalf("unexpected encryption key: %q", serverCfg.EncryptionKey)
	}
	if serverCfg.MiscConfig == nil || serverCfg.MiscConfig.PacketLength != 10485760 {
		t.Fatalf("unexpected misc config: %#v", serverCfg.MiscConfig)
	}
	if serverCfg.NotifyConfig == nil || serverCfg.NotifyConfig.Lark == nil || serverCfg.NotifyConfig.Lark.Enable {
		t.Fatalf("unexpected notify config: %#v", serverCfg.NotifyConfig)
	}
	if serverCfg.SaasConfig == nil || !serverCfg.SaasConfig.Enable {
		t.Fatalf("unexpected saas config: %#v", serverCfg.SaasConfig)
	}
	if github := GetGithubConfig(); github == nil || github.Workflow != "generate.yml" || github.ToProtobuf().Repo != "malefic" {
		t.Fatalf("unexpected github getter/protobuf: %#v", github)
	}
	if saas := GetSaasConfig(); saas == nil || saas.Url != "https://build.chainreactors.red" {
		t.Fatalf("unexpected saas getter: %#v", saas)
	}
	if notify := GetNotifyConfig(); notify == nil || notify.Lark == nil || notify.Lark.WebHookUrl != "" {
		t.Fatalf("unexpected notify getter: %#v", notify)
	}
	if acme := GetAcmeConfig(); acme == nil || acme.CAUrl != "https://acme-v02.api.letsencrypt.org/directory" {
		t.Fatalf("unexpected acme getter/defaults: %#v", acme)
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
	if serverCfg.NotifyConfig == nil || serverCfg.NotifyConfig.Enable {
		t.Fatalf("unexpected notify config: %#v", serverCfg.NotifyConfig)
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

func TestFullConfigFixtureParsesAndDrivesMechanisms(t *testing.T) {
	dir := t.TempDir()
	certPath := writeTestFile(t, dir, "cert.pem", "cert-data")
	keyPath := writeTestFile(t, dir, "key.pem", "key-data")
	caPath := writeTestFile(t, dir, "ca.pem", "ca-data")
	errorPagePath := writeTestFile(t, dir, "error.html", "<html>error</html>")
	webContentPath := writeTestFile(t, dir, "index.html", "<html>ok</html>")
	configPath := writeTestFile(t, dir, "config.yaml", fullConfigYAML(certPath, keyPath, caPath, errorPagePath, webContentPath))

	loadTestConfig(t, configPath)
	withServerConfigFile(t, configPath)

	serverCfg := GetServerConfig()
	listenerCfg := GetListenerConfig()
	if serverCfg == nil || listenerCfg == nil {
		t.Fatalf("expected parsed config, got server=%#v listener=%#v", serverCfg, listenerCfg)
	}

	if !serverCfg.Enable || !serverCfg.DaemonConfig {
		t.Fatalf("unexpected server enable/daemon config: %#v", serverCfg)
	}
	if serverCfg.GRPCHost != "10.0.0.7" || serverCfg.GRPCPort != 7443 || serverCfg.IP != "10.0.0.8" {
		t.Fatalf("unexpected server network config: %#v", serverCfg)
	}
	if serverCfg.Address() != "10.0.0.8:7443" {
		t.Fatalf("unexpected server address: %q", serverCfg.Address())
	}
	if serverCfg.EncryptionKey != "fixture-encryption" {
		t.Fatalf("unexpected encryption key: %q", serverCfg.EncryptionKey)
	}
	if serverCfg.LogConfig == nil || serverCfg.LogConfig.Level != 7 {
		t.Fatalf("unexpected log config: %#v", serverCfg.LogConfig)
	}
	if serverCfg.MiscConfig == nil || serverCfg.MiscConfig.PacketLength != 2048 || serverCfg.MiscConfig.Certificate != "cert-data" || serverCfg.MiscConfig.PrivateKey != "key-data" {
		t.Fatalf("unexpected misc config: %#v", serverCfg.MiscConfig)
	}
	if serverCfg.NotifyConfig == nil || !serverCfg.NotifyConfig.Enable {
		t.Fatalf("unexpected notify config: %#v", serverCfg.NotifyConfig)
	}
	if serverCfg.NotifyConfig.Telegram == nil || !serverCfg.NotifyConfig.Telegram.Enable || serverCfg.NotifyConfig.Telegram.APIKey != "bot-token" || serverCfg.NotifyConfig.Telegram.ChatID != 123456 {
		t.Fatalf("unexpected telegram config: %#v", serverCfg.NotifyConfig.Telegram)
	}
	if serverCfg.NotifyConfig.DingTalk == nil || !serverCfg.NotifyConfig.DingTalk.Enable || serverCfg.NotifyConfig.DingTalk.Secret != "ding-secret" || serverCfg.NotifyConfig.DingTalk.Token != "ding-token" {
		t.Fatalf("unexpected dingtalk config: %#v", serverCfg.NotifyConfig.DingTalk)
	}
	if serverCfg.NotifyConfig.Lark == nil || !serverCfg.NotifyConfig.Lark.Enable || serverCfg.NotifyConfig.Lark.WebHookUrl != "https://lark.example/webhook" || serverCfg.NotifyConfig.Lark.Secret != "lark-secret" {
		t.Fatalf("unexpected lark config: %#v", serverCfg.NotifyConfig.Lark)
	}
	if serverCfg.NotifyConfig.ServerChan == nil || !serverCfg.NotifyConfig.ServerChan.Enable || serverCfg.NotifyConfig.ServerChan.URL != "https://sctapi.ftqq.com/send" {
		t.Fatalf("unexpected serverchan config: %#v", serverCfg.NotifyConfig.ServerChan)
	}
	if serverCfg.NotifyConfig.PushPlus == nil || !serverCfg.NotifyConfig.PushPlus.Enable || serverCfg.NotifyConfig.PushPlus.Token != "push-token" || serverCfg.NotifyConfig.PushPlus.Topic != "ops" || serverCfg.NotifyConfig.PushPlus.Channel != "wechat" {
		t.Fatalf("unexpected pushplus config: %#v", serverCfg.NotifyConfig.PushPlus)
	}
	if github := GetGithubConfig(); github == nil || github.Owner != "chainreactors" || github.Repo != "malefic" || github.Token != "gh-token" || github.ToProtobuf().WorkflowId != "build.yml" {
		t.Fatalf("unexpected github config: %#v", github)
	}
	if saas := GetSaasConfig(); saas == nil || !saas.Enable || saas.Url != "https://saas.example/api" || saas.Token != "saas-token" {
		t.Fatalf("unexpected saas config: %#v", saas)
	}
	if acme := GetAcmeConfig(); acme == nil || acme.Email != "ops@example.com" || acme.CAUrl != "https://acme-staging-v02.api.letsencrypt.org/directory" || acme.Provider != "cloudflare" || acme.ToProtobuf().Credentials["CF_API_TOKEN"] != "cf-token" {
		t.Fatalf("unexpected acme config: %#v", acme)
	}
	if serverCfg.DatabaseConfig == nil {
		t.Fatal("expected database config")
	}
	dsn, err := serverCfg.DatabaseConfig.DSN()
	if err != nil {
		t.Fatalf("database dsn failed: %v", err)
	}
	if dsn == "" || serverCfg.DatabaseConfig.Dialect != Postgres || serverCfg.DatabaseConfig.Host != "db.example.com" || serverCfg.DatabaseConfig.Port != 5433 || serverCfg.DatabaseConfig.Database != "malice_fixture" || serverCfg.DatabaseConfig.Username != "malice" || serverCfg.DatabaseConfig.Password != "secret" || serverCfg.DatabaseConfig.MaxIdleConns != 12 || serverCfg.DatabaseConfig.MaxOpenConns != 34 || serverCfg.DatabaseConfig.LogLevel != "info" || serverCfg.DatabaseConfig.Params["sslmode"] != "require" {
		t.Fatalf("unexpected database config/dsn: %#v dsn=%q", serverCfg.DatabaseConfig, dsn)
	}
	cert, key, err := LoadMiscConfig()
	if err != nil {
		t.Fatalf("LoadMiscConfig failed: %v", err)
	}
	if string(cert) != "cert-data" || string(key) != "key-data" {
		t.Fatalf("unexpected misc cert payload: cert=%q key=%q", cert, key)
	}

	if listenerCfg.Name != "fixture-listener" || listenerCfg.Auth != "fixture.auth" || listenerCfg.IP != "10.9.0.1" {
		t.Fatalf("unexpected listener config: %#v", listenerCfg)
	}
	if listenerCfg.AutoBuildConfig == nil || len(listenerCfg.AutoBuildConfig.Target) != 2 || len(listenerCfg.AutoBuildConfig.Pipeline) != 2 {
		t.Fatalf("unexpected auto build config: %#v", listenerCfg.AutoBuildConfig)
	}
	if len(listenerCfg.TcpPipelines) != 1 || len(listenerCfg.HttpPipelines) != 1 || len(listenerCfg.BindPipelineConfig) != 1 || len(listenerCfg.REMs) != 1 || len(listenerCfg.Websites) != 1 {
		t.Fatalf("unexpected listener collections: %#v", listenerCfg)
	}

	tcpPB, err := listenerCfg.TcpPipelines[0].ToProtobuf(listenerCfg.Name)
	if err != nil {
		t.Fatalf("tcp ToProtobuf failed: %v", err)
	}
	if tcpPB.Parser != "tcp-parser" || tcpPB.GetTcp().Host != "0.0.0.0" || tcpPB.GetTcp().Port != 5001 || tcpPB.Tls == nil || !tcpPB.Tls.Enable || tcpPB.Tls.CertSubject == nil || tcpPB.Tls.CertSubject.Cn != "fixture-cn" || tcpPB.Tls.CertSubject.O != "fixture-org" || tcpPB.Tls.CertSubject.C != "CN" || tcpPB.Tls.CertSubject.L != "Shanghai" || tcpPB.Tls.CertSubject.Ou != "Red" || tcpPB.Tls.CertSubject.St != "Shanghai" {
		t.Fatalf("unexpected tcp protobuf: %#v", tcpPB)
	}
	if tcpPB.Secure == nil || !tcpPB.Secure.Enable || tcpPB.Secure.GetServerKeypair().GetPublicKey() != "spub" || tcpPB.Secure.GetServerKeypair().GetPrivateKey() != "spriv" || tcpPB.Secure.GetImplantKeypair().GetPublicKey() != "ipub" || tcpPB.Secure.GetImplantKeypair().GetPrivateKey() != "ipriv" {
		t.Fatalf("unexpected tcp secure protobuf: %#v", tcpPB.Secure)
	}
	if _, err := NewCrypto(tcpPB.Encryption); err != nil {
		t.Fatalf("tcp NewCrypto failed: %v", err)
	}

	httpPB, err := listenerCfg.HttpPipelines[0].ToProtobuf(listenerCfg.Name)
	if err != nil {
		t.Fatalf("http ToProtobuf failed: %v", err)
	}
	httpParams, err := implanttypes.UnmarshalPipelineParams(httpPB.GetHttp().Params)
	if err != nil {
		t.Fatalf("failed to unmarshal http params: %v", err)
	}
	if httpPB.Parser != "http-parser" || httpPB.GetHttp().Host != "0.0.0.0" || httpPB.GetHttp().Port != 8080 || httpParams.ErrorPage != "<html>error</html>" || httpParams.BodyPrefix != "before" || httpParams.BodySuffix != "after" {
		t.Fatalf("unexpected http pipeline protobuf/params: pb=%#v params=%#v", httpPB, httpParams)
	}
	if len(httpParams.Headers["X-Test"]) != 2 || httpPB.Secure == nil || !httpPB.Secure.Enable || httpPB.Secure.GetServerKeypair().GetPublicKey() != "hspub" || httpPB.Secure.GetServerKeypair().GetPrivateKey() != "hspriv" || httpPB.Secure.GetImplantKeypair().GetPublicKey() != "hipub" || httpPB.Secure.GetImplantKeypair().GetPrivateKey() != "hipriv" {
		t.Fatalf("unexpected http headers/secure: params=%#v secure=%#v", httpParams, httpPB.Secure)
	}
	if _, err := NewCrypto(httpPB.Encryption); err != nil {
		t.Fatalf("http NewCrypto failed: %v", err)
	}

	bindPB, err := listenerCfg.BindPipelineConfig[0].ToProtobuf(listenerCfg.Name)
	if err != nil {
		t.Fatalf("bind ToProtobuf failed: %v", err)
	}
	if bindPB.Parser != consts.ImplantMalefic || bindPB.Tls == nil || !bindPB.Tls.Enable || bindPB.Tls.Cert == nil || bindPB.Tls.Cert.Cert != "cert-data" || bindPB.Tls.Cert.Key != "key-data" || len(bindPB.Encryption) != 1 {
		t.Fatalf("unexpected bind protobuf: %#v", bindPB)
	}

	remPB, err := listenerCfg.REMs[0].ToProtobuf(listenerCfg.Name)
	if err != nil {
		t.Fatalf("rem ToProtobuf failed: %v", err)
	}
	if remPB.GetRem().Console != "tcp://127.0.0.1:12345" {
		t.Fatalf("unexpected rem protobuf: %#v", remPB)
	}

	website := listenerCfg.Websites[0]
	if website.WebsiteName != "fixture-site" || website.RootPath != "/site" || website.Port != 9443 {
		t.Fatalf("unexpected website config: %#v", website)
	}
	websiteTLS, err := website.TlsConfig.ReadCert()
	if err != nil {
		t.Fatalf("website tls read failed: %v", err)
	}
	if !websiteTLS.Enable || websiteTLS.Cert == nil || websiteTLS.CA == nil {
		t.Fatalf("unexpected website tls config: %#v", websiteTLS)
	}
	contentPB, err := website.WebContents[0].ToProtobuf()
	if err != nil {
		t.Fatalf("web content ToProtobuf failed: %v", err)
	}
	if contentPB.Type != "raw" || contentPB.File != webContentPath || string(contentPB.Content) != "<html>ok</html>" || contentPB.Path != "/index.html" {
		t.Fatalf("unexpected web content protobuf: %#v", contentPB)
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
	rem := listenerCfg.REMs[0]
	website := listenerCfg.Websites[0]

	if tcp.Parser != "auto" || tcp.TlsConfig == nil || !tcp.TlsConfig.Enable || len(tcp.EncryptionConfig) != 2 {
		t.Fatalf("unexpected tcp pipeline: %#v", tcp)
	}
	if http.Parser != "auto" || http.TlsConfig == nil || !http.TlsConfig.Enable || len(http.EncryptionConfig) != 2 {
		t.Fatalf("unexpected http pipeline: %#v", http)
	}
	if bind.Name != "bind_pipelines" || len(bind.EncryptionConfig) != 1 {
		t.Fatalf("unexpected bind pipeline: %#v", bind)
	}
	if rem.Name != "rem_default" {
		t.Fatalf("unexpected rem pipeline: %#v", rem)
	}
	if website.WebsiteName != "default-website" || website.RootPath != "/" || website.Port != 80 {
		t.Fatalf("unexpected website config: %#v", website)
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

func withServerConfigFile(t *testing.T, path string) {
	t.Helper()

	prev := ServerConfigFileName
	ServerConfigFileName = path
	t.Cleanup(func() {
		ServerConfigFileName = prev
	})
}

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

func fullConfigYAML(certPath, keyPath, caPath, errorPagePath, webContentPath string) string {
	return "server:\n" +
		"  enable: true\n" +
		"  grpc_host: 10.0.0.7\n" +
		"  grpc_port: 7443\n" +
		"  ip: 10.0.0.8\n" +
		"  daemon: true\n" +
		"  encryption_key: fixture-encryption\n" +
		"  log:\n" +
		"    level: 7\n" +
		"  config:\n" +
		"    packet_length: 2048\n" +
		"    certificate: cert-data\n" +
		"    certificate_key: key-data\n" +
		"  notify:\n" +
		"    enable: true\n" +
		"    telegram:\n" +
		"      enable: true\n" +
		"      api_key: bot-token\n" +
		"      chat_id: 123456\n" +
		"    dingtalk:\n" +
		"      enable: true\n" +
		"      secret: ding-secret\n" +
		"      token: ding-token\n" +
		"    lark:\n" +
		"      enable: true\n" +
		"      webhook_url: https://lark.example/webhook\n" +
		"      secret: lark-secret\n" +
		"    serverchan:\n" +
		"      enable: true\n" +
		"      url: https://sctapi.ftqq.com/send\n" +
		"    pushplus:\n" +
		"      enable: true\n" +
		"      token: push-token\n" +
		"      topic: ops\n" +
		"      channel: wechat\n" +
		"  github:\n" +
		"    owner: chainreactors\n" +
		"    repo: malefic\n" +
		"    token: gh-token\n" +
		"    workflow: build.yml\n" +
		"  saas:\n" +
		"    enable: true\n" +
		"    url: https://saas.example/api\n" +
		"    token: saas-token\n" +
		"  acme:\n" +
		"    email: ops@example.com\n" +
		"    ca_url: https://acme-staging-v02.api.letsencrypt.org/directory\n" +
		"    provider: cloudflare\n" +
		"    credentials:\n" +
		"      CF_API_TOKEN: cf-token\n" +
		"  database:\n" +
		"    dialect: postgresql\n" +
		"    host: db.example.com\n" +
		"    port: 5433\n" +
		"    database: malice_fixture\n" +
		"    username: malice\n" +
		"    password: secret\n" +
		"    params:\n" +
		"      sslmode: require\n" +
		"    max_idle_conns: 12\n" +
		"    max_open_conns: 34\n" +
		"    log_level: info\n" +
		"listeners:\n" +
		"  enable: true\n" +
		"  name: fixture-listener\n" +
		"  auth: fixture.auth\n" +
		"  ip: 10.9.0.1\n" +
		"  auto_build:\n" +
		"    enable: true\n" +
		"    build_pulse: true\n" +
		"    target:\n" +
		"    - x86_64-pc-windows-gnu\n" +
		"    - x86_64-unknown-linux-gnu\n" +
		"    pipeline:\n" +
		"    - tcp-main\n" +
		"    - http-main\n" +
		"  tcp:\n" +
		"  - enable: true\n" +
		"    name: tcp-main\n" +
		"    host: 0.0.0.0\n" +
		"    port: 5001\n" +
		"    parser: tcp-parser\n" +
		"    tls:\n" +
		"      enable: true\n" +
		"      cert_file: " + certPath + "\n" +
		"      key_file: " + keyPath + "\n" +
		"      ca_file: " + caPath + "\n" +
		"      CN: fixture-cn\n" +
		"      O: fixture-org\n" +
		"      C: CN\n" +
		"      L: Shanghai\n" +
		"      OU: Red\n" +
		"      ST: Shanghai\n" +
		"    encryption:\n" +
		"    - type: aes\n" +
		"      key: aes-key\n" +
		"    - type: xor\n" +
		"      key: xor-key\n" +
		"    secure:\n" +
		"      enable: true\n" +
		"      server_public_key: spub\n" +
		"      server_private_key: spriv\n" +
		"      implant_public_key: ipub\n" +
		"      implant_private_key: ipriv\n" +
		"  bind:\n" +
		"  - enable: true\n" +
		"    name: bind-main\n" +
		"    tls:\n" +
		"      enable: true\n" +
		"      cert_file: " + certPath + "\n" +
		"      key_file: " + keyPath + "\n" +
		"    encryption:\n" +
		"    - type: aes\n" +
		"      key: bind-key\n" +
		"  http:\n" +
		"  - enable: true\n" +
		"    name: http-main\n" +
		"    host: 0.0.0.0\n" +
		"    port: 8080\n" +
		"    parser: http-parser\n" +
		"    tls:\n" +
		"      enable: true\n" +
		"      cert_file: " + certPath + "\n" +
		"      key_file: " + keyPath + "\n" +
		"    encryption:\n" +
		"    - type: aes\n" +
		"      key: http-aes\n" +
		"    secure:\n" +
		"      enable: true\n" +
		"      server_public_key: hspub\n" +
		"      server_private_key: hspriv\n" +
		"      implant_public_key: hipub\n" +
		"      implant_private_key: hipriv\n" +
		"    headers:\n" +
		"      X-Test:\n" +
		"      - a\n" +
		"      - b\n" +
		"    error_page: " + errorPagePath + "\n" +
		"    body_prefix: before\n" +
		"    body_suffix: after\n" +
		"  rem:\n" +
		"  - enable: true\n" +
		"    name: rem-main\n" +
		"    console: tcp://127.0.0.1:12345\n" +
		"  website:\n" +
		"  - enable: true\n" +
		"    name: fixture-site\n" +
		"    root: /site\n" +
		"    port: 9443\n" +
		"    tls:\n" +
		"      enable: true\n" +
		"      cert_file: " + certPath + "\n" +
		"      key_file: " + keyPath + "\n" +
		"      ca_file: " + caPath + "\n" +
		"    content:\n" +
		"    - file: " + webContentPath + "\n" +
		"      path: /index.html\n" +
		"      type: raw\n"
}
