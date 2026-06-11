package root

import (
	"context"
	"fmt"
	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
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
	ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
	defer cancel()
	resp, err := rpc.AddListener(ctx, msg)
	if err != nil {
		return nil, err
	}
	wd, _ := os.Getwd()
	var conf *mtls.ClientConfig
	err = yaml.Unmarshal([]byte(resp.Response), &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal listener auth: %w", err)
	}
	yamlPath := filepath.Join(wd, fmt.Sprintf("%s.auth", name))
	err = fileutils.AtomicWriteFile(yamlPath, []byte(resp.Response), 0o600)
	if err != nil {
		return nil, err
	}
	logs.Log.Importantf("listener auth file written to %s", yamlPath)
	return resp, nil
}
