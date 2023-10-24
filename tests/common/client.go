package common

import (
	"github.com/chainreactors/malice-network/helper/packet"
	"github.com/chainreactors/malice-network/proto/implant/commonpb"
	"github.com/chainreactors/malice-network/proto/implant/pluginpb"
	"google.golang.org/protobuf/proto"
	"net"
)

func NewClient(addr string, sid string) *Client {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		panic(err)
	}
	return &Client{
		conn: conn,
		Sid:  sid,
	}
}

type Client struct {
	conn net.Conn
	Sid  string
}

func (c *Client) Register() {
	spite := &commonpb.Spite{
		TaskId: 1,
	}
	body := &commonpb.Register{
		Os: &commonpb.Os{
			Name: "windows",
		},
		Process: &commonpb.Process{
			Name: "test",
			Pid:  123,
			Uid:  "admin",
			Gid:  "root",
		},
		Timer: &commonpb.Timer{
			Interval: 10,
		},
	}
	c.BuildSpite(spite, body)
	c.WriteSpite(spite)
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
func (c *Client) Write(msg proto.Message) error {
	err := packet.WritePacket(c.conn, msg, c.Sid)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) WriteSpite(msg proto.Message) error {
	spites := &commonpb.Spites{
		Spites: []*commonpb.Spite{msg.(*commonpb.Spite)},
	}

	return c.Write(spites)
}

func (c *Client) Read() (proto.Message, error) {
	_, msg, err := packet.ReadPacket(c.conn)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
