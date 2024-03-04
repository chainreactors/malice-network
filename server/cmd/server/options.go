package main

import (
	configs "github.com/chainreactors/malice-network/server/internal/configs"
)

type Options struct {
	Config    string                  `long:"config" description:"Path to config file"`
	Daemon    bool                    `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec     bool                    `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA        string                  `long:"ca" description:"Path to CA file" config:"ca"`
	User      userCommand             `command:"user" description:"User commands" `
	Listener  listenerCommand         `command:"listener" description:"Listener commands" `
	Debug     bool                    `long:"debug" description:"Debug mode" config:"debug"`
	Server    *configs.ServerConfig   `config:"server" default:""`
	Listeners *configs.ListenerConfig `config:"listeners" default:""`
}

type addCommand struct {
	Name string `long:"name" short:"n" description:"Name of the listener/user"`
}

type delCommand struct {
	Name string `long:"name" short:"n" description:"Name of the listener/user"`
}

type listCommand struct {
	Called bool `long:"called" short:"c" description:"List called listeners/users"`
}

// UserCommand - User command
type userCommand struct {
	Add  addCommand  `command:"add" description:"Add a user" subcommands-optional:"true" `
	Del  delCommand  `command:"del" description:"Delete a user" subcommands-optional:"true" `
	List listCommand `command:"list" description:"List all users"`
}

// ListenerCommand - Listener command
type listenerCommand struct {
	Add  addCommand  `command:"add" description:"Add a listener" subcommands-optional:"true" `
	Del  delCommand  `command:"del" description:"Delete a listener" subcommands-optional:"true" `
	List listCommand `command:"list" description:"List all listeners"`
}
