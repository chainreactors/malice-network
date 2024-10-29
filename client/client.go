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
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/client/repl"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

func init() {
	logs.Log.SetFormatter(repl.DefaultLogStyle)
	core.Log.SetFormatter(repl.DefaultLogStyle)
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yaml.Driver)
}

func main() {
	err := cli.Start()
	if err != nil {
		logs.Log.Errorf(err.Error())
		return
	}
}
