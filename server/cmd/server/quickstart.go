package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/wizard"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

// RunQuickstart runs the interactive quickstart wizard using a single-page
// tabbed form so that users can review/edit all configuration at once.
func RunQuickstart(opt *Options) error {
	// skip if config already exists
	if _, err := os.Stat(opt.Config); err == nil {
		logs.Log.Warnf("config %s already exists, skipping quickstart", opt.Config)
		return nil
	}

	// --- field value holders ---

	// Group 1: Server
	serverIP := detectLocalIP()
	grpcHost := "0.0.0.0"
	grpcPort := "5004"
	encryptionKey := "maliceofinternal"

	// Group 2: Listener
	listenerName := "listener"
	listenerIP := serverIP

	// Group 3: Pipelines
	var selectedPipelines []string // filled by MultiSelect
	tcpPort := "5001"
	httpPort := "8080"
	remName := "rem_default"
	enableTLS := true

	// Group 4: Build (optional)
	enableAutoBuild := true
	buildPulse := true
	var buildTargets []string
	buildSource := "saas"
	saasURL := "https://build.chainreactors.red"
	saasToken := ""

	// Group 5: Notify (optional)
	notifyType := "lark"
	notifyParam1 := "" // webhook URL / API key / token
	notifyParam2 := "" // chat ID / secret (optional depending on type)

	// --- build form groups ---

	groups := []*wizard.FormGroup{
		{
			Name:  "server",
			Title: "Server",
			Fields: []*wizard.FormField{
				{
					Name: "server_ip", Title: "Server IP (external)",
					Kind: wizard.KindInput, InputValue: serverIP,
					Required: true, Validate: validateIP,
					Value: &serverIP,
				},
				{
					Name: "grpc_host", Title: "gRPC Host",
					Kind: wizard.KindInput, InputValue: grpcHost,
					Value: &grpcHost,
				},
				{
					Name: "grpc_port", Title: "gRPC Port",
					Kind: wizard.KindInput, InputValue: grpcPort,
					Required: true, Validate: validatePort,
					Value: &grpcPort,
				},
				{
					Name: "encryption_key", Title: "Encryption Key",
					Kind: wizard.KindInput, InputValue: encryptionKey,
					Required: true,
					Value: &encryptionKey,
				},
			},
		},
		{
			Name:  "listener",
			Title: "Listener",
			Fields: []*wizard.FormField{
				{
					Name: "listener_name", Title: "Listener Name",
					Kind: wizard.KindInput, InputValue: listenerName,
					Value: &listenerName,
				},
				{
					Name: "listener_ip", Title: "Listener IP",
					Kind: wizard.KindInput, InputValue: listenerIP,
					Required: true, Validate: validateIP,
					Value: &listenerIP,
				},
			},
		},
		{
			Name:  "pipelines",
			Title: "Pipelines",
			Fields: []*wizard.FormField{
				{
					Name: "pipeline_types", Title: "Pipeline Types",
					Kind:        wizard.KindMultiSelect,
					Options:     []string{"tcp", "http", "rem"},
					MultiSelect: map[int]bool{0: true, 1: true, 2: true},
					Value:       &selectedPipelines,
				},
				{
					Name: "tcp_port", Title: "TCP Port",
					Description: "ignored if tcp not selected",
					Kind: wizard.KindInput, InputValue: tcpPort,
					Validate: validatePort,
					Value:    &tcpPort,
				},
				{
					Name: "http_port", Title: "HTTP Port",
					Description: "ignored if http not selected",
					Kind: wizard.KindInput, InputValue: httpPort,
					Validate: validatePort,
					Value:    &httpPort,
				},
				{
					Name: "rem_name", Title: "REM Pipeline Name",
					Description: "ignored if rem not selected",
					Kind: wizard.KindInput, InputValue: remName,
					Value: &remName,
				},
				{
					Name: "enable_tls", Title: "Enable TLS",
					Kind: wizard.KindConfirm, ConfirmVal: enableTLS,
					Value: &enableTLS,
				},
			},
		},
		{
			Name:     "build",
			Title:    "Build",
			Optional: true,
			Fields: []*wizard.FormField{
				{
					Name: "enable_auto_build", Title: "Enable Auto-Build",
					Kind: wizard.KindConfirm, ConfirmVal: enableAutoBuild,
					Value: &enableAutoBuild,
				},
				{
					Name: "build_pulse", Title: "Build Pulse",
					Kind: wizard.KindConfirm, ConfirmVal: buildPulse,
					Value: &buildPulse,
				},
				{
					Name: "build_targets", Title: "Build Targets",
					Kind: wizard.KindMultiSelect,
					Options: []string{
						"x86_64-pc-windows-gnu",
						"x86_64-unknown-linux-musl",
						"i686-pc-windows-gnu",
						"x86_64-apple-darwin",
						"aarch64-apple-darwin",
						"aarch64-unknown-linux-musl",
					},
					MultiSelect: map[int]bool{0: true},
					Value:       &buildTargets,
				},
				{
					Name: "build_source", Title: "Build Source",
					Kind:    wizard.KindSelect,
					Options: []string{"saas", "github"},
					Value:   &buildSource,
				},
				{
					Name: "saas_url", Title: "SaaS Build URL",
					Kind: wizard.KindInput, InputValue: saasURL,
					Validate: validateURL,
					Value:    &saasURL,
				},
				{
					Name: "saas_token", Title: "SaaS Token (empty=auto)",
					Kind: wizard.KindInput, InputValue: saasToken,
					Value: &saasToken,
				},
			},
		},
		{
			Name:     "notify",
			Title:    "Notify",
			Optional: true,
			Fields: []*wizard.FormField{
				{
					Name: "notify_type", Title: "Notification Service",
					Kind:    wizard.KindSelect,
					Options: []string{"lark", "telegram", "dingtalk", "serverchan", "pushplus"},
					Value:   &notifyType,
				},
				{
					Name: "notify_param1", Title: "Webhook/Token/APIKey",
					Description: "main credential for the service",
					Kind: wizard.KindInput, InputValue: notifyParam1,
					Required: true,
					Value:    &notifyParam1,
				},
				{
					Name: "notify_param2", Title: "Secret/ChatID (optional)",
					Description: "dingtalk secret or telegram chat ID",
					Kind: wizard.KindInput, InputValue: notifyParam2,
					Value: &notifyParam2,
				},
			},
		},
	}

	form := wizard.NewGroupedWizardForm(groups)
	if err := form.Run(); err != nil {
		return err
	}

	// --- assemble configs from collected values ---

	port, _ := strconv.ParseUint(grpcPort, 10, 16)

	// Pipelines
	var tcpPipelines []*configs.TcpPipelineConfig
	var httpPipelines []*configs.HttpPipelineConfig
	var remConfigs []*configs.REMConfig

	for _, p := range selectedPipelines {
		switch p {
		case "tcp":
			tp, _ := strconv.ParseUint(tcpPort, 10, 16)
			tcpPipelines = append(tcpPipelines, &configs.TcpPipelineConfig{
				Enable:           true,
				Name:             "tcp",
				Host:             "0.0.0.0",
				Port:             uint16(tp),
				Parser:           "auto",
				TlsConfig:        &configs.TlsConfig{Enable: enableTLS},
				EncryptionConfig: defaultEncryption(encryptionKey),
			})
		case "http":
			hp, _ := strconv.ParseUint(httpPort, 10, 16)
			httpPipelines = append(httpPipelines, &configs.HttpPipelineConfig{
				Enable:           true,
				Name:             "http",
				Host:             "0.0.0.0",
				Port:             uint16(hp),
				Parser:           "auto",
				TlsConfig:        &configs.TlsConfig{Enable: enableTLS},
				EncryptionConfig: defaultEncryption(encryptionKey),
			})
		case "rem":
			remConfigs = append(remConfigs, &configs.REMConfig{
				Enable: true,
				Name:   remName,
			})
		}
	}

	// Auto-build (only if Build group was expanded and enabled)
	var autoBuild *configs.AutoBuildConfig
	if groups[3].Expanded && enableAutoBuild {
		pipelineNames := collectPipelineNames(tcpPipelines, httpPipelines, remConfigs)
		autoBuild = &configs.AutoBuildConfig{
			Enable:     true,
			BuildPulse: buildPulse,
			Target:     buildTargets,
			Pipeline:   pipelineNames,
		}
	}

	// Build source
	var githubConfig *configs.GithubConfig
	saasConfig := &configs.SaasConfig{Enable: false}
	if groups[3].Expanded {
		switch buildSource {
		case "saas":
			saasConfig = &configs.SaasConfig{
				Enable: true,
				Url:    saasURL,
				Token:  saasToken,
			}
		case "github":
			githubConfig = &configs.GithubConfig{
				Repo:     "malefic",
				Workflow: "generate.yml",
			}
		}
	}

	// Notify (only if Notify group was expanded)
	var notifyConfig *configs.NotifyConfig
	if groups[4].Expanded && notifyParam1 != "" {
		notifyConfig = buildNotifyConfig(notifyType, notifyParam1, notifyParam2)
	}

	// Assemble configs
	opt.Server = &configs.ServerConfig{
		Enable:        true,
		GRPCPort:      uint16(port),
		GRPCHost:      grpcHost,
		IP:            serverIP,
		EncryptionKey: encryptionKey,
		NotifyConfig:  notifyConfig,
		GithubConfig:  githubConfig,
		SaasConfig:    saasConfig,
		MiscConfig:    &configs.MiscConfig{PacketLength: 10485760},
	}

	opt.Listeners = &configs.ListenerConfig{
		Enable:          true,
		Name:            listenerName,
		Auth:            "listener.auth",
		IP:              listenerIP,
		TcpPipelines:    tcpPipelines,
		HttpPipelines:   httpPipelines,
		REMs:            remConfigs,
		AutoBuildConfig: autoBuild,
	}

	if err := opt.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	logs.Log.Importantf("quickstart config saved to %s", opt.Config)
	return nil
}

