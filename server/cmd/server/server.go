package main

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/core"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"github.com/chainreactors/malice-network/server/internal/generate"
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
	generate.GenerateRootCA()
	config.AddDriver(yaml.Driver)
}

func Execute() {
	var opt Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.Usage = Banner()

	// load config
	err = configs.LoadConfig(configs.ServerConfigFileName, &opt)
	if err != nil {
		logs.Log.Warnf("cannot load config , %s ", err.Error())
	}
	_, err = parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			logs.Log.Error(err.Error())
		}
		return
	}

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

	// start grpc
	StartGrpc(opt.Server.GRPCPort)

	// start listeners
	if opt.Listeners != nil {
		// init forwarder
		err := listener.NewListener(opt.Listeners, true)
		if err != nil {
			logs.Log.Errorf("cannot start listeners , %s ", err.Error())
			return
		}
	}

	// init operator
	if opt.User != "" {
		err = generate.ServerInitUserCert(opt.User)
		if err != nil {
			logs.Log.Errorf("cannot init operator , %s ", err.Error())
			return
		}
	}

	// start alive session
	dbSession := db.Session()
	sessions, err := models.FindActiveSessions(dbSession)
	if err != nil {
		logs.Log.Errorf("cannot find sessions in db , %s ", err.Error())
		return
	}
	if len(sessions) > 0 {
		for _, session := range sessions {
			registerSession := core.NewSession(session.ToProtobuf())
			core.Sessions.Add(registerSession)
		}
	}
}

// Start - Starts the server console
func StartGrpc(port uint16) {
	_, _, err := rpc.StartClientListener(port)
	if err != nil {
		logs.Log.Error(err.Error())
		return // If we fail to bind don't setup the Job
	}
	//ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
	//	return ln.Dial()
	//})

	//options := []grpc.DialOption{
	//	//ctxDialer,
	//	grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
	//	grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(constant.ClientMaxReceiveMessageSize)),
	//}
	//conn, err := grpc.DialContext(context.Background(), "bufnet", options...)
	//if err != nil {
	//	//fmt.Printf(Warn+"Failed to dial bufnet: %s\n", err)
	//	return
	//}
	//defer conn.Close()

	//localRPC := clientrpc.NewMaliceRPCClient(conn)
	//if err := configs.CheckHTTPC2ConfigErrors(); err != nil {
	//	fmt.Printf(Warn+"Error in HTTP C2 config: %s\n", err)
	//}
}

func Banner() string {
	return ""
}

func main() {
	Execute()
	select {}
}
