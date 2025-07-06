package server

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/server/internal/saas"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
	"os"
	"os/signal"
	"syscall"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/assets"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/listener"
	"github.com/chainreactors/malice-network/server/rpc"
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

func Execute(defaultConfig []byte) error {
	var opt Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.SubcommandsOptional = true
	args, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return nil
	}

	filename := configs.FindConfig(opt.Config)
	if filename == "" {
		err = os.WriteFile(configs.ServerConfigFileName, defaultConfig, 0644)
		if err != nil {
			return err
		}
		logs.Log.Warnf("config file not found, created default config %s", configs.ServerConfigFileName)
		filename = configs.ServerConfigFileName
	}

	// load config
	err = configutil.LoadConfig(filename, &opt)
	if err != nil {
		return fmt.Errorf("cannot load config , %s", err.Error())
	}
	if parser.Active != nil {
		err = opt.Execute(args, parser)
		if err != nil {
			logs.Log.Error(err)
		}
		return nil
	}
	configs.CurrentServerConfigFilename = filename
	// load config
	if opt.Debug {
		logs.Log.SetLevel(logs.DebugLevel)
	}
	err = opt.Validate()
	if err != nil {
		return fmt.Errorf("cannot validate config , %s", err.Error())
	}
	err = saas.RegisterLicense()
	if err != nil {
		return fmt.Errorf("register community license error %v", err)
	}
	if !opt.ListenerOnly && opt.Server.Enable {
		db.Client = db.NewDBClient()
		core.NewBroker()
		core.NewSessions()
		if opt.IP != "" {
			logs.Log.Infof("manually specified IP: %s will override %s config: %s", opt.IP, filename, opt.Server.IP)
			opt.Server.IP = opt.IP
			config.Set("server.ip", opt.IP)
		}

		if opt.Server.IP == "" {
			return fmt.Errorf("IP address not set, please set config.yaml `ip: [server_ip]` or `./malice_network -i [server_ip]`")
		}

		err = core.EventBroker.InitService(opt.Server.NotifyConfig)
		if err != nil {
			return fmt.Errorf("cannot init notifier , %s", err.Error())
		}
		err = certutils.GenerateRootCert()
		if err != nil {
			return fmt.Errorf("cannot init root ca , %s", err.Error())
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
			return fmt.Errorf("cannot start grpc , %s", err.Error())
		}

		err = opt.InitUser()
		if err != nil {
			return err
		}
		err = opt.InitListener()
		if err != nil {
			return err
		}
	}

	if !opt.ServerOnly && opt.Listeners.Enable {
		logs.Log.Importantf("[listener] listener config enabled, Starting listeners")
		if opt.IP != "" {
			logs.Log.Infof("manually specified IP: %s will override %s config: %s", opt.IP, filename, opt.Server.IP)
			opt.Listeners.IP = opt.IP
			config.Set("listeners.ip", opt.IP)
		}
		err := StartListener(opt.Listeners)
		if err != nil {
			return err
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
