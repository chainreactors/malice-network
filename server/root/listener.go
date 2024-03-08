package root

import (
	"context"
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/protobuf/proto"
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
		return rpc.AddListener(context.Background(), msg)
	} else if msg.Op == "del" {
		return rpc.RemoveListener(context.Background(), msg)
	} else if msg.Op == "list" {
		return rpc.ListListeners(context.Background(), msg)
	}
	return nil, ErrInvalidOperator
}
