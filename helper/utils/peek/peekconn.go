package peek

import (
	"bufio"
	"net"
)

func WrapPeekConn(conn net.Conn) *Conn {
	return &Conn{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}
}

type Conn struct {
	net.Conn
	Reader *bufio.Reader
}

func (pc *Conn) Peek(n int) ([]byte, error) {
	return pc.Reader.Peek(n)
}

func (pc *Conn) Read(b []byte) (int, error) {
	return pc.Reader.Read(b)
}

// Other methods to satisfy the net.Conn interface
func (pc *Conn) Write(b []byte) (int, error) {
	return pc.Conn.Write(b)
}
