package configs

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	types "github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	chunkparser "github.com/chainreactors/malice-network/server/internal/parser"
	maleficparser "github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"github.com/gookit/config/v2"
)

func TestInitConfigAndCertDirCreateRuntimePaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".malice")
	withTestServerPaths(t, root)

	if err := InitConfig(); err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}

	for _, dir := range []string{
		ServerRootPath,
		ContextPath,
		CertsPath,
		TempPath,
		LogPath,
		AuditPath,
		BinPath,
		WebsitePath,
		ListenerPath,
	} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("expected runtime dir %s to exist, err=%v", dir, err)
		}
	}

	if err := os.RemoveAll(CertsPath); err != nil {
		t.Fatalf("failed to remove cert dir: %v", err)
	}

	got := GetCertDir()
	if got != CertsPath {
		t.Fatalf("GetCertDir returned %q, want %q", got, CertsPath)
	}
	if info, err := os.Stat(CertsPath); err != nil || !info.IsDir() {
		t.Fatalf("expected cert dir %s to exist after GetCertDir, err=%v", CertsPath, err)
	}
}

func TestFindConfigPrefersCurrentWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	configPath := filepath.Join(dir, "fixture.yaml")
	if err := os.WriteFile(configPath, []byte("server:\n  enable: true\nlisteners:\n  enable: true\n"), 0o600); err != nil {
		t.Fatalf("failed to write fixture config: %v", err)
	}

	if got := FindConfig("fixture.yaml"); got != "fixture.yaml" {
		t.Fatalf("FindConfig returned %q, want %q", got, "fixture.yaml")
	}
	if got := FindConfig("missing.yaml"); got != "" {
		t.Fatalf("FindConfig returned %q for missing config", got)
	}
}

func TestUpdateConfigHelpersReflectInGetters(t *testing.T) {
	initTestConfigRuntime(t)

	github := &GithubConfig{
		Owner:    "fixture-owner",
		Repo:     "fixture-repo",
		Token:    "fixture-token",
		Workflow: "fixture.yml",
	}
	if err := UpdateGithubConfig(github); err != nil {
		t.Fatalf("UpdateGithubConfig failed: %v", err)
	}

	notify := &NotifyConfig{
		Enable: true,
		Telegram: &TelegramConfig{
			Enable: true,
			APIKey: "telegram-token",
			ChatID: 99,
		},
		DingTalk: &DingTalkConfig{
			Enable: true,
			Secret: "ding-secret",
			Token:  "ding-token",
		},
		Lark: &LarkConfig{
			Enable:     true,
			WebHookUrl: "https://lark.example/hook",
			Secret:     "lark-secret",
		},
		ServerChan: &ServerChanConfig{
			Enable: true,
			URL:    "https://serverchan.example/send",
		},
		PushPlus: &PushPlusConfig{
			Enable:  true,
			Token:   "push-token",
			Topic:   "soc",
			Channel: "mail",
		},
	}
	if err := UpdateNotifyConfig(notify); err != nil {
		t.Fatalf("UpdateNotifyConfig failed: %v", err)
	}

	saas := &SaasConfig{
		Enable: true,
		Url:    "https://saas.example/api",
		Token:  "saas-token",
	}
	if err := UpdateSaasConfig(saas); err != nil {
		t.Fatalf("UpdateSaasConfig failed: %v", err)
	}

	acme := &AcmeConfig{
		Email:    "ops@example.com",
		CAUrl:    "https://acme.example/directory",
		Provider: "cloudflare",
		Credentials: map[string]string{
			"CF_API_TOKEN": "cf-token",
		},
	}
	if err := UpdateAcmeConfig(acme); err != nil {
		t.Fatalf("UpdateAcmeConfig failed: %v", err)
	}

	gotGithub := GetGithubConfig()
	if gotGithub == nil || gotGithub.Owner != github.Owner || gotGithub.Repo != github.Repo || gotGithub.Token != github.Token || gotGithub.Workflow != github.Workflow {
		t.Fatalf("unexpected github getter result: %#v", gotGithub)
	}

	gotNotify := GetNotifyConfig()
	if gotNotify == nil || !gotNotify.Enable || gotNotify.Telegram == nil || gotNotify.Telegram.APIKey != "telegram-token" || gotNotify.DingTalk == nil || gotNotify.DingTalk.Secret != "ding-secret" || gotNotify.Lark == nil || gotNotify.Lark.WebHookUrl != "https://lark.example/hook" || gotNotify.ServerChan == nil || gotNotify.ServerChan.URL != "https://serverchan.example/send" || gotNotify.PushPlus == nil || gotNotify.PushPlus.Channel != "mail" {
		t.Fatalf("unexpected notify getter result: %#v", gotNotify)
	}

	gotSaas := GetSaasConfig()
	if gotSaas == nil || !gotSaas.Enable || gotSaas.Url != saas.Url || gotSaas.Token != saas.Token {
		t.Fatalf("unexpected saas getter result: %#v", gotSaas)
	}

	gotAcme := GetAcmeConfig()
	if gotAcme == nil || gotAcme.Email != acme.Email || gotAcme.CAUrl != acme.CAUrl || gotAcme.Provider != acme.Provider || gotAcme.Credentials["CF_API_TOKEN"] != "cf-token" {
		t.Fatalf("unexpected acme getter result: %#v", gotAcme)
	}

	if config.String("server.github.workflow") != "fixture.yml" {
		t.Fatalf("github config was not written to runtime config: %q", config.String("server.github.workflow"))
	}
	if !config.Bool("server.notify.telegram.enable") || config.String("server.notify.pushplus.channel") != "mail" {
		t.Fatalf("notify config was not written to runtime config")
	}
	if config.String("server.saas.url") != "https://saas.example/api" {
		t.Fatalf("saas config was not written to runtime config: %q", config.String("server.saas.url"))
	}
	if config.String("server.acme.provider") != "cloudflare" {
		t.Fatalf("acme config was not written to runtime config: %q", config.String("server.acme.provider"))
	}
}

