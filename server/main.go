package main

import (
	"github.com/chainreactors/malice-network/server/cmd/server"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
)

func init() {
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
	})
	config.AddDriver(yaml.Driver)
	err := config.LoadFiles("config.yaml")
	if err != nil {
		panic(err)
	}
}

func main() {
	server.Execute()
	select {}
}
