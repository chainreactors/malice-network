package cmd

import (
	"context"
	"fmt"
	"github.com/chainreactors/malice-network/server/transport"
	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc"
)

type ServerConfig struct {
	GRPCPort uint
	MTLSPort uint
}

func Execute() {
	var opt ServerOptions
	parser := flags.NewParser(&opt, flags.Default)
	parser.Usage = Banner()
	_, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Println(err.Error())
		}
		return
	}

}

// Start - Starts the server console
func StartGrpc() {
	_, _, err := transport.StartClientListener("0.0.0.0", 50001)
	if err != nil {
		return // If we fail to bind don't setup the Job
	}
	//ctxDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
	//	return ln.Dial()
	//})

	options := []grpc.DialOption{
		//ctxDialer,
		grpc.WithInsecure(), // This is an in-memory listener, no need for secure transport
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(transport.ClientMaxReceiveMessageSize)),
	}
	conn, err := grpc.DialContext(context.Background(), "bufnet", options...)
	if err != nil {
		//fmt.Printf(Warn+"Failed to dial bufnet: %s\n", err)
		return
	}
	defer conn.Close()

	//localRPC := clientrpc.NewMaliceRPCClient(conn)
	//if err := configs.CheckHTTPC2ConfigErrors(); err != nil {
	//	fmt.Printf(Warn+"Error in HTTP C2 config: %s\n", err)
	//}
	select {}
}

func Banner() string {
	return ""
}
