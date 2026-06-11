package cryptostream

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// testRWC wraps separate reader/writer with a custom close function for testing.
type testRWC struct {
	r       io.Reader
	w       io.Writer
	closeFn func() error
	closed  bool
	mu      sync.Mutex
}

func (t *testRWC) Read(p []byte) (int, error) {
	t.mu.Lock()
	closed := t.closed
	t.mu.Unlock()
	if closed {
		return 0, errors.New("closed")
	}
	return t.r.Read(p)
}

func (t *testRWC) Write(p []byte) (int, error) {
	t.mu.Lock()
	closed := t.closed
	t.mu.Unlock()
	if closed {
		return 0, errors.New("closed")
	}
	return t.w.Write(p)
}

func (t *testRWC) Close() error {
	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()
	if t.closeFn != nil {
		return t.closeFn()
	}
	return nil
}

// TestWrapReadWriteCloser_CloseCallsCustomFn verifies the custom close function is invoked.
func TestWrapReadWriteCloser_CloseCallsCustomFn(t *testing.T) {
	t.Parallel()
	called := false
	closeFn := func() error {
		called = true
		return nil
	}

	rwc := WrapReadWriteCloser(
		bytes.NewReader([]byte("data")),
		&bytes.Buffer{},
		closeFn,
	)

	if err := rwc.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if !called {
		t.Error("custom close function was not called")
	}
}

// TestWrapReadWriteCloser_CloseIdempotent verifies Close can be called multiple
// times without panicking and only invokes closeFn once.
func TestWrapReadWriteCloser_CloseIdempotent(t *testing.T) {
	t.Parallel()
	callCount := 0
	closeFn := func() error {
		callCount++
		return nil
	}

	rwc := WrapReadWriteCloser(
		bytes.NewReader(nil),
		&bytes.Buffer{},
		closeFn,
	)

	_ = rwc.Close()
	_ = rwc.Close()
	_ = rwc.Close()

	if callCount != 1 {
		t.Errorf("closeFn called %d times, expected 1", callCount)
	}
}

// TestWrapReadWriteCloser_CloseReturnsError verifies close error is propagated.
func TestWrapReadWriteCloser_CloseReturnsError(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("close failed")
	rwc := WrapReadWriteCloser(
		bytes.NewReader(nil),
		&bytes.Buffer{},
		func() error { return expectedErr },
	)

	err := rwc.Close()
	if !errors.Is(err, expectedErr) {
		t.Errorf("Close error = %v, want %v", err, expectedErr)
	}
}

// TestWrapReadWriteCloser_NilCloseFn verifies Close works when closeFn is nil.
func TestWrapReadWriteCloser_NilCloseFn(t *testing.T) {
	t.Parallel()
	rwc := WrapReadWriteCloser(
		bytes.NewReader(nil),
		&bytes.Buffer{},
		nil,
	)

	if err := rwc.Close(); err != nil {
		t.Errorf("Close with nil closeFn returned error: %v", err)
	}
}

// TestWrapReadWriteCloser_ReadWrite verifies read and write pass through correctly.
func TestWrapReadWriteCloser_ReadWrite(t *testing.T) {
	t.Parallel()
	input := []byte("hello world")
	reader := bytes.NewReader(input)
	writer := &bytes.Buffer{}

	rwc := WrapReadWriteCloser(reader, writer, nil)

	// Read
	buf := make([]byte, 20)
	n, err := rwc.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !bytes.Equal(buf[:n], input) {
		t.Errorf("Read got %q, want %q", buf[:n], input)
	}

	// Write
	toWrite := []byte("output data")
	n, err = rwc.Write(toWrite)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(toWrite) {
		t.Errorf("Write returned %d, want %d", n, len(toWrite))
	}
	if !bytes.Equal(writer.Bytes(), toWrite) {
		t.Errorf("writer contains %q, want %q", writer.Bytes(), toWrite)
	}
}

// TestConn_Peek_ReturnsBytes verifies Peek returns the first N bytes without consuming them.
func TestConn_Peek_ReturnsBytes(t *testing.T) {
	t.Parallel()
	data := []byte("hello peek test data")
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	go func() {
		clientConn.Write(data)
	}()

	// Use RAW cryptor so data passes through unchanged
	enc := NewXorEncryptor([]byte{0}, []byte{0})
	cc := NewCryptoConn(serverConn, enc)

	pc := &Conn{
		ReadWriteCloser: cc,
		buf:             nil,
	}

	peeked, err := pc.Peek(5)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if !bytes.Equal(peeked, data[:5]) {
		t.Errorf("Peek got %q, want %q", peeked, data[:5])
	}
}

