package server

import (
	"errors"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/cryptography"
	"github.com/chainreactors/malice-network/helper/proto/client/rootpb"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/helper/utils/mtls"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/saas"
	"github.com/chainreactors/malice-network/server/listener"
	"github.com/chainreactors/malice-network/server/root"
	"github.com/chainreactors/malice-network/server/rpc"
	"github.com/gookit/config/v2"
	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

func (opt *Options) PrepareConfig(defaultConfig []byte) error {
	filename := configs.FindConfig(opt.Config)
	if filename == "" {
		err := os.WriteFile(configs.ServerConfigFileName, defaultConfig, 0644)
		if err != nil {
			return err
		}
		logs.Log.Warnf("config file not found, created default config %s", configs.ServerConfigFileName)
		filename = configs.ServerConfigFileName
	}

	config.WithOptions(config.WithHookFunc(func(event string, c *config.Config) {
		if strings.HasPrefix(event, "set.") {
			open, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				logs.Log.Errorf("cannot open config , %s ", err.Error())
				return
			}
			defer open.Close()
			_, err = config.DumpTo(open, config.Yaml)
			if err != nil {
				logs.Log.Errorf("cannot dump config , %s ", err.Error())
				return
			}
		}
	}))

	// load config
	err := configutil.LoadConfig(filename, opt)
	if err != nil {
		return fmt.Errorf("cannot load config , %s", err.Error())
	}

	configs.CurrentServerConfigFilename = filename
	// load config
	if opt.Debug {
		logs.Log.SetLevel(logs.DebugLevel)
	}
	err = opt.Validate()
	if err != nil {
		return fmt.Errorf("cannot validate config , %s", err.Error())
	}
	return nil
}

func (opt *Options) PrepareServer() error {
	db.Client = db.NewDBClient()
	err := saas.RegisterLicense()
	if err != nil {
		logs.Log.Warnf("register community license error %v", err)
	}
	core.NewBroker()
	core.NewSessions()
	if opt.IP != "" {
		logs.Log.Infof("manually specified IP: %s will override config: %s", opt.IP, opt.Server.IP)
		opt.Server.IP = opt.IP
		config.Set("server.ip", opt.IP)
	}

	if opt.Server.IP == "" {
		return fmt.Errorf("IP address not set, please set config.yaml `ip: [server_ip]` or `./malice_network -i [server_ip]`")
	}

	err = core.EventBroker.InitService(opt.Server.NotifyConfig)
	if err != nil {
		return fmt.Errorf("cannot init notifier , %s", err.Error())
	}
	err = certutils.GenerateRootCert()
	if err != nil {
		return fmt.Errorf("cannot init root ca , %s", err.Error())
	}
	cryptography.InitAES(opt.Server.EncryptionKey)
	//if opt.Daemon == true {
	//	err = RecoverAliveSession()
	//	if err != nil {
	//		logs.Log.Errorf("cannot start alive session , %s ", err.Error())
	//		return
	//	}
	//	rpc.DaemonStart(opt.Server, opt.Listeners)
	//}

	err = StartGrpc(fmt.Sprintf("%s:%d", opt.Server.GRPCHost, opt.Server.GRPCPort))
	if err != nil {
		return fmt.Errorf("cannot start grpc , %s", err.Error())
	}

	err = opt.InitUser()
	if err != nil {
		return err
	}
	err = opt.InitListener()
	if err != nil {
		return err
	}
	return nil
}

func (opt *Options) PrepareListener() error {
	logs.Log.Importantf("[listener] listener config enabled, Starting listeners")
	if opt.IP != "" {
		logs.Log.Infof("manually specified IP: %s will override config: %s", opt.IP, opt.Server.IP)
		opt.Listeners.IP = opt.IP
		config.Set("listeners.ip", opt.IP)
	}
	err := StartListener(opt.Listeners)
	if err != nil {
		return err
	}
	return nil
}

func (opt *Options) Handler() error {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	<-c
	logs.Log.Importantf("exit signal, save stat and exit")

	signal.Stop(c)

	for _, session := range core.Sessions.All() {
		err := session.Save()
		if err != nil {
			return err
		}
	}
	//pprof.StopCPUProfile()
	core.GlobalTicker.RemoveAll()
	os.Exit(0)
	return nil
}

// Start - Starts the server console
func StartGrpc(address string) error {
	// start alive session
	err := RecoverAliveSession()
	if err != nil {
		return err
	}

	_, _, err = rpc.StartClientListener(address)
	if err != nil {
		return err
	}
	return nil
}

func RecoverAliveSession() error {
	// start alive session
	sessions, err := db.FindAliveSessions()
	if err != nil {
		return err
	}

	if len(sessions) > 0 {
		logs.Log.Debugf("recover %d sessions", len(sessions))
		for _, session := range sessions {
			newSession, err := core.RecoverSession(session)
			if err != nil {
				logs.Log.Errorf("cannot recover session %s , %s ", session.SessionID, err.Error())
				continue
			}
			core.Sessions.Add(newSession)
		}
	}
	return nil
}

func StartListener(opt *configs.ListenerConfig) error {
	if listenerConf, err := mtls.ReadConfig(opt.Auth); err != nil {
		return err
	} else {
		err = listener.NewListener(listenerConf, opt)
		if err != nil {
			return err
		}
	}
	return nil
}
