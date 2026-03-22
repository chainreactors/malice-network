package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chainreactors/logs"
)

func main() {
	cfg := &Config{}
	flag.StringVar(&cfg.AuthFile, "auth", "", "path to listener.auth mTLS certificate file")
	flag.StringVar(&cfg.ServerAddr, "server", "", "server address (overrides auth file)")
	flag.StringVar(&cfg.ListenerName, "listener", "webshell-listener", "listener name")
	flag.StringVar(&cfg.ListenerIP, "ip", "127.0.0.1", "listener external IP")
	flag.StringVar(&cfg.PipelineName, "pipeline", "", "pipeline name (auto-generated if empty)")
	flag.StringVar(&cfg.WebshellURL, "url", "", "webshell URL (e.g. http://target/shell.jsp)")
	flag.StringVar(&cfg.StageToken, "token", "", "auth token matching webshell's STAGE_TOKEN")
	flag.StringVar(&cfg.DLLPath, "dll", "", "path to bridge DLL for auto-loading (optional)")
	flag.StringVar(&cfg.DepsDir, "deps", "", "dir containing dependency jars (e.g., jna.jar) for auto-delivery")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable debug logging")
	flag.Parse()

	if cfg.AuthFile == "" || cfg.WebshellURL == "" {
		fmt.Fprintf(os.Stderr, "Usage: webshell-bridge --auth <listener.auth> --url <url> --token <token>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if cfg.PipelineName == "" {
		cfg.PipelineName = fmt.Sprintf("webshell_%s", cfg.ListenerName)
	}

	if cfg.Debug {
		logs.Log.SetLevel(logs.DebugLevel)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logs.Log.Important("shutting down...")
		cancel()
	}()

	bridge, err := NewBridge(cfg)
	if err != nil {
		logs.Log.Errorf("failed to create bridge: %v", err)
		os.Exit(1)
	}

	if err := bridge.Start(ctx); err != nil {
		logs.Log.Errorf("bridge exited with error: %v", err)
		os.Exit(1)
	}
}
