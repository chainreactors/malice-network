package listener

import (
	"net"
)

type testAddr string

func (a testAddr) Network() string { return "tcp" }

func (a testAddr) String() string { return string(a) }

type testListener struct {
	accept func() (net.Conn, error)
	close  func() error
	addr   net.Addr
}

func (l testListener) Accept() (net.Conn, error) {
	return l.accept()
}

func (l testListener) Close() error {
	if l.close != nil {
		return l.close()
	}
	return nil
}

func (l testListener) Addr() net.Addr {
	if l.addr != nil {
		return l.addr
	}
	return testAddr("127.0.0.1:0")
}
