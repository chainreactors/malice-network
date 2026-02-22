package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/implanttypes"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/charmbracelet/huh"
)

func RunQuickstart(opt *Options) error {
	// skip if config already exists
	if _, err := os.Stat(opt.Config); err == nil {
		logs.Log.Warnf("config %s already exists, skipping quickstart", opt.Config)
		return nil
	}

	// Step 1: Server config
	serverIP := detectLocalIP()
	grpcHost := "0.0.0.0"
	grpcPort := "5004"
	encryptionKey := "maliceofinternal"

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Server IP (external)").Value(&serverIP).
				Validate(validateIP),
			huh.NewInput().Title("gRPC Host").Value(&grpcHost),
			huh.NewInput().Title("gRPC Port").Value(&grpcPort).
				Validate(validatePort),
			huh.NewInput().Title("Encryption Key").Value(&encryptionKey).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("encryption key cannot be empty")
					}
					return nil
				}),
		).Title("Server Configuration"),
	).Run()
	if err != nil {
		return err
	}

	port, _ := strconv.ParseUint(grpcPort, 10, 16)

	// Step 2: Listener config
	listenerName := "listener"
	listenerIP := serverIP

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("Listener Name").Value(&listenerName),
			huh.NewInput().Title("Listener IP").Value(&listenerIP).
				Validate(validateIP),
		).Title("Listener Configuration"),
	).Run()
	if err != nil {
		return err
	}

	// Step 3: Pipeline selection
	var selectedPipelines []string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Pipelines to enable").
				Options(
					huh.NewOption("TCP", "tcp").Selected(true),
					huh.NewOption("HTTP", "http").Selected(true),
					huh.NewOption("REM", "rem").Selected(true),
				).
				Value(&selectedPipelines),
		).Title("Pipeline Selection"),
	).Run()
	if err != nil {
		return err
	}

	// Step 3b: Configure each selected pipeline
	var tcpPipelines []*configs.TcpPipelineConfig
	var httpPipelines []*configs.HttpPipelineConfig
	var remConfigs []*configs.REMConfig

	for _, p := range selectedPipelines {
		switch p {
		case "tcp":
			cfg, err := configureTcpPipeline(encryptionKey)
			if err != nil {
				return err
			}
			tcpPipelines = append(tcpPipelines, cfg)
		case "http":
			cfg, err := configureHttpPipeline(encryptionKey)
			if err != nil {
				return err
			}
			httpPipelines = append(httpPipelines, cfg)
		case "rem":
			cfg, err := configureRemPipeline()
			if err != nil {
				return err
			}
			remConfigs = append(remConfigs, cfg)
		}
	}

	// Step 4: Auto-build
	enableAutoBuild := true
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Enable Auto-Build?").Value(&enableAutoBuild),
		).Title("Auto-Build Configuration"),
	).Run()
	if err != nil {
		return err
	}

	var autoBuild *configs.AutoBuildConfig
	if enableAutoBuild {
		buildPulse := true
		var buildTargets []string
		var buildPipelines []string

		// build pipeline options from selected pipelines
		pipelineOpts := buildPipelineOptions(selectedPipelines, tcpPipelines, httpPipelines, remConfigs)

		formFields := []huh.Field{
			huh.NewConfirm().Title("Build Pulse?").Value(&buildPulse),
			huh.NewMultiSelect[string]().
				Title("Build Targets").
				Options(
					huh.NewOption("x86_64-pc-windows-gnu", "x86_64-pc-windows-gnu").Selected(true),
					huh.NewOption("x86_64-unknown-linux-musl", "x86_64-unknown-linux-musl"),
					huh.NewOption("i686-pc-windows-gnu", "i686-pc-windows-gnu"),
					huh.NewOption("x86_64-apple-darwin", "x86_64-apple-darwin"),
					huh.NewOption("aarch64-apple-darwin", "aarch64-apple-darwin"),
					huh.NewOption("aarch64-unknown-linux-musl", "aarch64-unknown-linux-musl"),
				).
				Value(&buildTargets),
		}

		if len(pipelineOpts) > 0 {
			formFields = append(formFields,
				huh.NewMultiSelect[string]().
					Title("Auto-Build Pipelines").
					Options(pipelineOpts...).
					Value(&buildPipelines),
			)
		}

		err = huh.NewForm(
			huh.NewGroup(formFields...).Title("Auto-Build Details"),
		).Run()
		if err != nil {
			return err
		}

		autoBuild = &configs.AutoBuildConfig{
			Enable:     true,
			BuildPulse: buildPulse,
			Target:     buildTargets,
			Pipeline:   buildPipelines,
		}
	}

	// Step 5: Build source config (SaaS / GitHub)
	githubConfig, saasConfig, err := configureBuildSource()
	if err != nil {
		return err
	}

	// Step 6: Notify (optional)
	notifyConfig, err := configureNotify()
	if err != nil {
		return err
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

func buildPipelineOptions(
	selectedPipelines []string,
	tcpPipelines []*configs.TcpPipelineConfig,
	httpPipelines []*configs.HttpPipelineConfig,
	remConfigs []*configs.REMConfig,
) []huh.Option[string] {
	var opts []huh.Option[string]
	for _, p := range selectedPipelines {
		switch p {
		case "tcp":
			for _, tc := range tcpPipelines {
				opts = append(opts, huh.NewOption(tc.Name, tc.Name).Selected(true))
			}
		case "http":
			for _, hc := range httpPipelines {
				opts = append(opts, huh.NewOption(hc.Name, hc.Name).Selected(true))
			}
		case "rem":
			for _, rc := range remConfigs {
				opts = append(opts, huh.NewOption(rc.Name, rc.Name).Selected(true))
			}
		}
	}
	return opts
}

func configureBuildSource() (*configs.GithubConfig, *configs.SaasConfig, error) {
	var selectedSources []string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select Build Sources").
				Options(
					huh.NewOption("SaaS (cloud build service)", "saas").Selected(true),
					huh.NewOption("GitHub Actions", "github"),
				).
				Value(&selectedSources),
		).Title("Build Source Configuration"),
	).Run()
	if err != nil {
		return nil, nil, err
	}

	saasConfig := &configs.SaasConfig{Enable: false}
	githubConfig := &configs.GithubConfig{Repo: "malefic", Workflow: "generate.yml"}

	for _, src := range selectedSources {
		switch src {
		case "saas":
			saasUrl := "https://build.chainreactors.red"
			saasToken := ""
			err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title("SaaS Build URL").Value(&saasUrl).
						Validate(validateURL),
					huh.NewInput().Title("SaaS Token (leave empty to auto-register)").Value(&saasToken),
				).Title("SaaS Configuration"),
			).Run()
			if err != nil {
				return nil, nil, err
			}
			saasConfig = &configs.SaasConfig{
				Enable: true,
				Url:    saasUrl,
				Token:  saasToken,
			}
		case "github":
			owner := ""
			repo := "malefic"
			token := ""
			workflow := "generate.yml"
			err := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title("GitHub Owner").Value(&owner).
						Validate(validateNotEmpty("GitHub Owner")),
					huh.NewInput().Title("GitHub Repo").Value(&repo),
					huh.NewInput().Title("GitHub Token").Value(&token).
						Validate(validateNotEmpty("GitHub Token")),
					huh.NewInput().Title("GitHub Workflow").Value(&workflow),
				).Title("GitHub Actions Configuration"),
			).Run()
			if err != nil {
				return nil, nil, err
			}
			githubConfig = &configs.GithubConfig{
				Owner:    owner,
				Repo:     repo,
				Token:    token,
				Workflow: workflow,
			}
		}
	}

	return githubConfig, saasConfig, nil
}

