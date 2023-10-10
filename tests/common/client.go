package common

import (
	"github.com/chainreactors/malice-network/utils/packet"
	"google.golang.org/protobuf/proto"
	"net"
)

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

func (c *Client) Request(spite proto.Message) proto.Message {
	err := packet.WriteMessage(c.conn, spite)
	if err != nil {
		panic(err)
		return nil
	}
	spite, err = packet.ReadMessage(c.conn)
	if err != nil {
		panic(err)
		return nil
	}
	return spite
}
