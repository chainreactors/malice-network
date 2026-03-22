package main

// Config holds the bridge configuration.
type Config struct {
	AuthFile     string // path to listener.auth mTLS certificate
	ServerAddr   string // optional server address override
	ListenerName string // listener name for registration
	ListenerIP   string // listener external IP
	PipelineName string // pipeline name
	Suo5URL      string // suo5 webshell URL (e.g. suo5://target/suo5.jsp)
	StageToken   string // auth token for X-Stage requests (must match webshell's STAGE_TOKEN)
	Debug        bool   // enable debug logging
}

// WebshellHTTPURL converts the suo5:// URL to an http(s):// URL.
func (c *Config) WebshellHTTPURL() string {
	if len(c.Suo5URL) < 6 {
		return c.Suo5URL
	}
	switch {
	case len(c.Suo5URL) > 6 && c.Suo5URL[:6] == "suo5s:":
		return "https:" + c.Suo5URL[6:]
	case len(c.Suo5URL) > 5 && c.Suo5URL[:5] == "suo5:":
		return "http:" + c.Suo5URL[5:]
	default:
		return c.Suo5URL
	}
}
