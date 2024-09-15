package main

//go:generate protoc -I ../proto/ ../proto/client/clientpb/client.proto --go_out=paths=source_relative:../proto/
//go:generate protoc -I ../proto/ ../proto/client/rootpb/root.proto --go_out=paths=source_relative:../proto/
//go:generate protoc -I ../proto/ ../proto/implant/implantpb/implant.proto --go_out=paths=source_relative:../proto/
//go:generate protoc -I ../proto/ ../proto/listener/lispb/listener.proto --go_out=paths=source_relative:../proto/
//go:generate protoc -I ../proto/ ../proto/services/clientrpc/service.proto --go_out=paths=source_relative:../proto/ --go-grpc_out=paths=source_relative:../proto/
//go:generate protoc -I ../proto/ ../proto/services/listenerrpc/service.proto --go_out=paths=source_relative:../proto/ --go-grpc_out=paths=source_relative:../proto/

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/cli"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/utils"
)

func init() {
	logs.Log.SetFormatter(utils.DefaultLogStyle)
	core.Log.SetFormatter(utils.DefaultLogStyle)
}

func main() {
	err := cli.StartConsole()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
}
