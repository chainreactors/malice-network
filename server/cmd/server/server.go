package server

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	crConfig "github.com/chainreactors/malice-network/helper/utils/config"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/listener"
	"github.com/chainreactors/malice-network/server/rpc"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
	"os"
	"os/signal"
	"syscall"
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
	codenames.SetupCodenames(configs.ServerRootPath)
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
		confStr := crConfig.InitDefaultConfig(&opt, 0)
		err := os.WriteFile(opt.Config, confStr, 0644)
		if err != nil {
			logs.Log.Errorf("cannot write default config , %s ", err.Error())
			return
		}
		logs.Log.Warnf("config file not found, created default config %s", opt.Config)
	}

	// load config
	err = crConfig.LoadConfig(opt.Config, &opt)
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
		logs.Log.SetLevel(logs.Debug)
	}
	err = opt.Validate()
	if err != nil {
		logs.Log.Errorf(err.Error())
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
				logs.Log.Errorf("cannot recover session %s , %s ", session.SessionId, err.Error())
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
