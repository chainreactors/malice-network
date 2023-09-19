package cmd

type ServerOptions struct {
	Daemon bool   `long:"daemon" description:"Run as a daemon"`
	Config string `long:"config" description:"Path to config file"`
	Opsec  string `long:"opsec" description:"Path to opsec file"`
	CA     string `long:"ca" description:"Path to CA file"`
}
