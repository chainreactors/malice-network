package main

import (
	configs "github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/root"
)

type Options struct {
	Config      string                  `long:"config" description:"Path to config file"`
	Daemon      bool                    `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec       bool                    `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA          string                  `long:"ca" description:"Path to CA file" config:"ca"`
	UserCmd     *root.UserCommand       `command:"user" description:"User commands" `
	ListenerCmd *root.ListenerCommand   `command:"listener" description:"Listener commands" `
	Debug       bool                    `long:"debug" description:"Debug mode" config:"debug"`
	Server      *configs.ServerConfig   `config:"server" default:""`
	Listeners   *configs.ListenerConfig `config:"listeners" default:""`
}

func (opt *Options) Command() root.Command {
	if opt.UserCmd != nil {
		return opt.UserCmd
	} else if opt.ListenerCmd != nil {
		return opt.ListenerCmd
	}
	return nil
}
