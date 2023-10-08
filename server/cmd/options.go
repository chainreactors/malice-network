package cmd

type Options struct {
	Config string `long:"config" description:"Path to config file"`
	Daemon bool   `long:"daemon" description:"Run as a daemon" config:"daemon"`
	Opsec  bool   `long:"opsec" description:"Path to opsec file" config:"opsec"`
	CA     string `long:"ca" description:"Path to CA file" config:"ca"`
	Server ServerConfig
}

type ServerConfig struct {
	GRPCPort   uint16 `config:"grpc_port"`
	GRPCHost   string `config:"grpc_host"`
	GRPCEnable bool   `config:"grpc_enable"`
	MTLSPort   uint16 `config:"mtls_port"`
	MTLSHost   string `config:"mtls_host"`
	MTLSEnable bool   `config:"mtls_enable"`
}
