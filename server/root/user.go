package root

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/protobuf/proto"
)

// UserCommand - User command
type UserCommand struct {
	Add  addCommand  `command:"add" description:"Add a user" subcommands-optional:"true" `
	Del  delCommand  `command:"del" description:"Delete a user" subcommands-optional:"true" `
	List listCommand `command:"list" description:"List all users"`
}

func (user *UserCommand) Name() string {
	return "user"
}

func (user *UserCommand) Execute(rpc clientrpc.RootRPCClient) (proto.Message, error) {
	// init operator
	if user.Add.Name != "" {
		msg := &rootpb.Operator{
			Name:      user.Name(),
			Operation: "add",
			Arg:       user.Add.Name,
		}
		resp, err := rpc.AddClient(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	if user.Del.Name != "" {
		msg := &rootpb.Operator{
			Name:      user.Name(),
			Operation: "del",
			Arg:       user.Del.Name,
		}
		resp, err := rpc.RemoveClient(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	if user.List.Called {
		msg := &rootpb.Operator{
			Name:      user.Name(),
			Operation: "list",
		}
		resp, err := rpc.ListClients(context.Background(), msg)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	return nil, ErrInvalidOperator
}
