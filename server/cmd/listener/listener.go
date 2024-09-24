package main

import (
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/listener"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
)

//go:generate protoc -I proto/ proto/client/clientpb/client.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/client/rootpb/root.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/implant/implantpb/implant.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/listener/lispb/listener.proto --go_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/services/clientrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/
//go:generate protoc -I proto/ proto/services/listenerrpc/service.proto --go_out=paths=source_relative:proto/ --go-grpc_out=paths=source_relative:proto/

type Options struct {
	Config string `long:"config" description:"Path to config file"`
	Daemon bool   `long:"daemon" description:"Run as a daemon" config:"daemon"`
	CA     string `long:"ca" description:"Path to CA file" config:"ca" default:"ca.pem"`
	Debug  bool   `long:"debug" description:"Debug mode" config:"debug"`

	Listeners *configs.ListenerConfig `config:"listeners"`
}

func init() {
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yaml.Driver)
}

func banner() string {
	return "IoM listener"
}

func Execute() {
	var opt Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.Usage = banner()

	// load config
	err = configs.LoadConfig(configs.ListenerConfigFileName, &opt)
	if err != nil {
		logs.Log.Debugf("cannot load config , %s ", err.Error())
		return
	}

	_, err = parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			logs.Log.Error(err.Error())
		}
		return
	}
	if opt.Debug {
		logs.Log.SetLevel(logs.Debug)
	}
	if opt.Config != "" {
		err = configs.LoadConfig(opt.Config, &opt)
		if err != nil {
			logs.Log.Errorf("cannot load config , %s ", err.Error())
			return
		}
	}

	// start listeners
	if opt.Listeners != nil {
		// init forwarder
		clientConf, err := mtls.ReadConfig(opt.Listeners.Name + ".yaml")
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
		err = listener.NewListener(clientConf, opt.Listeners)
		if err != nil {
			logs.Log.Error(err.Error())
			return
		}
	}
}

func main() {
	Execute()
	select {}
}
