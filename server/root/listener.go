package root

import (
	"context"
	"fmt"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"time"
)

const rootRPCTimeout = 10 * time.Second

// ListenerCommand - Listener command
type ListenerCommand struct {
	Add   subCommand `command:"add" description:"Add a listener, e.g. 'listener add <name>'" subcommands-optional:"true" `
	Del   subCommand `command:"del" description:"Delete a listener, e.g. 'listener del <name>'" subcommands-optional:"true" `
	List  subCommand `command:"list" description:"List all listeners"`
	Reset subCommand `command:"reset" description:"Reset listener cert and regenerate auth file, e.g. 'listener reset <name>'" subcommands-optional:"true" `
}

func (ln *ListenerCommand) Name() string {
	return "listener"
}

func (ln *ListenerCommand) Execute(rpc clientrpc.RootRPCClient, msg *rootpb.Operator) (proto.Message, error) {
	if msg.Op == "add" {
		return saveListenerAuth(rpc, msg)
	} else if msg.Op == "del" {
		ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
		defer cancel()
		return rpc.RemoveListener(ctx, msg)
	} else if msg.Op == "list" {
		ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
		defer cancel()
		listeners, err := rpc.ListListeners(ctx, msg)
		if err != nil {
			return nil, err
		}
		for _, listener := range listeners.Listeners {
			logs.Log.Consolef("%s\t%s\n", listener.Id, listener.Ip)
		}
		return nil, nil
	} else if msg.Op == "reset" {
		ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
		defer cancel()
		_, _ = rpc.RemoveListener(ctx, msg)
		return saveListenerAuth(rpc, msg)
	}
	return nil, ErrInvalidOperator
}

func saveListenerAuth(rpc clientrpc.RootRPCClient, msg *rootpb.Operator) (proto.Message, error) {
	if len(msg.Args) == 0 {
		return nil, fmt.Errorf("missing name argument")
	}
	name, err := fileutils.SanitizeBasename(msg.Args[0])
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}
	authPath := filepath.Join(wd, fmt.Sprintf("%s.auth", name))
	configPath := filepath.Join(wd, fmt.Sprintf("%s.yaml", name))
	if msg.Op != "reset" {
		for _, target := range []string{authPath, configPath} {
			if _, statErr := os.Stat(target); statErr == nil {
				return nil, fmt.Errorf("listener file %q already exists", target)
			} else if !os.IsNotExist(statErr) {
				return nil, fmt.Errorf("inspect listener file %q: %w", target, statErr)
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
	defer cancel()
	resp, err := rpc.AddListener(ctx, msg)
	if err != nil {
		return nil, err
	}
	var conf *mtls.ClientConfig
	err = yaml.Unmarshal([]byte(resp.Response), &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal listener auth: %w", err)
	}
	if conf == nil || conf.Operator != name {
		operator := ""
		if conf != nil {
			operator = conf.Operator
		}
		return nil, fmt.Errorf("listener auth operator %q does not match requested name %q", operator, name)
	}

	type listenerConfig struct {
		Listeners struct {
			Enable    bool   `yaml:"enable"`
			Name      string `yaml:"name"`
			Auth      string `yaml:"auth"`
			Transport string `yaml:"transport"`
		} `yaml:"listeners"`
	}
	generatedConfig := listenerConfig{}
	generatedConfig.Listeners.Enable = true
	generatedConfig.Listeners.Name = name
	generatedConfig.Listeners.Auth = filepath.Base(authPath)
	generatedConfig.Listeners.Transport = configs.ListenerTransportReverse
	configData, err := yaml.Marshal(generatedConfig)
	if err != nil {
		return nil, fmt.Errorf("marshal listener config: %w", err)
	}

	wroteConfig := false
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		if err := fileutils.AtomicWriteFile(configPath, configData, 0o600); err != nil {
			return nil, fmt.Errorf("write listener config: %w", err)
		}
		wroteConfig = true
	} else if statErr != nil {
		return nil, fmt.Errorf("inspect listener config %q: %w", configPath, statErr)
	}

	err = fileutils.AtomicWriteFile(authPath, []byte(resp.Response), 0o600)
	if err != nil {
		if wroteConfig {
			_ = os.Remove(configPath)
		}
		return nil, err
	}
	logs.Log.Importantf("listener auth file written to %s", authPath)
	if wroteConfig {
		logs.Log.Importantf("listener config file written to %s", configPath)
	}
	return resp, nil
}
