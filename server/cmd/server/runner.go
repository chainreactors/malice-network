package server

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/generate"
	"github.com/chainreactors/malice-network/server/rpc"
	"github.com/gookit/config/v2"
	"github.com/jessevdk/go-flags"
)

func Execute() {
	var opt Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.Usage = Banner()

	err = config.Decode(&opt)
	if err != nil {
		logs.Log.Error(err.Error())
		return
	}

	_, err = parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			logs.Log.Error(err.Error())
		}
		return
	}

	if opt.Config != "" {
		err := config.LoadFiles(opt.Config)
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
		err = config.Decode(&opt)
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
	}

	// start grpc
	StartGrpc(opt.Server.GRPCPort)

	// start listeners
	if opt.Listeners != nil {
		// init forwarder
		opt.Listeners.Start()
	}

	// generate certs
	generate.InitRootCA()
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
