package common

import (
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"github.com/chainreactors/malice-network/utils/packet"
	"google.golang.org/protobuf/proto"
	"net"
)

var SID = "1234"

func NewClient(addr string) *Client {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	return &Client{
		conn: conn,
	}
}

type Client struct {
	conn net.Conn
}

func (c *Client) BuildSpite(spite *commonpb.Spite, msg proto.Message) {
	switch msg.(type) {
	case *commonpb.Register:
		spite.Body = &commonpb.Spite_Register{Register: msg.(*commonpb.Register)}
	case *pluginpb.ExecRequest:
		spite.Body = &commonpb.Spite_ExecRequest{ExecRequest: msg.(*pluginpb.ExecRequest)}
	}

}

// request spites
func (c *Client) Request(msg proto.Message) proto.Message {
	err := packet.WritePacket(c.conn, msg, SID)
	if err != nil {
		panic(err)
		return nil
	}
	_, msg, err = packet.ReadPacket(c.conn)
	if err != nil {
		panic(err)
		return nil
	}
	return msg
}

func (c *Client) RequestSpite(msg proto.Message) proto.Message {
	spites := &commonpb.Spites{
		Spites: []*commonpb.Spite{msg.(*commonpb.Spite)},
	}

	return c.Request(spites)
}