func configureNotify() (*configs.NotifyConfig, error) {
	enableNotify := false
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title("Enable Notifications?").Value(&enableNotify),
		).Title("Notification Configuration"),
	).Run()
	if err != nil {
		return nil, err
	}

	if !enableNotify {
		return nil, nil
	}

	notifyType := "lark"
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Notification Service").
				Options(
					huh.NewOption("Lark", "lark"),
					huh.NewOption("Telegram", "telegram"),
					huh.NewOption("DingTalk", "dingtalk"),
					huh.NewOption("ServerChan", "serverchan"),
					huh.NewOption("PushPlus", "pushplus"),
				).
				Value(&notifyType),
		).Title("Notification Service"),
	).Run()
	if err != nil {
		return nil, err
	}

	return collectNotifyParams(notifyType)
}

func collectNotifyParams(notifyType string) (*configs.NotifyConfig, error) {
	cfg := &configs.NotifyConfig{Enable: true}

	switch notifyType {
	case "lark":
		webhookURL := ""
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Lark Webhook URL").Value(&webhookURL).
					Validate(validateURL),
			).Title("Lark Configuration"),
		).Run()
		if err != nil {
			return nil, err
		}
		cfg.Lark = &configs.LarkConfig{Enable: true, WebHookUrl: webhookURL}

	case "telegram":
		apiKey := ""
		chatIDStr := ""
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Telegram API Key").Value(&apiKey).
					Validate(validateNotEmpty("Telegram API Key")),
				huh.NewInput().Title("Telegram Chat ID").Value(&chatIDStr).
					Validate(func(s string) error {
						if s == "" {
							return fmt.Errorf("Chat ID is required")
						}
						if _, err := strconv.ParseInt(s, 10, 64); err != nil {
							return fmt.Errorf("Chat ID must be a number")
						}
						return nil
					}),
			).Title("Telegram Configuration"),
		).Run()
		if err != nil {
			return nil, err
		}
		chatID, _ := strconv.ParseInt(chatIDStr, 10, 64)
		cfg.Telegram = &configs.TelegramConfig{Enable: true, APIKey: apiKey, ChatID: chatID}

	case "dingtalk":
		token := ""
		secret := ""
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("DingTalk Token").Value(&token).
					Validate(validateNotEmpty("DingTalk Token")),
				huh.NewInput().Title("DingTalk Secret").Value(&secret).
					Validate(validateNotEmpty("DingTalk Secret")),
			).Title("DingTalk Configuration"),
		).Run()
		if err != nil {
			return nil, err
		}
		cfg.DingTalk = &configs.DingTalkConfig{Enable: true, Token: token, Secret: secret}

	case "serverchan":
		scURL := ""
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("ServerChan URL").Value(&scURL).
					Validate(validateURL),
			).Title("ServerChan Configuration"),
		).Run()
		if err != nil {
			return nil, err
		}
		cfg.ServerChan = &configs.ServerChanConfig{Enable: true, URL: scURL}

	case "pushplus":
		token := ""
		topic := ""
		channel := "wechat"
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("PushPlus Token").Value(&token).
					Validate(validateNotEmpty("PushPlus Token")),
				huh.NewInput().Title("PushPlus Topic (optional)").Value(&topic),
				huh.NewInput().Title("PushPlus Channel").Value(&channel),
			).Title("PushPlus Configuration"),
		).Run()
		if err != nil {
			return nil, err
		}
		cfg.PushPlus = &configs.PushPlusConfig{Enable: true, Token: token, Topic: topic, Channel: channel}
	}

	return cfg, nil
}

