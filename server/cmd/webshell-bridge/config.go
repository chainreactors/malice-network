package main

// Config holds the bridge configuration.
type Config struct {
	AuthFile     string // path to listener.auth mTLS certificate
	ServerAddr   string // optional server address override
	ListenerName string // listener name for registration
	ListenerIP   string // listener external IP
	PipelineName string // pipeline name
	Suo5URL      string // suo5 webshell URL (e.g. suo5://target/suo5.jsp)
	DLLAddr      string // target-side malefic bind DLL address (e.g. 127.0.0.1:13338)
	Debug        bool   // enable debug logging
}
