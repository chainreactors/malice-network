package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/logs"
	"github.com/gookit/config/v2"
)

func main() {
	cfg := &Config{}
	flag.StringVar(&cfg.AuthFile, "auth", "", "path to listener.auth mTLS certificate file")
	flag.StringVar(&cfg.ServerAddr, "server", "", "server address (overrides auth file)")
	flag.StringVar(&cfg.ListenerName, "listener", "webshell-listener", "listener name")
	flag.StringVar(&cfg.ListenerIP, "ip", "127.0.0.1", "listener external IP")
	flag.StringVar(&cfg.PipelineName, "pipeline", "", "pipeline name (auto-generated if empty)")
	flag.StringVar(&cfg.Suo5URL, "suo5", "", "suo5 webshell URL (e.g. suo5://target/suo5.jsp)")
	flag.StringVar(&cfg.DLLAddr, "dll-addr", "127.0.0.1:13338", "target-side malefic bind DLL address")
	flag.BoolVar(&cfg.Debug, "debug", false, "enable debug logging")
	flag.Parse()

	if cfg.AuthFile == "" || cfg.Suo5URL == "" {
		fmt.Fprintf(os.Stderr, "Usage: webshell-bridge --auth <listener.auth> --suo5 <url>\n")
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

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logs.Log.Important("shutting down...")
		cancel()
	}()

	// Initialize packet length config for the malefic parser.
	config.Set(consts.ConfigMaxPacketLength, 10*1024*1024)

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
