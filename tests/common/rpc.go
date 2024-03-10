package common

import (
	"context"
	"errors"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
	"github.com/chainreactors/malice-network/proto/client/clientpb"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"

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
	options := RpcOptions()
	conn, err := grpc.Dial(addr, options...)
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
	Root     clientrpc.RootRPCClient
}

func (c *Client) Send() {
	c.conn.Close()
}

func (c *Client) Call(rpcname string, msg proto.Message) (proto.Message, error) {
	meta := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("session_id", hash.Md5Hash(c.sid)))
	var resp proto.Message
	var err error
	switch rpcname {
	case consts.ModuleExecution:
		resp, err = c.Client.Execute(meta, msg.(*implantpb.ExecRequest))
	case consts.ModuleUpload:
		resp, err = c.Client.Upload(meta, msg.(*implantpb.UploadRequest))
	case consts.ModuleDownload:
		resp, err = c.Client.Download(meta, msg.(*implantpb.DownloadRequest))
	case consts.ModulePwd:
		resp, err = c.Client.Pwd(meta, msg.(*implantpb.Empty))

	case consts.CommandBroadcast:
		resp, err = c.Client.Broadcast(meta, msg.(*clientpb.Event))
	default:
		return nil, errors.New("unknown rpc")
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) WaitResponse(task *clientpb.Task) (*implantpb.Spite, error) {
	meta := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("session_id", hash.Md5Hash(c.sid)))
	resp, err := c.Client.WaitTaskContent(meta, task)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