func configureTcpPipeline(encKey string) (*configs.TcpPipelineConfig, error) {
	name := "tcp"
	host := "0.0.0.0"
	portStr := "5001"
	enableTLS := true

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("TCP Pipeline Name").Value(&name),
			huh.NewInput().Title("Host").Value(&host),
			huh.NewInput().Title("Port").Value(&portStr).Validate(validatePort),
			huh.NewConfirm().Title("Enable TLS?").Value(&enableTLS),
		).Title("TCP Pipeline"),
	).Run()
	if err != nil {
		return nil, err
	}

	p, _ := strconv.ParseUint(portStr, 10, 16)
	return &configs.TcpPipelineConfig{
		Enable:           true,
		Name:             name,
		Host:             host,
		Port:             uint16(p),
		Parser:           "auto",
		TlsConfig:        &configs.TlsConfig{Enable: enableTLS},
		EncryptionConfig: defaultEncryption(encKey),
	}, nil
}

func configureHttpPipeline(encKey string) (*configs.HttpPipelineConfig, error) {
	name := "http"
	host := "0.0.0.0"
	portStr := "8080"
	enableTLS := true

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("HTTP Pipeline Name").Value(&name),
			huh.NewInput().Title("Host").Value(&host),
			huh.NewInput().Title("Port").Value(&portStr).Validate(validatePort),
			huh.NewConfirm().Title("Enable TLS?").Value(&enableTLS),
		).Title("HTTP Pipeline"),
	).Run()
	if err != nil {
		return nil, err
	}

	p, _ := strconv.ParseUint(portStr, 10, 16)
	return &configs.HttpPipelineConfig{
		Enable:           true,
		Name:             name,
		Host:             host,
		Port:             uint16(p),
		Parser:           "auto",
		TlsConfig:        &configs.TlsConfig{Enable: enableTLS},
		EncryptionConfig: defaultEncryption(encKey),
	}, nil
}

func configureRemPipeline() (*configs.REMConfig, error) {
	name := "rem_default"

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("REM Pipeline Name").Value(&name),
		).Title("REM Pipeline"),
	).Run()
	if err != nil {
		return nil, err
	}

	return &configs.REMConfig{
		Enable: true,
		Name:   name,
	}, nil
}

func defaultEncryption(key string) implanttypes.EncryptionsConfig {
	return implanttypes.EncryptionsConfig{
		{Type: "aes", Key: key},
		{Type: "xor", Key: key},
	}
}

func buildNotifyConfig(notifyType string, params map[string]string) *configs.NotifyConfig {
	cfg := &configs.NotifyConfig{Enable: true}
	switch notifyType {
	case "lark":
		cfg.Lark = &configs.LarkConfig{Enable: true, WebHookUrl: params["webhook_url"]}
	case "telegram":
		chatID, _ := strconv.ParseInt(params["chat_id"], 10, 64)
		cfg.Telegram = &configs.TelegramConfig{Enable: true, APIKey: params["api_key"], ChatID: chatID}
	case "dingtalk":
		cfg.DingTalk = &configs.DingTalkConfig{Enable: true, Token: params["token"], Secret: params["secret"]}
	case "serverchan":
		cfg.ServerChan = &configs.ServerChanConfig{Enable: true, URL: params["url"]}
	case "pushplus":
		cfg.PushPlus = &configs.PushPlusConfig{Enable: true, Token: params["token"], Topic: params["topic"], Channel: params["channel"]}
	}
	return cfg
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
