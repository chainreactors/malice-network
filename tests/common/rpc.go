package common

import (
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"github.com/chainreactors/malice-network/utils/constant"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var (
	DefaultGRPCAddr     = "127.0.0.1:51004"
	DefaultListenerAddr = "127.0.0.1:51001"
)

func NewRPC(addr string) *MaliceRPC {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := clientrpc.NewMaliceRPCClient(conn)
	implant := listenerrpc.NewImplantRPCClient(conn)
	return &MaliceRPC{
		conn:    conn,
		Client:  client,
		Implant: implant,
	}
}

type MaliceRPC struct {
	conn    *grpc.ClientConn
	Client  clientrpc.MaliceRPCClient
	Implant listenerrpc.ImplantRPCClient
}

func (c *MaliceRPC) BuildMessage(m constant.NMessage) proto.Message {
	return m.Message()
}

func (c *MaliceRPC) Send() {
	c.conn.Close()
}
