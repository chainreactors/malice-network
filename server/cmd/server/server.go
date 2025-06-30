package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/utils"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/assets"
	"github.com/chainreactors/malice-network/server/build"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/listener"
	"github.com/chainreactors/malice-network/server/rpc"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func init() {
	err := configs.InitConfig()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yaml.Driver)
	codenames.SetupCodenames()
	assets.SetupGithubFile()
}

func Execute() {
	var opt Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.SubcommandsOptional = true
	args, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return
	}
	if !fileutils.Exist(opt.Config) {
		confStr := configutil.InitDefaultConfig(&opt, 0)
		err := os.WriteFile(opt.Config, confStr, 0644)
		if err != nil {
			logs.Log.Errorf("cannot write default config , %s ", err.Error())
			return
		}
		logs.Log.Warnf("config file not found, created default config %s", opt.Config)
	}
	config.WithOptions(config.WithHookFunc(func(event string, c *config.Config) {
		if strings.HasPrefix(event, "set.") {
			open, err := os.OpenFile(opt.Config, os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				logs.Log.Errorf("cannot open config , %s ", err.Error())
				return
			}
			defer open.Close()
			_, err = config.DumpTo(open, config.Yaml)
			if err != nil {
				logs.Log.Errorf("cannot dump config , %s ", err.Error())
				return
			}
		}
	}))
	// load config
	err = configutil.LoadConfig(opt.Config, &opt)
	if err != nil {
		logs.Log.Warnf("cannot load config , %s ", err.Error())
		return
	}
	if parser.Active != nil {
		err = opt.Execute(args, parser)
		if err != nil {
			logs.Log.Error(err)
		}
		return
	}
	configs.CurrentServerConfigFilename = opt.Config
	// load config
	if opt.Debug {
		logs.Log.SetLevel(logs.DebugLevel)
	}
	err = opt.Validate()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
	err = RegisterLicense(opt)
	if err != nil {
		logs.Log.Errorf("register community license error %v", err)
		return
	}
	if opt.Server.Enable {
		db.Client = db.NewDBClient()
		core.NewBroker()
		core.NewSessions()
		if opt.IP != "" {
			logs.Log.Infof("manually specified IP: %s will override %s config: %s", opt.IP, opt.Config, opt.Server.IP)
			opt.Server.IP = opt.IP
			config.Set("server.ip", opt.IP)
		}

		if opt.Server.IP == "" {
			logs.Log.Errorf("IP address not set, please set config.yaml `ip: [server_ip]` or `./malice_network -i [server_ip]`")
			return
		}

		err = core.EventBroker.InitService(opt.Server.NotifyConfig)
		if err != nil {
			logs.Log.Errorf("cannot init notifier , %s ", err.Error())
			return
		}
		err = certutils.GenerateRootCert()
		if err != nil {
			logs.Log.Errorf("cannot init root ca , %s ", err.Error())
			return
		}
		//if opt.Daemon == true {
		//	err = RecoverAliveSession()
		//	if err != nil {
		//		logs.Log.Errorf("cannot start alive session , %s ", err.Error())
		//		return
		//	}
		//	rpc.DaemonStart(opt.Server, opt.Listeners)
		//}

		err = StartGrpc(fmt.Sprintf("%s:%d", opt.Server.GRPCHost, opt.Server.GRPCPort))
		if err != nil {
			logs.Log.Errorf("cannot start grpc , %s ", err.Error())
			return
		}

		err = opt.InitUser()
		if err != nil {
			logs.Log.Errorf(err.Error())
			return
		}
		err = opt.InitListener()
		if err != nil {
			logs.Log.Errorf(err.Error())
			return
		}
	}

	if opt.Listeners.Enable {
		logs.Log.Importantf("[listener] listener config enabled, Starting listeners")
		if opt.IP != "" {
			logs.Log.Infof("manually specified IP: %s will override %s config: %s", opt.IP, opt.Config, opt.Server.IP)
			opt.Listeners.IP = opt.IP
			config.Set("listeners.ip", opt.IP)
		}
		err := StartListener(opt.Listeners)
		if err != nil {
			logs.Log.Errorf(err.Error())
			return
		}
	}

	_, cancel := context.WithCancel(context.Background())
	go func() {
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		<-c
		logs.Log.Importantf("exit signal, save stat and exit")

		signal.Stop(c)

		for _, session := range core.Sessions.All() {
			session.Save()
		}
		//pprof.StopCPUProfile()
		core.GlobalTicker.RemoveAll()
		cancel()
		os.Exit(0)
	}()
	err = ReDownloadSaasArtifact()
	if err != nil {
		logs.Log.Errorf("recover download saas artifact error %v", err)
	}
	select {}
}

