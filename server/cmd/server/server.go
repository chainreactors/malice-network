package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
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

//go:generate protoc -I proto/ proto/client/clientpb/client.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/client/rootpb/root.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/implant/implantpb/implant.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/listener/lispb/listener.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/services/clientrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/services/listenerrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/

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
}

func Execute() {
	var opt Options
	var err error
	core.NewTicker()
	parser := flags.NewParser(&opt, flags.Default)

	// load config
	err = configs.LoadConfig(configs.ServerConfigFileName, &opt)
	if err != nil {
		logs.Log.Warnf("cannot load config , %s ", err.Error())
	}
	parser.SubcommandsOptional = true
	sub, err := parser.Parse()
	if err != nil {
		if !errors.Is(err, flags.ErrHelp) {
			logs.Log.Error(err.Error())
		}
		return
	}

	err = opt.Execute(sub, parser)
	if err != nil {
		logs.Log.Error(err)
		return
	}
	if parser.Command.Active != nil {
		return
	}
	// load config
	if opt.Config != "" {
		err = configs.LoadConfig(opt.Config, &opt)
		if err != nil {
			logs.Log.Errorf("cannot load config , %s ", err.Error())
			return
		}
		configs.CurrentServerConfigFilename = opt.Config
	} else if opt.Server == nil {
		logs.Log.Errorf("null server config , %s ", err.Error())
	}
	if opt.Debug {
		logs.Log.SetLevel(logs.Debug)
	}

	db.Client = db.NewDBClient()
	_, _, err = certs.ServerGenerateCertificate("root", true, opt.Listeners.Auth)
	if err != nil {
		logs.Log.Errorf("cannot init root ca , %s ", err.Error())
		return
	}
	//if opt.Daemon == true {
	//	err = StartAliveSession()
	//	if err != nil {
	//		logs.Log.Errorf("cannot start alive session , %s ", err.Error())
	//		return
	//	}
	//	rpc.DaemonStart(opt.Server, opt.Listeners)
	//}

	err = StartGrpc(opt.Server.GRPCPort)
	if err != nil {
		logs.Log.Errorf("cannot start grpc , %s ", err.Error())
		return
	}

	// start listeners
	if opt.Listeners.Auth != "" {
		// init forwarder
		clientConf, err := listener.GenerateClientConfig(opt.Server, opt.Listeners)
		if err != nil {
			logs.Log.Errorf("init client failed, %s", err.Error())
			return
		}
		err = listener.NewListener(clientConf, opt.Listeners)
		if err != nil {
			logs.Log.Errorf("cannot start listeners , %s ", err.Error())
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
func StartGrpc(port uint16) error {
	// start alive session
	err := StartAliveSession()
	if err != nil {
		return err
	}

	_, _, err = rpc.StartClientListener(port)
	if err != nil {
		return err
	}
	return nil
}

func StartAliveSession() error {
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

func main() {
	Execute()
}
