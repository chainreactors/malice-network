package main

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/server/internal/certs"
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
	db.Client = db.NewDBClient()
}

func Execute() {
	var opt Options
	var err error
	core.NewTicker()
	parser := flags.NewParser(&opt, flags.Default)
	parser.SubcommandsOptional = true
	args, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return
	}

	// load config
	err = configs.LoadConfig(opt.Config, &opt)
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

	if opt.IP != "" {
		logs.Log.Infof("manually specified IP: %s will override %s config: %s", opt.IP, opt.Config, opt.Server.IP)
		opt.Server.IP = opt.IP
	}

	if opt.Server.Enable {
		err = certs.GenerateRootCert()
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
		logs.Log.Importantf("listener config enabled, Starting listeners")
		err := StartListener(opt.Listeners)
		if err != nil {
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
			newSession := core.NewSession(session)
			err = newSession.Load()
			if err != nil {
				logs.Log.Debugf("cannot load session , %s ", err.Error())
			}
			tasks, taskID, err := db.FindTaskAndMaxTasksID(session.SessionId)
			if err != nil {
				logs.Log.Errorf("cannot find max task id , %s ", err.Error())
			}
			newSession.SetLastTaskId(uint32(taskID))
			for _, task := range tasks {
				newTask, err := db.ToTask(*task)
				if err != nil {
					logs.Log.Errorf("cannot convert task to core task , %s ", err.Error())
					continue
				}
				newSession.Tasks.Add(newTask)
			}
			core.Sessions.Add(newSession)
		}
	}
	go func() {
		err := db.UpdateSessionStatus()
		if err != nil {
			logs.Log.Errorf("cannot update session status , %s ", err.Error())
		}
	}()
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

func main() {
	Execute()
}