func TestPacketLengthConfigDrivesChunkingAndParserLimits(t *testing.T) {
	configPath := writeTestFile(t, t.TempDir(), "config.yaml", ""+
		"server:\n"+
		"  enable: true\n"+
		"  ip: 127.0.0.1\n"+
		"  config:\n"+
		"    packet_length: 8\n"+
		"listeners:\n"+
		"  enable: true\n")

	loadTestConfig(t, configPath)

	if got := config.Int(consts.ConfigMaxPacketLength); got != 8 {
		t.Fatalf("unexpected max packet length: %d", got)
	}

	payload := []byte("123456789")
	if count := chunkparser.Count(payload, config.Int(consts.ConfigMaxPacketLength)); count != 2 {
		t.Fatalf("unexpected chunk count: %d", count)
	}

	var chunks [][]byte
	for chunk := range chunkparser.Chunked(payload, config.Int(consts.ConfigMaxPacketLength)) {
		chunks = append(chunks, append([]byte(nil), chunk...))
	}
	if len(chunks) != 2 || string(chunks[0]) != "12345678" || string(chunks[1]) != "9" {
		t.Fatalf("unexpected chunked payload: %#v", chunks)
	}

	parser := maleficparser.NewMaleficParser()
	allowed := uint32(config.Uint(consts.ConfigMaxPacketLength)) + consts.KB*16
	sid, length, err := parser.ReadHeader(newTestHeaderConn(7, allowed))
	if err != nil {
		t.Fatalf("ReadHeader failed at allowed limit: %v", err)
	}
	if sid != 7 || length != allowed+1 {
		t.Fatalf("unexpected header parse result sid=%d length=%d", sid, length)
	}

	_, _, err = parser.ReadHeader(newTestHeaderConn(9, allowed+1))
	if !errors.Is(err, types.ErrPacketTooLarge) {
		t.Fatalf("expected ErrPacketTooLarge, got %v", err)
	}
}

