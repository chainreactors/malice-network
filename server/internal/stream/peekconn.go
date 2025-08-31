package cryptostream

import (
	"github.com/chainreactors/malice-network/server/internal/parser"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
	"io"
	"net"
	"sync"
	"time"
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

func PeekSid(conn *Conn) (uint32, error) {
	data, err := conn.Peek(9)
	if err != nil {
		return 0, err
	}
	return malefic.ParseSid(data), nil
}

func WrapPeekConn(conn io.ReadWriteCloser, cryptos []Cryptor, parserName string) (*Conn, error) {
	bs := make([]byte, 9)
	_, err := io.ReadFull(conn, bs)
	if err != nil {
		return nil, err
	}

	var c Cryptor
	var p *parser.MessageParser
	for _, c = range cryptos {
		var de []byte
		de, err = Decrypt(c, bs)
		if err != nil {
			continue
		}
		if parserName == "auto" {
			p, err = parser.DetectProtocol(de)
		} else {
			p, err = parser.NewParser(parserName)
		}
		if err != nil {
			continue
		} else {
			err = nil
			bs = de
			break
		}
	}
	if err != nil {
		return nil, err
	}

	if _, ok := conn.(net.Conn); ok {
		conn = NewCryptoConn(conn.(net.Conn), c)
	} else {
		conn = NewCryptoRWC(conn, c)
	}

	return &Conn{
		ReadWriteCloser: conn,
		Parser:          p,
		buf:             bs,
	}, nil
}

type Conn struct {
	io.ReadWriteCloser
	buf    []byte
	Parser *parser.MessageParser
}

func (pc *Conn) Peek(n int) ([]byte, error) {
	// 如果已有足够数据，直接返回
	if len(pc.buf) >= n {
		return pc.buf[:n], nil
	}

	// 读取缺少的部分
	need := n - len(pc.buf)
	newBuf := make([]byte, need)
	_, err := io.ReadFull(pc.ReadWriteCloser, newBuf)
	if err != nil {
		return nil, err
	}
	pc.buf = append(pc.buf, newBuf...)
	return pc.buf[:n], nil
}

func (pc *Conn) Read(b []byte) (int, error) {
	var n int
	// 先消耗 buf 里的数据
	if len(pc.buf) > 0 {
		n = copy(b, pc.buf)
		pc.buf = pc.buf[n:]
		if n < len(b) {
			m, err := pc.ReadWriteCloser.Read(b[n:])
			return n + m, err
		}
		return n, nil
	}
	// 如果 buf 为空，直接从底层读取
	return pc.ReadWriteCloser.Read(b)
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

func (pc *Conn) SetDeadline(t time.Time) error {
	deadline, ok := pc.ReadWriteCloser.(interface {
		SetDeadline(time.Time) error
	})
	if ok {
		return deadline.SetDeadline(t)
	}
	return nil
}

func (pc *Conn) SetReadDeadline(t time.Time) error {
	readDeadline, ok := pc.ReadWriteCloser.(interface {
		SetReadDeadline(time.Time) error
	})
	if ok {
		return readDeadline.SetReadDeadline(t)
	}
	return nil
}

func (pc *Conn) SetWriteDeadline(t time.Time) error {
	writeDeadline, ok := pc.ReadWriteCloser.(interface {
		SetWriteDeadline(time.Time) error
	})
	if ok {
		return writeDeadline.SetWriteDeadline(t)
	}
	return nil
}
