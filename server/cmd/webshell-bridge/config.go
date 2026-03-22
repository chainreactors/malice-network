package main

// Config holds the bridge configuration.
type Config struct {
	AuthFile     string // path to listener.auth mTLS certificate
	ServerAddr   string // optional server address override
	ListenerName string // listener name for registration
	ListenerIP   string // listener external IP
	PipelineName string // pipeline name
	WebshellURL  string // webshell URL (http:// or https://)
	StageToken   string // auth token for X-Stage requests (must match webshell's STAGE_TOKEN)
	DLLPath      string // optional path to bridge DLL for auto-loading
	DepsDir      string // optional dir containing dependency jars (e.g., jna.jar) for auto-delivery
	Debug        bool   // enable debug logging
}

// WebshellHTTPURL returns the webshell URL for HTTP requests.
func (c *Config) WebshellHTTPURL() string {
	return c.WebshellURL
}
