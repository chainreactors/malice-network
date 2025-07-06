package server

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/proto/client/rootpb"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/root"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
	"os"
)

var (
	ErrUnknownCommand  = errors.New("unknown command")
	ErrUnknownOperator = errors.New("unknown operator")
)

type Options struct {
	Config       string               `short:"c" long:"config" default:"config.yaml" description:"Path to config file"`
	IP           string               `short:"i" long:"ip" description:"external ip address, -i 123.123.123.123"`
	ServerOnly   bool                 `long:"server-only" description:"Run server only"`
	ListenerOnly bool                 `long:"listener-only" description:"Run listener only"`
	Daemon       bool                 `long:"daemon" description:"Run as a daemon"`
	Opsec        bool                 `long:"opsec" description:"Path to opsec file"`
	Debug        bool                 `long:"debug" description:"Debug mode" config:"debug"`
	UserCmd      root.UserCommand     `command:"user" description:"User commands" `
	ListenerCmd  root.ListenerCommand `command:"listener" description:"Listener commands" `
	License      root.LicenseCmd      `command:"license" description:"License management"`

	// configs
	Server    *configs.ServerConfig   `config:"server" `
	Listeners *configs.ListenerConfig `config:"listeners" `

	localRpc *root.RootClient
}

func (opt *Options) Validate() error {
	if !opt.Server.Enable && !opt.Listeners.Enable {
		return errors.New("must enable one of server/listener ")
	}
	return nil
}

func (opt *Options) Execute(args []string, parser *flags.Parser) error {
	if parser.Active == nil {
		return nil
	}
	var err error
	opt.localRpc, err = root.NewRootClient(fmt.Sprintf("127.0.0.1:%d", opt.Server.GRPCPort))
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

	client, err := root.NewRootClient(fmt.Sprintf("127.0.0.1:%d", opt.Server.GRPCPort))
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

	client, err := root.NewRootClient(fmt.Sprintf("127.0.0.1:%d", opt.Server.GRPCPort))
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

// Save 保存配置到文件
func (opt *Options) Save() error {

	configToSave := struct {
		Server    *configs.ServerConfig   `yaml:"server"`
		Listeners *configs.ListenerConfig `yaml:"listeners"`
	}{
		Server:    opt.Server,
		Listeners: opt.Listeners,
	}

	data, err := yaml.Marshal(configToSave)
	if err != nil {
		return err
	}
	err = os.WriteFile(opt.Config, data, 0600)
	if err != nil {
		logs.Log.Errorf("Failed to write config %s", err)
		return err
	}
	return nil
}
