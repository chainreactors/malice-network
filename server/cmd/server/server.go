package server

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/codenames"
	"github.com/chainreactors/malice-network/server/assets"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/yaml"
	"github.com/jessevdk/go-flags"
)

func init() {
	config.WithOptions(func(opt *config.Options) {
		opt.DecoderConfig.TagName = "config"
		opt.ParseDefault = true
	})
	config.AddDriver(yaml.Driver)
	codenames.SetupCodenames()
}

func isInteractiveTerminal() bool {
	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return stdinInfo.Mode()&os.ModeCharDevice != 0 && stdoutInfo.Mode()&os.ModeCharDevice != 0
}

func shouldOfferQuickstart(opt *Options, configMissing bool, interactive bool, hasActiveCommand bool) bool {
	if opt == nil {
		return false
	}
	if opt.Quickstart || !configMissing || !interactive || hasActiveCommand {
		return false
	}
	if opt.ServerOnly || opt.ListenerOnly || opt.Daemon {
		return false
	}
	return true
}

func promptQuickstart(configPath string) (bool, error) {
	if configPath == "" {
		configPath = configs.ServerConfigFileName
	}

	fmt.Printf("config %s not found. Run quickstart wizard now? [y/N] ", configPath)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func Start(defaultConfig []byte) error {
	var opt Options
	var err error
	parser := flags.NewParser(&opt, flags.Default)
	parser.SubcommandsOptional = true
	args, err := parser.Parse()
	if err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		return nil
	}

	configMissing := configs.FindConfig(opt.Config) == ""
	if opt.Quickstart {
		if err := RunQuickstart(&opt); err != nil {
			return fmt.Errorf("quickstart failed: %w", err)
		}
	} else if shouldOfferQuickstart(&opt, configMissing, isInteractiveTerminal(), parser.Active != nil) {
		runWizard, promptErr := promptQuickstart(opt.Config)
		if promptErr != nil {
			return fmt.Errorf("prompt quickstart: %w", promptErr)
		}
		if runWizard {
			if err := RunQuickstart(&opt); err != nil {
				return fmt.Errorf("quickstart failed: %w", err)
			}
		}
	}

	err = opt.PrepareConfig(defaultConfig)
	if err != nil {
		return err
	}

	if parser.Active != nil {
		err = opt.Execute(args, parser)
		if err != nil {
			logs.Log.Error(err)
		}
		return nil
	}

	serverReady := false
	if !opt.ListenerOnly && opt.Server.Enable {
		if err := assets.SetupGithubFile(); err != nil {
			logs.Log.Warnf("failed to setup github files: %s", err)
		}
		err = opt.PrepareServer()
		if err != nil {
			return fmt.Errorf("cannot prepare server, %s", err.Error())
		}
		serverReady = true
	}

	if !opt.ServerOnly && opt.Listeners.Enable {
		err = opt.PrepareListener()
		if err != nil {
			return fmt.Errorf("cannot prepare listener, %s", err.Error())
		}
	}
	if serverReady && opt.Listeners.Enable && opt.Listeners.IsForwardTransport() {
		if err := opt.PrepareForwardListenerClient(); err != nil {
			return fmt.Errorf("cannot prepare forward listener client, %s", err.Error())
		}
	}
	return opt.Handler()
}
