package cmd

import (
	"github.com/chainreactors/malice-network/server/listener"
)

type Options struct {
	Config    string              `long:"config" description:"Path to config file"`
	Daemon    bool                `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec     bool                `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA        string              `long:"ca" description:"Path to CA file" config:"ca"`
	Server    ServerConfig        `config:"server"`
	Listeners *listener.Listeners `config:"listeners"`
}

type ServerConfig struct {
	GRPCPort uint16 `config:"grpc_port"`
	GRPCHost string `config:"grpc_host"`
}
