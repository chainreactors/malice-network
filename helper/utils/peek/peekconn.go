package peek

import (
	"bufio"
	"io"
	"net"
	"sync"
)

type ReadWriteCloser struct {
	r       io.Reader
	w       io.Writer
	closeFn func() error

	closed bool
	mu     sync.Mutex
}

func WrapReadWriteCloser(r io.Reader, w io.Writer, closeFn func() error) io.ReadWriteCloser {
	return &ReadWriteCloser{
		r:       r,
		w:       w,
		closeFn: closeFn,
		closed:  false,
	}
}

func (rwc *ReadWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.r.Read(p)
}

func (rwc *ReadWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.w.Write(p)
}

func (rwc *ReadWriteCloser) Close() error {
	rwc.mu.Lock()
	if rwc.closed {
		rwc.mu.Unlock()
		return nil
	}
	rwc.closed = true
	rwc.mu.Unlock()
	if rwc.closeFn != nil {
		return rwc.closeFn()
	}
	return nil
}

func WrapPeekConn(conn io.ReadWriteCloser) *Conn {
	return &Conn{
		ReadWriteCloser: conn,
		Reader:          bufio.NewReader(conn),
	}
}

type Conn struct {
	io.ReadWriteCloser
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
	return pc.ReadWriteCloser.Write(b)
}

func (pc *Conn) RemoteAddr() net.Addr {
	remote, ok := pc.ReadWriteCloser.(interface {
		RemoteAddr() net.Addr
	})
	if ok {
		return remote.RemoteAddr()
	}
	return nil
}

func (pc *Conn) LocalAddr() net.Addr {
	local, ok := pc.ReadWriteCloser.(interface {
		LocalAddr() net.Addr
	})
	if ok {
		return local.LocalAddr()
	}
	return nil
}
