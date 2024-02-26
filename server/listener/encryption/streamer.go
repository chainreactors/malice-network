package encryption

import (
	"io"
	"net"
	"sync"
)

var (
	bufPool16k sync.Pool
	bufPool5k  sync.Pool
	bufPool2k  sync.Pool
	bufPool1k  sync.Pool
	bufPool    sync.Pool
)

func GetBuf(size int) []byte {
	var x interface{}
	if size >= 16*1024 {
		x = bufPool16k.Get()
	} else if size >= 5*1024 {
		x = bufPool5k.Get()
	} else if size >= 2*1024 {
		x = bufPool2k.Get()
	} else if size >= 1*1024 {
		x = bufPool1k.Get()
	} else {
		x = bufPool.Get()
	}
	if x == nil {
		return make([]byte, size)
	}
	buf := x.([]byte)
	if cap(buf) < size {
		return make([]byte, size)
	}
	return buf[:size]
}

func PutBuf(buf []byte) {
	size := cap(buf)
	if size >= 16*1024 {
		bufPool16k.Put(buf)
	} else if size >= 5*1024 {
		bufPool5k.Put(buf)
	} else if size >= 2*1024 {
		bufPool2k.Put(buf)
	} else if size >= 1*1024 {
		bufPool1k.Put(buf)
	} else {
		bufPool.Put(buf)
	}
}

func WrapWithEncryption(conn net.Conn, key []byte) (net.Conn, error) {
	w, err := NewAESWriter(conn, key)
	if err != nil {
		return nil, err
	}
	return WrapConn(conn, NewAESReader(conn, key), w, func() error {
		return conn.Close()
	}), nil
}

// closeFn will be called only once
func WrapConn(conn net.Conn, r io.Reader, w io.Writer, closeFn func() error) net.Conn {
	return &WrappedConn{
		r:       r,
		w:       w,
		closeFn: closeFn,
		closed:  false,
		Conn:    conn,
	}
}

// Join two io.ReadWriteCloser and do some operations.
func Join(c1 io.ReadWriteCloser, c2 io.ReadWriteCloser) (inCount int64, outCount int64, errors []error) {
	var wait sync.WaitGroup
	recordErrs := make([]error, 2)
	pipe := func(number int, to io.ReadWriteCloser, from io.ReadWriteCloser, count *int64) {
		defer wait.Done()
		defer to.Close()
		defer from.Close()

		buf := GetBuf(16 * 1024)
		defer PutBuf(buf)
		*count, recordErrs[number] = io.CopyBuffer(to, from, buf)
	}

	wait.Add(2)
	go pipe(0, c1, c2, &inCount)
	go pipe(1, c2, c1, &outCount)
	wait.Wait()

	for _, e := range recordErrs {
		if e != nil {
			errors = append(errors, e)
		}
	}
	return
}

type WrappedConn struct {
	r       io.Reader
	w       io.Writer
	closeFn func() error

	closed bool
	mu     sync.Mutex
	net.Conn
}

func (rwc *WrappedConn) Read(p []byte) (n int, err error) {
	return rwc.r.Read(p)
}

func (rwc *WrappedConn) Write(p []byte) (n int, err error) {
	return rwc.w.Write(p)
}

func (rwc *WrappedConn) Close() error {
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