// Start - Starts the server console
func StartGrpc(address string) error {
	// start alive session
	err := RecoverAliveSession()
	if err != nil {
		return err
	}

	_, _, err = rpc.StartClientListener(address)
	if err != nil {
		return err
	}
	return nil
}

func RecoverAliveSession() error {
	// start alive session
	sessions, err := db.FindAliveSessions()
	if err != nil {
		return err
	}

	if len(sessions) > 0 {
		logs.Log.Debugf("recover %d sessions", len(sessions))
		for _, session := range sessions {
			newSession, err := core.RecoverSession(session)
			if err != nil {
				logs.Log.Errorf("cannot recover session %s , %s ", session.SessionID, err.Error())
				continue
			}
			core.Sessions.Add(newSession)
		}
	}
	return nil
}

func ReDownloadSaasArtifact() error {
	saasConfig := configs.GetSaasConfig()
	artifacts, err := db.GetArtifactWithSaas()
	if err != nil {
		return err
	}
	if len(artifacts) > 0 {
		for _, artifact := range artifacts {
			if artifact.Status != consts.BuildStatusCompleted && artifact.Status != consts.BuildStatusFailure {
				go func() {
					statusUrl := fmt.Sprintf("%s/api/build/status/%s", saasConfig.Url, artifact.Name)
					downloadUrl := fmt.Sprintf("%s/api/build/download/%s", saasConfig.Url, artifact.Name)
					_, _, err = build.CheckAndDownloadArtifact(statusUrl, downloadUrl, saasConfig.Token, artifact, 0, 0)
					if err != nil {
						return
					}
				}()
			}
		}
	}
	return nil
}

func StartListener(opt *configs.ListenerConfig) error {
	if listenerConf, err := mtls.ReadConfig(opt.Auth); err != nil {
		return err
	} else {
		err = listener.NewListener(listenerConf, opt)
		if err != nil {
			return err
		}
	}
	return nil
}

func RegisterLicense(opt Options) error {
	saasConfig := configs.GetSaasConfig()
	if saasConfig.Token == "" && saasConfig.Enable {
		licenseUrl := fmt.Sprintf("%s/api/license/", saasConfig.Url)

		// 获取机器码作为用户名
		machineID := utils.GetMachineID()
		username := fmt.Sprintf("machine_%s", machineID)

		licenseData := map[string]interface{}{
			"username":    username,                // 使用机器码作为用户名
			"email":       "community@example.com", // 默认邮箱
			"type":        "community",             // 默认为community类型
			"max_builds":  0,                       // community类型无限制
			"build_count": 0,
		}

		jsonData, err := json.Marshal(licenseData)
		if err != nil {
			return fmt.Errorf("failed to marshal license data: %v", err)
		}

		req, err := http.NewRequest("POST", licenseUrl, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send HTTP request: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
		}

		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}

		if success, ok := response["success"].(bool); !ok || !success {
			return fmt.Errorf("license registration failed: %v", response)
		}

		if licenseData, ok := response["license"].(map[string]interface{}); ok {
			if returnedToken, ok := licenseData["Token"].(string); ok {
				fmt.Printf("注册成功，返回的token: %s\n", returnedToken)
				saasConfig.Token = returnedToken
				err = configs.UpdateSaasConfig(saasConfig)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}
