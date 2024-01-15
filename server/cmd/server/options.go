package main

import (
	configs "github.com/chainreactors/malice-network/server/internal/configs"
)

type Options struct {
	Config    string                  `long:"config" description:"Path to config file"`
	Daemon    bool                    `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec     bool                    `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA        string                  `long:"ca" description:"Path to CA file" config:"ca"`
	User      string                  `long:"user" description:"User name" config:"user"`
	Debug     bool                    `long:"debug" description:"Debug mode" config:"debug"`
	Server    *configs.ServerConfig   `config:"server" default:""`
	Listeners *configs.ListenerConfig `config:"listeners" default:""`
}
