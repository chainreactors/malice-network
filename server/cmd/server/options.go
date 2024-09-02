package main

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	configs "github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/root"
	"github.com/jessevdk/go-flags"
)

var (
	ErrUnknownCommand  = errors.New("unknown command")
	ErrUnknownOperator = errors.New("unknown operator")
)

type Options struct {
	Config      string               `long:"config" description:"Path to config file"`
	IP          string               `short:"i" long:"ip" description:"external ip address, -i 123.123.123.123"`
	Daemon      bool                 `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec       bool                 `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA          string               `long:"ca" description:"Path to CA file" config:"ca"`
	Debug       bool                 `long:"debug" description:"Debug mode" config:"debug"`
	UserCmd     root.UserCommand     `command:"user" description:"User commands" `
	ListenerCmd root.ListenerCommand `command:"listener" description:"Listener commands" `

	// configs
	Server    *configs.ServerConfig   `config:"server" default:""`
	Listeners *configs.ListenerConfig `config:"listeners" default:""`

	localRpc *root.RootClient
}

func (opt *Options) Execute(args []string, parser *flags.Parser) error {
	if parser.Active == nil {
		return nil
	}
	var err error
	opt.localRpc, err = root.NewRootClient(opt.Server.Address())
	if err != nil {
		return err
	}
	if parser.Active.Name == opt.UserCmd.Name() {
		if parser.Active.Active == nil {
			return ErrUnknownOperator
		}
		return opt.localRpc.Execute(&opt.UserCmd, &rootpb.Operator{
			Name: opt.UserCmd.Name(),
			Op:   parser.Active.Active.Name,
			Args: args,
		})
	}
	if parser.Active.Name == opt.ListenerCmd.Name() {
		if parser.Active.Active == nil {
			return ErrUnknownOperator
		}

		return opt.localRpc.Execute(&opt.ListenerCmd, &rootpb.Operator{
			Name: opt.ListenerCmd.Name(),
			Op:   parser.Active.Active.Name,
			Args: args,
		})
	}
	return ErrUnknownCommand
}

func (opt *Options) InitUser() error {
	if has, err := db.HasOperator(mtls.Client); err != nil {
		return err
	} else if has {
		return nil
	}

	client, err := root.NewRootClient(opt.Server.Address())
	if err != nil {
		return err
	}
	err = client.Execute(&opt.UserCmd, &rootpb.Operator{
		Name: "user",
		Op:   "add",
		Args: []string{"admin"},
	})
	if err != nil {
		return err
	}
	return nil
}

func (opt *Options) InitListener() error {
	if has, err := db.HasOperator(mtls.Listener); err != nil {
		return err
	} else if has {
		return nil
	}

	client, err := root.NewRootClient(opt.Server.Address())
	if err != nil {
		return err
	}
	err = client.Execute(&opt.ListenerCmd, &rootpb.Operator{
		Name: "listener",
		Op:   "add",
		Args: []string{"listener"},
	})
	if err != nil {
		return err
	}
	return nil
}
