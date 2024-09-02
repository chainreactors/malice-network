package root

import (
	"context"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/mtls"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

// ListenerCommand - Listener command
type ListenerCommand struct {
	Add  subCommand `command:"add" description:"Add a listener" subcommands-optional:"true" `
	Del  subCommand `command:"del" description:"Delete a listener" subcommands-optional:"true" `
	List subCommand `command:"list" description:"List all listeners"`
}

func (ln *ListenerCommand) Name() string {
	return "listener"
}

func (ln *ListenerCommand) Execute(rpc clientrpc.RootRPCClient, msg *rootpb.Operator) (proto.Message, error) {
	// init operator
	if msg.Op == "add" {
		resp, err := rpc.AddListener(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		configDir, _ := os.Getwd()
		var conf *mtls.ClientConfig
		err = yaml.Unmarshal([]byte(resp.Response), &conf)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal listener auth: %w", err)
		}
		yamlPath := filepath.Join(configDir, fmt.Sprintf("%s.auth", msg.Args[0]))
		err = os.WriteFile(yamlPath, []byte(resp.Response), 0644)
		if err != nil {
			return nil, err
		}
		logs.Log.Importantf("listener auth file written to %s", yamlPath)
		return resp, nil
	} else if msg.Op == "del" {
		return rpc.RemoveListener(context.Background(), msg)
	} else if msg.Op == "list" {
		return rpc.ListListeners(context.Background(), msg)
	}
	return nil, ErrInvalidOperator
}
