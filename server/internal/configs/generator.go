package configs

type GeneratorBasicConfig struct {
	Name       string   `yaml:"name" config:"name" default:"malefic"`
	Urls       []string `yaml:"urls" config:"urls" default:"[]"`
	Protocol   string   `yaml:"protocol" config:"protocol" default:"tcp"`
	TLS        bool     `yaml:"tls" config:"tls" default:"false"`
	Proxy      string   `yaml:"proxy" config:"proxy" default:""`
	Interval   int      `yaml:"interval" config:"interval" default:"5"`
	Jitter     float64  `yaml:"jitter" config:"jitter" default:"0.2"`
	CA         string   `yaml:"ca" config:"ca" default:""`
	Encryption string   `yaml:"encryption" config:"encryption" default:"aes"`
	Key        string   `yaml:"key" config:"key" default:"maliceofinternal"`
}

type GeneratorConfig struct {
	Basic GeneratorBasicConfig `yaml:"basic" config:"basic"`
}