func TestPipelineMechanismsUseConfiguredValues(t *testing.T) {
	dir := t.TempDir()
	certPath := writeTestFile(t, dir, "cert.pem", "cert-data")
	keyPath := writeTestFile(t, dir, "key.pem", "key-data")
	caPath := writeTestFile(t, dir, "ca.pem", "ca-data")
	errorPagePath := writeTestFile(t, dir, "error.html", "<html>error</html>")
	webContentPath := writeTestFile(t, dir, "index.html", "<html>ok</html>")
	configPath := writeTestFile(t, dir, "config.yaml", fullConfigYAML(certPath, keyPath, caPath, errorPagePath, webContentPath))

	loadTestConfig(t, configPath)

	listenerCfg := GetListenerConfig()
	if listenerCfg == nil {
		t.Fatal("expected listener config")
	}

	tcpPB, err := listenerCfg.TcpPipelines[0].ToProtobuf(listenerCfg.Name)
	if err != nil {
		t.Fatalf("tcp ToProtobuf failed: %v", err)
	}
	tcpCryptors, err := NewCrypto(tcpPB.Encryption)
	if err != nil {
		t.Fatalf("NewCrypto failed: %v", err)
	}
	if len(tcpCryptors) != len(listenerCfg.TcpPipelines[0].EncryptionConfig) {
		t.Fatalf("unexpected tcp cryptor count: %d", len(tcpCryptors))
	}

	httpPB, err := listenerCfg.HttpPipelines[0].ToProtobuf(listenerCfg.Name)
	if err != nil {
		t.Fatalf("http ToProtobuf failed: %v", err)
	}
	params, err := implanttypes.UnmarshalPipelineParams(httpPB.GetHttp().Params)
	if err != nil {
		t.Fatalf("failed to decode http params: %v", err)
	}
	if params.ErrorPage != "<html>error</html>" || params.BodyPrefix != "before" || params.BodySuffix != "after" || len(params.Headers["X-Test"]) != 2 {
		t.Fatalf("unexpected http params: %#v", params)
	}
	if httpPB.Secure == nil || httpPB.Secure.GetServerKeypair().GetPublicKey() != "hspub" {
		t.Fatalf("unexpected http secure config: %#v", httpPB.Secure)
	}

	websitePB, err := listenerCfg.Websites[0].WebContents[0].ToProtobuf()
	if err != nil {
		t.Fatalf("website content ToProtobuf failed: %v", err)
	}
	if websitePB.Size != uint64(len("<html>ok</html>")) || string(websitePB.Content) != "<html>ok</html>" {
		t.Fatalf("unexpected website content protobuf: %#v", websitePB)
	}
}

func withTestServerPaths(t *testing.T, root string) {
	t.Helper()

	oldServerRootPath := ServerRootPath
	oldContextPath := ContextPath
	oldLogPath := LogPath
	oldCertsPath := CertsPath
	oldListenerPath := ListenerPath
	oldTempPath := TempPath
	oldPluginPath := PluginPath
	oldAuditPath := AuditPath
	oldWebsitePath := WebsitePath
	oldProfilePath := ProfilePath
	oldBinPath := BinPath
	oldDatabaseFileName := databaseFileName

	ServerRootPath = root
	ContextPath = filepath.Join(root, "context")
	LogPath = filepath.Join(root, "log")
	CertsPath = filepath.Join(root, "certs")
	ListenerPath = filepath.Join(root, "listener")
	TempPath = filepath.Join(root, "temp")
	PluginPath = filepath.Join(root, "plugins")
	AuditPath = filepath.Join(root, "audit")
	WebsitePath = filepath.Join(root, "web")
	ProfilePath = filepath.Join(root, "profile")
	BinPath = filepath.Join(root, "bin")
	databaseFileName = filepath.Join(root, "malice.db")

	t.Cleanup(func() {
		ServerRootPath = oldServerRootPath
		ContextPath = oldContextPath
		LogPath = oldLogPath
		CertsPath = oldCertsPath
		ListenerPath = oldListenerPath
		TempPath = oldTempPath
		PluginPath = oldPluginPath
		AuditPath = oldAuditPath
		WebsitePath = oldWebsitePath
		ProfilePath = oldProfilePath
		BinPath = oldBinPath
		databaseFileName = oldDatabaseFileName
	})
}

type testHeaderConn struct {
	reader *bytes.Reader
}

func newTestHeaderConn(sid, length uint32) *testHeaderConn {
	buf := make([]byte, maleficparser.HeaderLength)
	buf[maleficparser.MsgStart] = maleficparser.DefaultStartDelimiter
	binary.LittleEndian.PutUint32(buf[maleficparser.MsgSessionStart:maleficparser.MsgSessionEnd], sid)
	binary.LittleEndian.PutUint32(buf[maleficparser.MsgSessionEnd:], length)
	return &testHeaderConn{reader: bytes.NewReader(buf)}
}

func (c *testHeaderConn) Read(p []byte) (int, error) {
	return c.reader.Read(p)
}

func (c *testHeaderConn) Write(p []byte) (int, error) {
	return len(p), nil
}

func (c *testHeaderConn) Close() error {
	return nil
}
