package common

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/proto/services/clientrpc"
	"github.com/chainreactors/malice-network/proto/services/listenerrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

var (
	DefaultGRPCAddr     = "127.0.0.1:5004"
	DefaultListenerAddr = "127.0.0.1:5001"
	TestSid             = []byte{1, 2, 3, 4}
)

func NewClient(addr string, sid []byte) *Client {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	return &Client{
		conn:     conn,
		sid:      sid,
		Client:   clientrpc.NewMaliceRPCClient(conn),
		Implant:  listenerrpc.NewImplantRPCClient(conn),
		Listener: listenerrpc.NewListenerRPCClient(conn),
	}
}

type Client struct {
	conn     *grpc.ClientConn
	sid      []byte
	Client   clientrpc.MaliceRPCClient
	Implant  listenerrpc.ImplantRPCClient
	Listener listenerrpc.ListenerRPCClient
}

func (c *Client) Send() {
	c.conn.Close()
}

func (c *Client) Call(rpcname string, msg proto.Message) (proto.Message, error) {
	switch rpcname {
	case consts.ExecutionStr:
		resp, err := c.Client.Execute(metadata.NewOutgoingContext(context.Background(), metadata.Pairs(
			"session_id", hash.Md5Hash(c.sid))), msg.(*pluginpb.ExecRequest))
		if err != nil {
			return nil, err
		}
		return resp, nil
	default:
		return nil, errors.New("unknown rpc")
	}
}
