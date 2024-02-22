package encryption

import (
	"io"
	"sync"
)

func WrapWithEncryption(rwc io.ReadWriteCloser, key []byte) (io.ReadWriteCloser, error) {
	w, err := NewAESWriter(rwc, key)
	if err != nil {
		return nil, err
	}
	return WrapReadWriteCloser(NewAESReader(rwc, key), w, func() error {
		return rwc.Close()
	}), nil
}

// closeFn will be called only once
func WrapReadWriteCloser(r io.Reader, w io.Writer, closeFn func() error) io.ReadWriteCloser {
	return &ReadWriteCloser{
		r:       r,
		w:       w,
		closeFn: closeFn,
		closed:  false,
	}
}

type ReadWriteCloser struct {
	r       io.Reader
	w       io.Writer
	closeFn func() error

	closed bool
	mu     sync.Mutex
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
