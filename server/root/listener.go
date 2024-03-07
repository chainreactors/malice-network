package root

import (
	"github.com/chainreactors/malice-network/proto/client/rootpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/protobuf/proto"
)

// ListenerCommand - Listener command
type ListenerCommand struct {
	Add  addCommand  `command:"add" description:"Add a listener" subcommands-optional:"true" `
	Del  delCommand  `command:"del" description:"Delete a listener" subcommands-optional:"true" `
	List listCommand `command:"list" description:"List all listeners"`
}

func (ln *ListenerCommand) Name() string {
	return "listener"
}

func (ln *ListenerCommand) Execute(rpc clientrpc.RootRPCClient) (proto.Message, error) {
	// init operator
	if ln.Add.Name != "" {
		msg := &rootpb.Operator{
			Name:      ln.Name(),
			Operation: "add",
			Arg:       ln.Add.Name,
		}
		resp, err := rpc.AddListener(nil, msg)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	if ln.Del.Name != "" {
		msg := &rootpb.Operator{
			Name:      ln.Name(),
			Operation: "del",
			Arg:       ln.Del.Name,
		}
		resp, err := rpc.RemoveListener(nil, msg)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}

	if ln.List.Called {
		msg := &rootpb.Operator{
			Name:      ln.Name(),
			Operation: "list",
		}
		resp, err := rpc.ListListeners(nil, msg)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
	return nil, ErrInvalidOperator
}