// collectPipelineNames gathers names from all configured pipelines.
func collectPipelineNames(
	tcpPipelines []*configs.TcpPipelineConfig,
	httpPipelines []*configs.HttpPipelineConfig,
	remConfigs []*configs.REMConfig,
) []string {
	var names []string
	for _, tc := range tcpPipelines {
		names = append(names, tc.Name)
	}
	for _, hc := range httpPipelines {
		names = append(names, hc.Name)
	}
	for _, rc := range remConfigs {
		names = append(names, rc.Name)
	}
	return names
}

// buildNotifyConfig creates a NotifyConfig from the selected type and parameters.
func buildNotifyConfig(notifyType, param1, param2 string) *configs.NotifyConfig {
	cfg := &configs.NotifyConfig{Enable: true}
	switch notifyType {
	case "lark":
		cfg.Lark = &configs.LarkConfig{Enable: true, WebHookUrl: param1}
	case "telegram":
		chatID, _ := strconv.ParseInt(param2, 10, 64)
		cfg.Telegram = &configs.TelegramConfig{Enable: true, APIKey: param1, ChatID: chatID}
	case "dingtalk":
		cfg.DingTalk = &configs.DingTalkConfig{Enable: true, Token: param1, Secret: param2}
	case "serverchan":
		cfg.ServerChan = &configs.ServerChanConfig{Enable: true, URL: param1}
	case "pushplus":
		cfg.PushPlus = &configs.PushPlusConfig{Enable: true, Token: param1, Topic: param2, Channel: "wechat"}
	}
	return cfg
}