// TestConn_Peek_MultiplePeeks verifies peeking twice returns the same data (not consumed).
func TestConn_Peek_MultiplePeeks(t *testing.T) {
	t.Parallel()
	data := []byte("abcdefghijklmnop")
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	go func() {
		clientConn.Write(data)
	}()

	enc := NewXorEncryptor([]byte{0}, []byte{0})
	cc := NewCryptoConn(serverConn, enc)

	pc := &Conn{
		ReadWriteCloser: cc,
		buf:             nil,
	}

	peek1, err := pc.Peek(5)
	if err != nil {
		t.Fatalf("first Peek failed: %v", err)
	}

	peek2, err := pc.Peek(5)
	if err != nil {
		t.Fatalf("second Peek failed: %v", err)
	}

	if !bytes.Equal(peek1, peek2) {
		t.Errorf("multiple peeks returned different data: %q vs %q", peek1, peek2)
	}
}

// TestConn_Peek_GrowingPeek verifies peeking with increasing N accumulates data.
func TestConn_Peek_GrowingPeek(t *testing.T) {
	t.Parallel()
	data := []byte("abcdefghij")
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	go func() {
		clientConn.Write(data)
	}()

	enc := NewXorEncryptor([]byte{0}, []byte{0})
	cc := NewCryptoConn(serverConn, enc)

	pc := &Conn{
		ReadWriteCloser: cc,
		buf:             nil,
	}

	// First peek 3 bytes
	peek3, err := pc.Peek(3)
	if err != nil {
		t.Fatalf("Peek(3) failed: %v", err)
	}
	if string(peek3) != "abc" {
		t.Errorf("Peek(3) = %q, want %q", peek3, "abc")
	}

	// Then peek 7 bytes - should include the first 3 plus 4 more
	peek7, err := pc.Peek(7)
	if err != nil {
		t.Fatalf("Peek(7) failed: %v", err)
	}
	if string(peek7) != "abcdefg" {
		t.Errorf("Peek(7) = %q, want %q", peek7, "abcdefg")
	}

	// Verify the buf has accumulated
	if len(pc.buf) < 7 {
		t.Errorf("internal buf should have at least 7 bytes, got %d", len(pc.buf))
	}
}

// TestConn_Read_ConsumesBufferFirst verifies that after Peek, Read gets
// buffered data before reading from underlying connection.
func TestConn_Read_ConsumesBufferFirst(t *testing.T) {
	t.Parallel()
	data := []byte("peeked-then-read-data")
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	go func() {
		clientConn.Write(data)
	}()

	enc := NewXorEncryptor([]byte{0}, []byte{0})
	cc := NewCryptoConn(serverConn, enc)

	pc := &Conn{
		ReadWriteCloser: cc,
		buf:             nil,
	}

	// Peek first 6 bytes
	_, err := pc.Peek(6)
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}

	// Now Read should get the peeked data first
	buf := make([]byte, 100)
	n, err := pc.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// The read should include the peeked bytes plus whatever else was available
	result := buf[:n]
	if !bytes.HasPrefix(data, result[:6]) {
		t.Errorf("Read should start with peeked data, got %q", result[:6])
	}
}

// TestConn_Read_AfterBufferDrained verifies that once the peek buffer is
// consumed, subsequent reads go directly to the underlying reader.
func TestConn_Read_AfterBufferDrained(t *testing.T) {
	t.Parallel()
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	enc := NewXorEncryptor([]byte{0}, []byte{0})
	cc := NewCryptoConn(serverConn, enc)

	pc := &Conn{
		ReadWriteCloser: cc,
		buf:             []byte("buffered"),
	}

	// Read the buffered data
	buf := make([]byte, 8)
	n, err := pc.Read(buf)
	if err != nil {
		t.Fatalf("Read buffered failed: %v", err)
	}
	if string(buf[:n]) != "buffered" {
		t.Errorf("got %q, want %q", buf[:n], "buffered")
	}

	// Buffer should be drained
	if len(pc.buf) != 0 {
		t.Errorf("buf should be empty, got %d bytes", len(pc.buf))
	}

	// Next read should come from underlying
	go func() {
		clientConn.Write([]byte("from-pipe"))
	}()

	buf2 := make([]byte, 100)
	n, err = pc.Read(buf2)
	if err != nil {
		t.Fatalf("Read from underlying failed: %v", err)
	}
	if string(buf2[:n]) != "from-pipe" {
		t.Errorf("got %q from underlying, want %q", buf2[:n], "from-pipe")
	}
}

// TestConn_Read_BufferPartialConsume verifies that if the caller's buffer is
// smaller than the peek buffer, only the requested amount is returned and
// the remainder stays in buf. Then the NEXT read also tries the underlying
// (because Conn.Read drains buf AND reads underlying in the same call when
// the caller's buffer is larger than remaining buf).
func TestConn_Read_BufferPartialConsume(t *testing.T) {
	t.Parallel()
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	enc := NewXorEncryptor([]byte{0}, []byte{0})
	cc := NewCryptoConn(serverConn, enc)

	pc := &Conn{
		ReadWriteCloser: cc,
		buf:             []byte("ABCDEFGHIJ"), // 10 bytes buffered
	}

	// Read only 4 bytes
	buf := make([]byte, 4)
	n, err := pc.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != 4 {
		t.Errorf("Read returned %d bytes, want 4", n)
	}
	if string(buf[:n]) != "ABCD" {
		t.Errorf("got %q, want %q", buf[:n], "ABCD")
	}

	// Remaining 6 bytes should still be in buf
	if string(pc.buf) != "EFGHIJ" {
		t.Errorf("remaining buf = %q, want %q", pc.buf, "EFGHIJ")
	}
}

