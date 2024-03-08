package main

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/certs"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/listener"
	"github.com/chainreactors/malice-network/server/rpc"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
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
}

func Execute() {
	var opt Options
	var err error
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
	_, _, err = certs.ServerGenerateCertificate("root", true, opt.Listeners.Auth)
	if err != nil {
		logs.Log.Errorf("cannot init root ca , %s ", err.Error())
		return
	}
	if opt.Debug {
		logs.Log.SetLevel(logs.Debug)
	}

	err = StartGrpc(opt.Server.GRPCPort)
	if err != nil {
		logs.Log.Errorf("cannot start grpc , %s ", err.Error())
		return
	}
	// start listeners
	if opt.Listeners != nil {
		// init forwarder
		err := listener.NewListener(opt.Listeners)
		if err != nil {
			logs.Log.Errorf("cannot start listeners , %s ", err.Error())
			return
		}
	}
}

// Start - Starts the server console
func StartGrpc(port uint16) error {
	// start grpc

	// start alive session
	db.Client = db.NewDBClient()
	dbSession := db.Session()
	sessions, err := models.FindActiveSessions(dbSession)
	if err != nil {
		return err
	}
	if len(sessions) > 0 {
		for _, session := range sessions {
			registerSession := core.NewSession(session.ToProtobuf())
			core.Sessions.Add(registerSession)
			//tasks, err := models.FindTasksWithNonOneCurTotal(dbSession, session)
			//if err != nil {
			//	logs.Log.Errorf("cannot find tasks in db , %s ", err.Error())
			//}
			//for _, task := range tasks {
		}
	}

	_, _, err = rpc.StartClientListener(port)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	Execute()
	select {}
}