func defaultEncryption(key string) implanttypes.EncryptionsConfig {
	return implanttypes.EncryptionsConfig{
		{Type: "aes", Key: key},
		{Type: "xor", Key: key},
	}
}

func detectLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

func validatePort(s string) error {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("port must be 1-65535")
	}
	return nil
}

func validateIP(s string) error {
	if s == "" {
		return fmt.Errorf("IP address is required")
	}
	if net.ParseIP(s) == nil {
		return fmt.Errorf("invalid IP address: %s", s)
	}
	return nil
}

func validateURL(s string) error {
	if s == "" {
		return fmt.Errorf("URL is required")
	}
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("invalid URL: %s", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	if u.Host == "" {
		return fmt.Errorf("URL must have a host")
	}
	return nil
}

func validateNotEmpty(fieldName string) func(string) error {
	return func(s string) error {
		if s == "" {
			return fmt.Errorf("%s is required", fieldName)
		}
		return nil
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// buildPipelineOptions is kept for potential external use.
func buildPipelineOptions(
	selectedPipelines []string,
	tcpPipelines []*configs.TcpPipelineConfig,
	httpPipelines []*configs.HttpPipelineConfig,
	remConfigs []*configs.REMConfig,
) []string {
	var opts []string
	for _, p := range selectedPipelines {
		switch p {
		case "tcp":
			for _, tc := range tcpPipelines {
				opts = append(opts, tc.Name)
			}
		case "http":
			for _, hc := range httpPipelines {
				opts = append(opts, hc.Name)
			}
		case "rem":
			for _, rc := range remConfigs {
				opts = append(opts, rc.Name)
			}
		}
	}
	return opts
}

// containsString checks if a slice contains a given string.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
