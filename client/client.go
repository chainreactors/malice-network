package main

//go:generate protoc -I ../proto/ ../proto/client/clientpb/client.proto --go_out=paths=source_relative:../helper/proto/
//go:generate protoc -I ../proto/ ../proto/client/rootpb/root.proto --go_out=paths=source_relative:../helper/proto/
//go:generate protoc -I ../proto/ ../proto/implant/implantpb/implant.proto --go_out=paths=source_relative:../helper/proto/
//go:generate protoc -I ../proto/ ../proto/implant/implantpb/module.proto --go_out=paths=source_relative:../helper/proto/
//go:generate protoc -I ../proto/ ../proto/services/clientrpc/service.proto --go_out=paths=source_relative:../helper/proto/ --go-grpc_out=paths=source_relative:../helper/proto/
//go:generate protoc -I ../proto/ ../proto/services/listenerrpc/service.proto --go_out=paths=source_relative:../helper/proto/ --go-grpc_out=paths=source_relative:../helper/proto/

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/cmd/cli"
)

func main() {
	err := cli.Start()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
}
