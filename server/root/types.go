package root

import (
	"errors"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"google.golang.org/protobuf/proto"
)

var (
	ErrInvalidOperator = errors.New("not a valid operator")
)

type Command interface {
	Execute(rpc clientrpc.RootRPCClient) (proto.Message, error)
	Name() string
}

type addCommand struct {
	Name string `long:"name" short:"n" description:"Name of the listener/user"`
}

type delCommand struct {
	Name string `long:"name" short:"n" description:"Name of the listener/user"`
}

type listCommand struct {
	Called bool `long:"called" short:"c" description:"List called listeners/users"`
}
