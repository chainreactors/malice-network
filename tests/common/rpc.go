package common

import (
	"github.com/chainreactors/malice-network/helper/types"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var (
	DefaultGRPCAddr     = "127.0.0.1:5004"
	DefaultListenerAddr = "127.0.0.1:5001"
)

func NewRPC(addr string) *MaliceRPC {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return &MaliceRPC{
		conn:     conn,
		Client:   clientrpc.NewMaliceRPCClient(conn),
		Implant:  listenerrpc.NewImplantRPCClient(conn),
		Listener: listenerrpc.NewListenerRPCClient(conn),
	}
}

type MaliceRPC struct {
	conn     *grpc.ClientConn
	Client   clientrpc.MaliceRPCClient
	Implant  listenerrpc.ImplantRPCClient
	Listener listenerrpc.ListenerRPCClient
}

func (c *MaliceRPC) BuildMessage(m types.NMessage) proto.Message {
	return m.Message()
}

func (c *MaliceRPC) Send() {
	c.conn.Close()
}
