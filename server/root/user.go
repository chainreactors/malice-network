package root

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/protobuf/proto"
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
		return rpc.AddClient(context.Background(), msg)
	} else if msg.Op == "del" {
		return rpc.RemoveClient(context.Background(), msg)
	} else if msg.Op == "list" {
		return rpc.ListClients(context.Background(), msg)
	}
	return nil, ErrInvalidOperator
}