// TestConn_Read_BufferAndUnderlying verifies that when caller's buffer is larger
// than the peek buffer, Conn.Read reads from buf AND underlying in one call.
// This is the code path: copy from buf, then read from underlying to fill rest.
func TestConn_Read_BufferAndUnderlying(t *testing.T) {
	t.Parallel()
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	enc := NewXorEncryptor([]byte{0}, []byte{0})
	cc := NewCryptoConn(serverConn, enc)

	pc := &Conn{
		ReadWriteCloser: cc,
		buf:             []byte("BUF"),
	}

	go func() {
		clientConn.Write([]byte("UNDERLYING"))
	}()

	// Request 20 bytes - 3 from buf + rest from underlying
	buf := make([]byte, 20)
	n, err := pc.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	result := string(buf[:n])
	// Should start with "BUF" then have data from underlying
	if len(result) < 3 || result[:3] != "BUF" {
		t.Errorf("result should start with BUF, got %q", result)
	}
}

// TestConn_RemoteAddr_NilUnderlying verifies RemoteAddr returns nil when
// the underlying ReadWriteCloser does not implement RemoteAddr.
func TestConn_RemoteAddr_NilUnderlying(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	rwc := &testRWC{r: buf, w: buf}

	pc := &Conn{
		ReadWriteCloser: rwc,
	}

	addr := pc.RemoteAddr()
	if addr != nil {
		t.Errorf("expected nil RemoteAddr, got %v", addr)
	}
}

// TestConn_LocalAddr_NilUnderlying verifies LocalAddr returns nil when
// the underlying ReadWriteCloser does not implement LocalAddr.
func TestConn_LocalAddr_NilUnderlying(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	rwc := &testRWC{r: buf, w: buf}

	pc := &Conn{
		ReadWriteCloser: rwc,
	}

	addr := pc.LocalAddr()
	if addr != nil {
		t.Errorf("expected nil LocalAddr, got %v", addr)
	}
}

// TestConn_SetDeadline_NoOp verifies SetDeadline returns nil for non-Conn underlying.
func TestConn_SetDeadline_NoOp(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	rwc := &testRWC{r: buf, w: buf}

	pc := &Conn{
		ReadWriteCloser: rwc,
	}

	if err := pc.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("SetDeadline returned error: %v", err)
	}
	if err := pc.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("SetReadDeadline returned error: %v", err)
	}
	if err := pc.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Errorf("SetWriteDeadline returned error: %v", err)
	}
}

// TestConn_Write_PassesThrough verifies Conn.Write delegates to underlying.
func TestConn_Write_PassesThrough(t *testing.T) {
	t.Parallel()
	writer := &bytes.Buffer{}
	rwc := &testRWC{r: bytes.NewReader(nil), w: writer}

	pc := &Conn{
		ReadWriteCloser: rwc,
	}

	data := []byte("write through")
	n, err := pc.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write returned %d, want %d", n, len(data))
	}
	if !bytes.Equal(writer.Bytes(), data) {
		t.Errorf("underlying writer got %q, want %q", writer.Bytes(), data)
	}
}

// TestConn_Peek_ZeroBytes verifies Peek(0) returns empty slice without error.
func TestConn_Peek_ZeroBytes(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	rwc := &testRWC{r: buf, w: buf}

	pc := &Conn{
		ReadWriteCloser: rwc,
		buf:             nil,
	}

	peeked, err := pc.Peek(0)
	if err != nil {
		t.Fatalf("Peek(0) failed: %v", err)
	}
	if len(peeked) != 0 {
		t.Errorf("Peek(0) returned %d bytes, want 0", len(peeked))
	}
}

// TestConn_Peek_WithPreExistingBuf verifies Peek works correctly when the
// Conn already has data in its buf (e.g., from WrapPeekConn initialization).
func TestConn_Peek_WithPreExistingBuf(t *testing.T) {
	t.Parallel()
	pc := &Conn{
		ReadWriteCloser: &testRWC{r: bytes.NewReader([]byte("extra")), w: &bytes.Buffer{}},
		buf:             []byte("pre"),
	}

	// Peek 3 bytes - should come entirely from existing buf
	peeked, err := pc.Peek(3)
	if err != nil {
		t.Fatalf("Peek(3) failed: %v", err)
	}
	if string(peeked) != "pre" {
		t.Errorf("Peek(3) = %q, want %q", peeked, "pre")
	}

	// Peek 5 bytes - needs 2 more from underlying
	peeked, err = pc.Peek(5)
	if err != nil {
		t.Fatalf("Peek(5) failed: %v", err)
	}
	if string(peeked) != "preex" {
		t.Errorf("Peek(5) = %q, want %q", peeked, "preex")
	}
}
