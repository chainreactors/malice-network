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
)

// UserCommand - User command
type UserCommand struct {
	Add   subCommand `command:"add" description:"Add a client user, e.g. 'user add <name>'" subcommands-optional:"true" `
	Del   subCommand `command:"del" description:"Delete a client user, e.g. 'user del <name>'" subcommands-optional:"true" `
	List  subCommand `command:"list" description:"List all client users"`
	Reset subCommand `command:"reset" description:"Reset user cert and regenerate auth file, e.g. 'user reset <name>'" subcommands-optional:"true" `
}

func (user *UserCommand) Name() string {
	return "user"
}

func (user *UserCommand) Execute(rpc clientrpc.RootRPCClient, msg *rootpb.Operator) (proto.Message, error) {
	if msg.Op == "add" {
		return saveClientAuth(rpc, msg)
	} else if msg.Op == "del" {
		ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
		defer cancel()
		return rpc.RemoveClient(ctx, msg)
	} else if msg.Op == "list" {
		ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
		defer cancel()
		clients, err := rpc.ListClients(ctx, msg)
		if err != nil {
			return nil, err
		}
		for _, client := range clients.Clients {
			logs.Log.Console(client.Name + "\n")
		}
		return nil, nil
	} else if msg.Op == "reset" {
		// Remove existing operator (ignore error if not found)
		ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
		defer cancel()
		_, _ = rpc.RemoveClient(ctx, msg)
		return saveClientAuth(rpc, msg)
	}
	return nil, ErrInvalidOperator
}

func saveClientAuth(rpc clientrpc.RootRPCClient, msg *rootpb.Operator) (proto.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), rootRPCTimeout)
	defer cancel()
	resp, err := rpc.AddClient(ctx, msg)
	if err != nil {
		return nil, err
	}
	configDir, _ := os.Getwd()
	var conf *mtls.ClientConfig
	err = yaml.Unmarshal([]byte(resp.Response), &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal client auth: %w", err)
	}
	yamlPath := filepath.Join(configDir, fmt.Sprintf("%s_%s.auth", conf.Operator, conf.Host))
	err = fileutils.AtomicWriteFile(yamlPath, []byte(resp.Response), 0o600)
	if err != nil {
		return nil, err
	}
	logs.Log.Importantf("client auth file written to %s", yamlPath)
	return resp, nil
}
