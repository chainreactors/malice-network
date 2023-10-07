package main

import (
	"fmt"
	"github.com/chainreactors/malice-network/client/command"
	"github.com/chainreactors/malice-network/client/console"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:50001", grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to connect: %v", err)
	}
	defer conn.Close()
	rpc := clientrpc.NewMaliceRPCClient(conn)
	console.Start(rpc, command.BindCommands, false)
}
