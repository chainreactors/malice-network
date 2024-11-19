package configs

type GeneratorBasicConfig struct {
	Name       string   `yaml:"name" config:"name" default:"malefic"`
	Targets    []string `yaml:"targets" config:"targets" default:"[]"`
	Protocol   string   `yaml:"protocol" config:"protocol" default:"tcp"`
	TLS        bool     `yaml:"tls" config:"tls" default:"false"`
	Proxy      string   `yaml:"proxy" config:"proxy" default:""`
	Interval   int      `yaml:"interval" config:"interval" default:"5"`
	Jitter     float64  `yaml:"jitter" config:"jitter" default:"0.2"`
	CA         string   `yaml:"ca" config:"ca" default:""`
	Encryption string   `yaml:"encryption" config:"encryption" default:"aes"`
	Key        string   `yaml:"key" config:"key" default:"maliceofinternal"`
}

type GeneratePulseConfig struct {
	Target     string `yaml:"target" config:"target" default:""`
	Encryption string `yaml:"encryption" config:"encryption" default:"aes"`
	Key        string `yaml:"key" config:"key" default:"maliceofinternal"`
}

type GeneratorConfig struct {
	Basic  *GeneratorBasicConfig  `yaml:"basic" config:"basic"`
	Pulse  *GeneratePulseConfig   `yaml:"pulse" config:"pulse"`
	Extras map[string]interface{} `yaml:",inline"`
}
