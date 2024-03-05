package main

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/listener/lispb"
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
	"os"
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
	parser.Usage = Banner()

	// load config
	err = configs.LoadConfig(configs.ServerConfigFileName, &opt)
	if err != nil {
		logs.Log.Warnf("cannot load config , %s ", err.Error())
	}
	parser.SubcommandsOptional = true
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
	_, _, err = certs.ServerGenerateCertificate("root", true, opt.Listeners.Auth)
	if err != nil {
		logs.Log.Errorf("cannot init root ca , %s ", err.Error())
		return
	}
	if opt.Debug {
		logs.Log.SetLevel(logs.Debug)
	}
	// start grpc
	StartGrpc(opt.Server.GRPCPort)

	client, conn, err := certs.NewRootClient()
	defer conn.Close()
	// init operator
	if opt.User.Add.Name != "" {
		addReq := &clientpb.LoginReq{
			Name:  opt.User.Add.Name,
			Token: "",
			Host:  "localhost",
			Port:  5004,
		}
		_, err := client.AddClient(context.Background(), addReq)
		if err != nil {
			logs.Log.Errorf("cannot add user , %s ", err.Error())
			return
		}
		logs.Log.Importantf("user %s added", opt.User.Add.Name)
		os.Exit(0)
	}

	if opt.User.Del.Name != "" {
		addReq := &clientpb.LoginReq{
			Name:  opt.User.Del.Name,
			Token: "",
			Host:  "localhost",
			Port:  5004,
		}
		_, err = client.RemoveClient(context.Background(), addReq)
		if err != nil {
			logs.Log.Errorf("cannot delete user , %s ", err.Error())
			return
		}
		logs.Log.Importantf("user %s deleted", opt.User.Del.Name)
		os.Exit(0)
	}
	if opt.User.List.Called != false {
		clients, err := client.ListClients(context.Background(), &clientpb.Empty{})
		if err != nil {
			logs.Log.Errorf("cannot list users , %s ", err.Error())
			return
		}
		for _, c := range clients.Clients {
			fmt.Println("User Name:", c.Name)
		}
		os.Exit(0)
	}

	// listener operation
	if opt.Listener.Add.Name != "" {
		addReq := &lispb.RegisterListener{
			Name: opt.Listener.Add.Name,
			Addr: "",
			Host: "localhost",
		}
		_, err := client.AddListener(context.Background(), addReq)
		if err != nil {
			logs.Log.Errorf("cannot add listener , %s ", err.Error())
			return
		}
		logs.Log.Importantf("listener %s added", opt.Listener.Add.Name)
		os.Exit(0)
	}

	if opt.Listener.Del.Name != "" {
		addReq := &lispb.RegisterListener{
			Name: opt.Listener.Del.Name,
			Addr: "",
			Host: "localhost",
		}
		_, err = client.RemoveListener(context.Background(), addReq)
		if err != nil {
			logs.Log.Errorf("cannot delete listener , %s ", err.Error())
			return
		}
		logs.Log.Importantf("listener %s deleted", opt.Listener.Del.Name)
		os.Exit(0)
	}

	if opt.Listener.List.Called != false {
		listeners, err := client.ListListeners(context.Background(), &clientpb.Empty{})
		if err != nil {
			logs.Log.Errorf("cannot list operators , %s ", err.Error())
			return
		}
		for _, l := range listeners.Listeners {
			fmt.Println("Listener Name:", l.Id)
		}
		os.Exit(0)
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
			//tasks, err := models.FindTasksWithNonOneCurTotal(dbSession, session)
			//if err != nil {
			//	logs.Log.Errorf("cannot find tasks in db , %s ", err.Error())
			//}
			//for _, task := range tasks {
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
