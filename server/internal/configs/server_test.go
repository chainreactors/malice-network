package configs

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"testing"
)

type Options struct {
	Config    string          `long:"config" description:"Path to config file"`
	Daemon    bool            `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec     bool            `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA        string          `long:"ca" description:"Path to CA file" config:"ca"`
	Server    *ServerConfig   `config:"server" default:""`
	Listeners *ListenerConfig `config:"listeners" default:""`
}

func TestGetConfig(t *testing.T) {
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yaml.Driver)
	var opt Options
	err := LoadConfig(ServerConfigFileName, &opt)
	if err != nil {
		logs.Log.Debugf("cannot load config , %s ", err.Error())
	}
	//max = GetConfig("server")
	fmt.Println(config.Get("server.config"))
}

func TestInitConfig(t *testing.T) {
	InitConfig()
}
