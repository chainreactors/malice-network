package root

import (
	"errors"
	"github.com/chainreactors/malice-network/helper/proto/client/rootpb"
	"github.com/chainreactors/malice-network/helper/proto/services/clientrpc"
	"google.golang.org/protobuf/proto"
)

var (
	ErrInvalidOperator = errors.New("not a valid operator")
)

type Command interface {
	Execute(rpc clientrpc.RootRPCClient, msg *rootpb.Operator) (proto.Message, error)
	Name() string
}

type subCommand struct {
}
