package server

import (
	"strconv"
	"testing"

	"github.com/chainreactors/malice-network/client/wizard"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestValidatePort(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"1", false},
		{"80", false},
		{"5004", false},
		{"65535", false},
		{"0", true},
		{"-1", true},
		{"65536", true},
		{"abc", true},
		{"", true},
	}
	for _, tt := range tests {
		err := validatePort(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("validatePort(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestValidateIP(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"127.0.0.1", false},
		{"0.0.0.0", false},
		{"192.168.1.1", false},
		{"10.0.0.1", false},
		{"::1", false},
		{"", true},
		{"not-an-ip", true},
		{"256.1.1.1", true},
		{"1.2.3", true},
	}
	for _, tt := range tests {
		err := validateIP(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateIP(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"https://example.com/hook/123", false},
		{"http://localhost:8080/api", false},
		{"https://open.feishu.cn/open-apis/bot/v2/hook/abc-def", false},
		{"", true},
		{"a", true},
		{"ftp://example.com", true},
		{"not-a-url", true},
		{"://missing-scheme", true},
		{"https://", true},
	}
	for _, tt := range tests {
		err := validateURL(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateURL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
		}
	}
}

func TestValidateNotEmpty(t *testing.T) {
	validator := validateNotEmpty("Test Field")
	if err := validator("some value"); err != nil {
		t.Errorf("validateNotEmpty(non-empty) should not error, got %v", err)
	}
	if err := validator(""); err == nil {
		t.Error("validateNotEmpty(empty) should error")
	}
}

func TestShouldOfferQuickstart(t *testing.T) {
	tests := []struct {
		name             string
		opt              *Options
		configMissing    bool
		interactive      bool
		hasActiveCommand bool
		want             bool
	}{
		{
			name:             "default startup offers quickstart on interactive terminal",
			opt:              &Options{},
			configMissing:    true,
			interactive:      true,
			hasActiveCommand: false,
			want:             true,
		},
		{
			name:             "nil options never offer quickstart",
			opt:              nil,
			configMissing:    true,
			interactive:      true,
			hasActiveCommand: false,
			want:             false,
		},
		{
			name:             "explicit quickstart skips prompt",
			opt:              &Options{Quickstart: true},
			configMissing:    true,
			interactive:      true,
			hasActiveCommand: false,
			want:             false,
		},
		{
			name:             "existing config skips prompt",
			opt:              &Options{},
			configMissing:    false,
			interactive:      true,
			hasActiveCommand: false,
			want:             false,
		},
		{
			name:             "non interactive terminal skips prompt",
			opt:              &Options{},
			configMissing:    true,
			interactive:      false,
			hasActiveCommand: false,
			want:             false,
		},
		{
			name:             "active subcommand skips prompt",
			opt:              &Options{},
			configMissing:    true,
			interactive:      true,
			hasActiveCommand: true,
			want:             false,
		},
		{
			name:             "daemon mode skips prompt",
			opt:              &Options{Daemon: true},
			configMissing:    true,
			interactive:      true,
			hasActiveCommand: false,
			want:             false,
		},
		{
			name:             "server only skips prompt",
			opt:              &Options{ServerOnly: true},
			configMissing:    true,
			interactive:      true,
			hasActiveCommand: false,
			want:             false,
		},
		{
			name:             "listener only skips prompt",
			opt:              &Options{ListenerOnly: true},
			configMissing:    true,
			interactive:      true,
			hasActiveCommand: false,
			want:             false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldOfferQuickstart(tt.opt, tt.configMissing, tt.interactive, tt.hasActiveCommand)
			if got != tt.want {
				t.Fatalf("shouldOfferQuickstart() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuickstartGroupActive(t *testing.T) {
	tests := []struct {
		name  string
		group *wizard.FormGroup
		want  bool
	}{
		{
			name:  "required collapsed group is active",
			group: &wizard.FormGroup{Optional: false, Expanded: false},
			want:  true,
		},
		{
			name:  "optional collapsed group is inactive",
			group: &wizard.FormGroup{Optional: true, Expanded: false},
			want:  false,
		},
		{
			name:  "optional expanded group is active",
			group: &wizard.FormGroup{Optional: true, Expanded: true},
			want:  true,
		},
		{
			name:  "nil group is inactive",
			group: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quickstartGroupActive(tt.group)
			if got != tt.want {
				t.Fatalf("quickstartGroupActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuickstartSectionEnabled(t *testing.T) {
	tests := []struct {
		name    string
		group   *wizard.FormGroup
		enabled bool
		want    bool
	}{
		{
			name:    "expanded optional section with enable on is enabled",
			group:   &wizard.FormGroup{Optional: true, Expanded: true},
			enabled: true,
			want:    true,
		},
		{
			name:    "expanded optional section with enable off is disabled",
			group:   &wizard.FormGroup{Optional: true, Expanded: true},
			enabled: false,
			want:    false,
		},
		{
			name:    "collapsed optional section is disabled even when enable defaults true",
			group:   &wizard.FormGroup{Optional: true, Expanded: false},
			enabled: true,
			want:    false,
		},
		{
			name:    "required section still respects explicit enable",
			group:   &wizard.FormGroup{Optional: false, Expanded: false},
			enabled: false,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quickstartSectionEnabled(tt.group, tt.enabled)
			if got != tt.want {
				t.Fatalf("quickstartSectionEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQuickstartBuildSectionEnabled(t *testing.T) {
	if !quickstartBuildSectionEnabled(true) {
		t.Fatal("Build section should be enabled by default even when collapsed")
	}
	if quickstartBuildSectionEnabled(false) {
		t.Fatal("Build section should be disabled when explicitly turned off")
	}
}

func TestQuickstartBuildDefaults(t *testing.T) {
	defaults := quickstartDefaultBuildOptions()
	if !defaults.enableBuild {
		t.Fatal("Build service should default to enabled when the Build section is expanded")
	}
	if defaults.enableAutoBuild {
		t.Fatal("Auto-build should default to disabled")
	}
	if !defaults.buildPulse {
		t.Fatal("Build pulse should default to enabled")
	}
	if defaults.buildSource != "saas" {
		t.Fatalf("Build source = %q, want saas", defaults.buildSource)
	}
}

func TestRandomHex(t *testing.T) {
	h := randomHex(16)
	if len(h) != 32 {
		t.Errorf("randomHex(16) length = %d, want 32", len(h))
	}
	// verify it's valid hex
	if _, err := strconv.ParseUint(h[:16], 16, 64); err != nil {
		t.Errorf("randomHex(16) produced invalid hex: %s", h)
	}
	// verify uniqueness (probabilistic)
	h2 := randomHex(16)
	if h == h2 {
		t.Error("randomHex produced identical values on consecutive calls")
	}
}

func TestDefaultEncryption(t *testing.T) {
	key := "maliceofinternal"
	enc := defaultEncryption(key)
	if len(enc) != 2 {
		t.Fatalf("defaultEncryption should return 2 entries, got %d", len(enc))
	}
	if enc[0].Type != "aes" || enc[0].Key != key {
		t.Errorf("enc[0] = {%s, %s}, want {aes, %s}", enc[0].Type, enc[0].Key, key)
	}
	if enc[1].Type != "xor" || enc[1].Key != key {
		t.Errorf("enc[1] = {%s, %s}, want {xor, %s}", enc[1].Type, enc[1].Key, key)
	}
}

func TestDetectLocalIP(t *testing.T) {
	ip := detectLocalIP()
	if ip == "" {
		t.Error("detectLocalIP returned empty string")
	}
	// should be a valid IP
	if err := validateIP(ip); err != nil {
		t.Errorf("detectLocalIP returned invalid IP: %s", ip)
	}
}

func TestBuildNotifyConfig_Lark(t *testing.T) {
	cfg := buildNotifyConfig("lark", "https://example.com/hook", "")
	if !cfg.Enable {
		t.Error("Enable should be true")
	}
	if cfg.Lark == nil {
		t.Fatal("Lark config should not be nil")
	}
	if !cfg.Lark.Enable {
		t.Error("Lark.Enable should be true")
	}
	if cfg.Lark.WebHookUrl != "https://example.com/hook" {
		t.Errorf("Lark.WebHookUrl = %q, want %q", cfg.Lark.WebHookUrl, "https://example.com/hook")
	}
	if cfg.Telegram != nil || cfg.DingTalk != nil || cfg.ServerChan != nil || cfg.PushPlus != nil {
		t.Error("other notification services should be nil for lark")
	}
}

func TestBuildNotifyConfig_Telegram(t *testing.T) {
	cfg := buildNotifyConfig("telegram", "bot123:ABC", "-100123456")
	if cfg.Telegram == nil {
		t.Fatal("Telegram config should not be nil")
	}
	if cfg.Telegram.APIKey != "bot123:ABC" {
		t.Errorf("Telegram.APIKey = %q, want %q", cfg.Telegram.APIKey, "bot123:ABC")
	}
	if cfg.Telegram.ChatID != -100123456 {
		t.Errorf("Telegram.ChatID = %d, want %d", cfg.Telegram.ChatID, -100123456)
	}
}

func TestBuildNotifyConfig_DingTalk(t *testing.T) {
	cfg := buildNotifyConfig("dingtalk", "tok123", "SEC456")
	if cfg.DingTalk == nil {
		t.Fatal("DingTalk config should not be nil")
	}
	if cfg.DingTalk.Token != "tok123" {
		t.Errorf("DingTalk.Token = %q, want %q", cfg.DingTalk.Token, "tok123")
	}
	if cfg.DingTalk.Secret != "SEC456" {
		t.Errorf("DingTalk.Secret = %q, want %q", cfg.DingTalk.Secret, "SEC456")
	}
}

func TestBuildNotifyConfig_ServerChan(t *testing.T) {
	cfg := buildNotifyConfig("serverchan", "https://sc.ftqq.com/key.send", "")
	if cfg.ServerChan == nil {
		t.Fatal("ServerChan config should not be nil")
	}
	if cfg.ServerChan.URL != "https://sc.ftqq.com/key.send" {
		t.Errorf("ServerChan.URL = %q", cfg.ServerChan.URL)
	}
}

func TestBuildNotifyConfig_PushPlus(t *testing.T) {
	cfg := buildNotifyConfig("pushplus", "pp_tok", "my_topic")
	if cfg.PushPlus == nil {
		t.Fatal("PushPlus config should not be nil")
	}
	if cfg.PushPlus.Token != "pp_tok" {
		t.Errorf("PushPlus.Token = %q", cfg.PushPlus.Token)
	}
	if cfg.PushPlus.Topic != "my_topic" {
		t.Errorf("PushPlus.Topic = %q", cfg.PushPlus.Topic)
	}
	if cfg.PushPlus.Channel != "wechat" {
		t.Errorf("PushPlus.Channel = %q", cfg.PushPlus.Channel)
	}
}

func TestBuildPipelineOptions(t *testing.T) {
	tcp := []*configs.TcpPipelineConfig{{Name: "tcp1"}, {Name: "tcp2"}}
	http := []*configs.HttpPipelineConfig{{Name: "http1"}}
	rem := []*configs.REMConfig{{Name: "rem1"}}

	// all selected
	opts := buildPipelineOptions([]string{"tcp", "http", "rem"}, tcp, http, rem)
	if len(opts) != 4 {
		t.Errorf("expected 4 options, got %d", len(opts))
	}

	// only tcp
	opts = buildPipelineOptions([]string{"tcp"}, tcp, http, rem)
	if len(opts) != 2 {
		t.Errorf("expected 2 options for tcp only, got %d", len(opts))
	}

	// none selected
	opts = buildPipelineOptions([]string{}, tcp, http, rem)
	if len(opts) != 0 {
		t.Errorf("expected 0 options for empty selection, got %d", len(opts))
	}

	// nil slices
	opts = buildPipelineOptions([]string{"tcp", "http", "rem"}, nil, nil, nil)
	if len(opts) != 0 {
		t.Errorf("expected 0 options for nil slices, got %d", len(opts))
	}
}

func TestCollectPipelineNames(t *testing.T) {
	tcp := []*configs.TcpPipelineConfig{{Name: "tcp1"}}
	http := []*configs.HttpPipelineConfig{{Name: "http1"}}
	rem := []*configs.REMConfig{{Name: "rem1"}}

	names := collectPipelineNames(tcp, http, rem)
	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	// nil slices
	names = collectPipelineNames(nil, nil, nil)
	if len(names) != 0 {
		t.Errorf("expected 0 names for nil slices, got %d", len(names))
	}
}

func TestContainsString(t *testing.T) {
	slice := []string{"tcp", "http", "rem"}
	if !containsString(slice, "tcp") {
		t.Error("should contain tcp")
	}
	if !containsString(slice, "TCP") {
		t.Error("should contain TCP (case-insensitive)")
	}
	if containsString(slice, "websocket") {
		t.Error("should not contain websocket")
	}
	if containsString(nil, "tcp") {
		t.Error("nil slice should not contain anything")
	}
}
