package cmd

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/transport"
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
	}

	if opt.Server.GRPCEnable {
		StartGrpc(opt.Server.GRPCHost, opt.Server.GRPCPort)
	}
}

// Start - Starts the server console
func StartGrpc(host string, port uint16) {
	_, _, err := transport.StartClientListener(host, port)
	if err != nil {
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
	select {}
}

func Banner() string {
	return ""
}
