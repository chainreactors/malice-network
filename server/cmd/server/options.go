package server

import (
	"github.com/chainreactors/malice-network/server/configs"
	"github.com/chainreactors/malice-network/server/pipeline"
)

type Options struct {
	Config    string               `long:"config" description:"Path to config file"`
	Daemon    bool                 `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec     bool                 `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA        string               `long:"ca" description:"Path to CA file" config:"ca"`
	Server    configs.ServerConfig `config:"server"`
	Listeners *pipeline.Pipelines  `config:"listeners"`
}
