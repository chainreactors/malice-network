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

// UserCommand - User command
type UserCommand struct {
	Add  subCommand `command:"add" description:"Add a user" subcommands-optional:"true" `
	Del  subCommand `command:"del" description:"Delete a user" subcommands-optional:"true" `
	List subCommand `command:"list" description:"List all users"`
}

func (user *UserCommand) Name() string {
	return "user"
}

func (user *UserCommand) Execute(rpc clientrpc.RootRPCClient, msg *rootpb.Operator) (proto.Message, error) {
	if msg.Op == "add" {
		resp, err := rpc.AddClient(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		configDir, _ := os.Getwd()
		var conf *mtls.ClientConfig
		err = yaml.Unmarshal([]byte(resp.Response), &conf)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal client auth: %w", err)
		}
		yamlPath := filepath.Join(configDir, fmt.Sprintf("%s_%s.auth", conf.Operator, conf.LHost))
		err = os.WriteFile(yamlPath, []byte(resp.Response), 0644)
		if err != nil {
			return nil, err
		}
		logs.Log.Importantf("client auth file written to %s", yamlPath)
		return resp, nil
	} else if msg.Op == "del" {
		return rpc.RemoveClient(context.Background(), msg)
	} else if msg.Op == "list" {
		clients, err := rpc.ListClients(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		for _, client := range clients.Clients {
			logs.Log.Console(client.Name + "\n")
		}

		return nil, nil
	}
	return nil, ErrInvalidOperator
}
