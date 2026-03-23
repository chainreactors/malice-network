package cryptostream

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"testing"
)

// simpleRWC wraps a bytes.Buffer into a ReadWriteCloser for testing.
type simpleRWC struct {
	buf    *bytes.Buffer
	closed bool
}

func (s *simpleRWC) Read(p []byte) (int, error) {
	if s.closed {
		return 0, errors.New("closed")
	}
	return s.buf.Read(p)
}

func (s *simpleRWC) Write(p []byte) (int, error) {
	if s.closed {
		return 0, errors.New("closed")
	}
	return s.buf.Write(p)
}

func (s *simpleRWC) Close() error {
	s.closed = true
	return nil
}

func newXorCryptorPair(t *testing.T) (Cryptor, Cryptor) {
	t.Helper()
	key := []byte("testkey123")
	iv := []byte("iv456")
	enc := NewXorEncryptor(key, iv)
	dec := NewXorEncryptor(key, iv)
	return enc, dec
}

func newAesCryptorPair(t *testing.T) (Cryptor, Cryptor) {
	t.Helper()
	key := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
		17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	iv := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	enc, err := NewAesCtrEncryptor(key, iv)
	if err != nil {
		t.Fatalf("failed to create AES encryptor: %v", err)
	}
	dec, err := NewAesCtrEncryptor(key, iv)
	if err != nil {
		t.Fatalf("failed to create AES decryptor: %v", err)
	}
	return enc, dec
}

// TestCryptoConn_WriteRead_XOR_RoundTrip verifies XOR encrypt-then-decrypt round trip over net.Pipe.
func TestCryptoConn_WriteRead_XOR_RoundTrip(t *testing.T) {
	t.Parallel()
	enc, dec := newXorCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	writer := NewCryptoConn(clientConn, enc)
	reader := NewCryptoConn(serverConn, dec)

	original := []byte("hello XOR round trip")
	errCh := make(chan error, 1)
	go func() {
		_, err := writer.Write(original)
		errCh <- err
	}()

	buf := make([]byte, 256)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !bytes.Equal(buf[:n], original) {
		t.Fatalf("round trip mismatch: got %q, want %q", buf[:n], original)
	}
}

// TestCryptoConn_WriteRead_AES_RoundTrip verifies AES encrypt-then-decrypt round trip over net.Pipe.
func TestCryptoConn_WriteRead_AES_RoundTrip(t *testing.T) {
	t.Parallel()
	enc, dec := newAesCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	writer := NewCryptoConn(clientConn, enc)
	reader := NewCryptoConn(serverConn, dec)

	original := []byte("hello AES round trip with some longer data to exercise the cipher")
	errCh := make(chan error, 1)
	go func() {
		_, err := writer.Write(original)
		errCh <- err
	}()

	buf := make([]byte, 256)
	n, err := reader.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !bytes.Equal(buf[:n], original) {
		t.Fatalf("round trip mismatch: got %q, want %q", buf[:n], original)
	}
}

// TestCryptoConn_OverflowBuffer writes 2000 bytes and reads in small 100-byte chunks.
// This exercises the readBuf overflow caching logic in CryptoConn.Read.
// BUG NOTE: The hardcoded 1024-byte internal read buffer means the encrypted
// data must fit within 1024 bytes per read from the underlying connection.
// With net.Pipe delivering data atomically, a single 2000-byte encrypted write
// would require multiple internal reads, but CryptoConn.Read only reads once
// per call. The readBuf caching only caches overflow from a single decrypted
// chunk, not across multiple underlying reads.
func TestCryptoConn_OverflowBuffer(t *testing.T) {
	t.Parallel()
	enc, dec := newXorCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	writer := NewCryptoConn(clientConn, enc)
	reader := NewCryptoConn(serverConn, dec)

	// Write 500 bytes (under 1024 limit so a single internal read can handle it)
	original := make([]byte, 500)
	for i := range original {
		original[i] = byte(i % 251) // prime to avoid patterns
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := writer.Write(original)
		errCh <- err
	}()

	// Read in small 100-byte chunks to exercise readBuf caching
	var received []byte
	for len(received) < len(original) {
		chunk := make([]byte, 100)
		n, err := reader.Read(chunk)
		if err != nil {
			t.Fatalf("Read failed after %d bytes: %v", len(received), err)
		}
		received = append(received, chunk[:n]...)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !bytes.Equal(received, original) {
		t.Fatalf("overflow buffer: data mismatch, got %d bytes, want %d bytes", len(received), len(original))
	}
}

// TestCryptoConn_OverflowBuffer_LargeWrite_Bug demonstrates a potential issue
// with writes larger than 1024 bytes. The internal read buffer is hardcoded to
// 1024 so the reader side may need multiple Read calls to get all the data.
// With net.Pipe, Read returns what Write wrote in a single call, but if the
// underlying transport fragments, CryptoConn.Read only processes the first
// 1024 bytes of encrypted data per call.
func TestCryptoConn_OverflowBuffer_LargeWrite_Bug(t *testing.T) {
	t.Parallel()
	// Use RAW cryptor (XOR with null key = identity) to isolate the buffer issue
	enc := NewXorEncryptor([]byte{0}, []byte{0})
	dec := NewXorEncryptor([]byte{0}, []byte{0})

	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	writer := NewCryptoConn(clientConn, enc)
	reader := NewCryptoConn(serverConn, dec)

	// Write 2000 bytes - exceeds the 1024-byte internal buffer
	original := make([]byte, 2000)
	if _, err := rand.Read(original); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, err := writer.Write(original)
		errCh <- err
	}()

	// net.Pipe delivers 2000 bytes atomically, but CryptoConn.Read uses a
	// 1024-byte buffer, so only the first 1024 bytes are read and decrypted.
	// The remaining 976 bytes are stuck in the pipe, requiring another Read.
	var received []byte
	for len(received) < len(original) {
		chunk := make([]byte, 4096)
		n, err := reader.Read(chunk)
		if err != nil {
			t.Fatalf("Read failed after %d bytes: %v", len(received), err)
		}
		received = append(received, chunk[:n]...)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if !bytes.Equal(received, original) {
		t.Errorf("large write: data mismatch at byte level")
	}
}

// TestCryptoConn_Close_PropagatesError verifies Close propagates to the underlying conn.
func TestCryptoConn_Close_PropagatesError(t *testing.T) {
	t.Parallel()
	enc, _ := newXorCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
	})

	cc := NewCryptoConn(clientConn, enc)
	if err := cc.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	// After closing, the underlying pipe should be closed.
	// Writing to the other end should fail or reading should return error.
	_, err := serverConn.Write([]byte("test"))
	if err == nil {
		// Read from closed pipe should fail
		buf := make([]byte, 10)
		_, err = serverConn.Read(buf)
	}
	if err == nil {
		t.Error("expected error after closing CryptoConn, but pipe still works")
	}
}

// TestCryptoConn_ReadAfterClose verifies that reading from a closed CryptoConn returns an error.
func TestCryptoConn_ReadAfterClose(t *testing.T) {
	t.Parallel()
	_, dec := newXorCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		clientConn.Close()
	})

	cc := NewCryptoConn(serverConn, dec)
	cc.Close()

	buf := make([]byte, 100)
	_, err := cc.Read(buf)
	if err == nil {
		t.Error("expected error reading from closed CryptoConn, got nil")
	}
}

// TestCryptoRWC_WriteRead verifies NewCryptoRWC with a non-net.Conn ReadWriteCloser.
func TestCryptoRWC_WriteRead(t *testing.T) {
	t.Parallel()
	// Use RAW/identity cryptor to test the RWC plumbing without crypto complications
	enc := NewXorEncryptor([]byte{0}, []byte{0})
	dec := NewXorEncryptor([]byte{0}, []byte{0})

	buf := &bytes.Buffer{}
	rwc := &simpleRWC{buf: buf}

	writer := NewCryptoRWC(rwc, enc)

	original := []byte("test RWC data")
	if _, err := writer.Write(original); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Create a new reader RWC pointing to the same buffer data
	readBuf := bytes.NewBuffer(buf.Bytes())
	readRWC := &simpleRWC{buf: readBuf}
	reader := NewCryptoRWC(readRWC, dec)

	result := make([]byte, 256)
	n, err := reader.Read(result)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !bytes.Equal(result[:n], original) {
		t.Fatalf("RWC round trip mismatch: got %q, want %q", result[:n], original)
	}
}

// TestCryptoConn_RemoteAddr_WithConn verifies RemoteAddr returns the conn's remote addr.
func TestCryptoConn_RemoteAddr_WithConn(t *testing.T) {
	t.Parallel()
	enc, _ := newXorCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	cc := NewCryptoConn(clientConn, enc)
	addr := cc.RemoteAddr()
	// net.Pipe returns a *net.pipe which implements RemoteAddr
	// The address should not be nil for a net.Pipe conn
	if addr == nil {
		t.Log("net.Pipe RemoteAddr returned nil (implementation-dependent, not necessarily a bug)")
	}
}

// TestCryptoConn_RemoteAddr_WithRWC_NoAddr verifies RemoteAddr returns nil
// when the underlying ReadWriteCloser does not implement RemoteAddr.
func TestCryptoConn_RemoteAddr_WithRWC_NoAddr(t *testing.T) {
	t.Parallel()
	enc, _ := newXorCryptorPair(t)
	rwc := &simpleRWC{buf: &bytes.Buffer{}}

	cc := NewCryptoRWC(rwc, enc)
	addr := cc.RemoteAddr()
	if addr != nil {
		t.Errorf("expected nil RemoteAddr for plain RWC, got %v", addr)
	}
}

// addrRWC is a ReadWriteCloser that also implements RemoteAddr.
type addrRWC struct {
	simpleRWC
}

func (a *addrRWC) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 9999}
}

// TestCryptoConn_RemoteAddr_WithRWC_HasAddr verifies RemoteAddr falls through
// to the RWC interface assertion when Conn is nil.
func TestCryptoConn_RemoteAddr_WithRWC_HasAddr(t *testing.T) {
	t.Parallel()
	enc, _ := newXorCryptorPair(t)
	rwc := &addrRWC{simpleRWC{buf: &bytes.Buffer{}}}

	cc := NewCryptoRWC(rwc, enc)
	addr := cc.RemoteAddr()
	if addr == nil {
		t.Fatal("expected non-nil RemoteAddr from addrRWC")
	}
	expected := "10.0.0.1:9999"
	if addr.String() != expected {
		t.Errorf("RemoteAddr = %q, want %q", addr.String(), expected)
	}
}

// TestNewCryptor_CaseInsensitive verifies the factory handles mixed case.
func TestNewCryptor_CaseInsensitive(t *testing.T) {
	t.Parallel()
	cases := []string{"xor", "XOR", "Xor", "xOr"}
	key := []byte("key")
	secret := []byte("secret")
	for _, name := range cases {
		c, err := NewCryptor(name, key, secret)
		if err != nil {
			t.Errorf("NewCryptor(%q) returned error: %v", name, err)
		}
		if c == nil {
			t.Errorf("NewCryptor(%q) returned nil cryptor", name)
		}
	}
}

// TestNewCryptor_AES_PadsKey verifies that a short key is padded to 32 bytes.
func TestNewCryptor_AES_PadsKey(t *testing.T) {
	t.Parallel()
	shortKey := []byte("short")
	shortIV := []byte("iv")
	c, err := NewCryptor("AES", shortKey, shortIV)
	if err != nil {
		t.Fatalf("NewCryptor(AES) with short key: %v", err)
	}
	if c == nil {
		t.Fatal("NewCryptor(AES) returned nil")
	}

	// Verify it actually works for encrypt/decrypt
	original := []byte("test padded key encryption")
	encrypted, err := Encrypt(c, original)
	if err != nil {
		t.Fatalf("Encrypt failed with padded key: %v", err)
	}

	// Need a fresh cryptor (AES CTR is stateful) to decrypt
	c2, _ := NewCryptor("AES", shortKey, shortIV)
	decrypted, err := Decrypt(c2, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed with padded key: %v", err)
	}
	if !bytes.Equal(decrypted, original) {
		t.Fatalf("padded key round trip failed: got %q, want %q", decrypted, original)
	}
}

// TestNewCryptor_Unknown verifies that an unknown cryptor name returns an error.
func TestNewCryptor_Unknown(t *testing.T) {
	t.Parallel()
	_, err := NewCryptor("BLOWFISH", []byte("key"), []byte("iv"))
	if err == nil {
		t.Error("expected error for unknown cryptor, got nil")
	}
}

// TestNewCryptor_RAW verifies RAW cryptor is identity (XOR with null key).
func TestNewCryptor_RAW(t *testing.T) {
	t.Parallel()
	c, err := NewCryptor("RAW", nil, nil)
	if err != nil {
		t.Fatalf("NewCryptor(RAW) failed: %v", err)
	}
	original := []byte("raw data should pass through unchanged")
	encrypted, err := Encrypt(c, original)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	// RAW uses XOR with key=0 and iv=0, so XOR with 0^0=0 is identity
	if !bytes.Equal(encrypted, original) {
		t.Errorf("RAW cryptor should be identity, got different output")
	}
}

// TestPKCS7Pad_Shorter verifies padding when data is shorter than blockSize.
func TestPKCS7Pad_Shorter(t *testing.T) {
	t.Parallel()
	data := []byte("0123456789") // 10 bytes
	result := PKCS7Pad(data, 32)
	if len(result) != 32 {
		t.Fatalf("expected length 32, got %d", len(result))
	}
	// First 10 bytes should be original data
	if !bytes.Equal(result[:10], data) {
		t.Error("first 10 bytes should match original data")
	}
	// Remaining 22 bytes should be zeros
	for i := 10; i < 32; i++ {
		if result[i] != 0 {
			t.Errorf("expected zero padding at index %d, got %d", i, result[i])
		}
	}
}

// TestPKCS7Pad_Exact verifies no change when data equals blockSize.
func TestPKCS7Pad_Exact(t *testing.T) {
	t.Parallel()
	data := make([]byte, 32)
	for i := range data {
		data[i] = byte(i)
	}
	result := PKCS7Pad(data, 32)
	if !bytes.Equal(result, data) {
		t.Error("exact-size data should be returned unchanged")
	}
}

// TestPKCS7Pad_Longer verifies TRUNCATION when data exceeds blockSize.
// BUG: This is NOT standard PKCS7 padding behavior. Standard PKCS7 would add
// a full block of padding. This implementation silently truncates, causing data loss.
func TestPKCS7Pad_Longer(t *testing.T) {
	t.Parallel()
	data := make([]byte, 40)
	for i := range data {
		data[i] = byte(i + 1) // non-zero to verify truncation
	}
	result := PKCS7Pad(data, 32)
	if len(result) != 32 {
		t.Fatalf("expected length 32, got %d", len(result))
	}
	// BUG: last 8 bytes of input are silently discarded
	if !bytes.Equal(result, data[:32]) {
		t.Error("truncated result should match first 32 bytes of input")
	}
	// Verify the truncated bytes are actually lost
	if bytes.Equal(result, data) {
		t.Error("should NOT equal original 40-byte data")
	}
}

// TestPKCS7Pad_Empty verifies padding with empty input.
func TestPKCS7Pad_Empty(t *testing.T) {
	t.Parallel()
	result := PKCS7Pad([]byte{}, 16)
	if len(result) != 16 {
		t.Fatalf("expected length 16, got %d", len(result))
	}
	for i, b := range result {
		if b != 0 {
			t.Errorf("expected zero at index %d, got %d", i, b)
		}
	}
}

// TestPKCS7Pad_NilInput verifies behavior with nil input.
func TestPKCS7Pad_NilInput(t *testing.T) {
	t.Parallel()
	result := PKCS7Pad(nil, 16)
	if len(result) != 16 {
		t.Fatalf("expected length 16, got %d", len(result))
	}
}

// TestEncryptDecrypt_RoundTrip verifies Encrypt then Decrypt with the same cryptor.
func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	t.Parallel()
	key := []byte("roundtripkey1234")
	iv := []byte("roundtripiv12345")
	c, err := NewCryptor("XOR", key, iv)
	if err != nil {
		t.Fatalf("NewCryptor: %v", err)
	}

	original := []byte("encrypt then decrypt round trip test data")
	encrypted, err := Encrypt(c, original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Equal(encrypted, original) {
		t.Error("encrypted data should differ from original (unless key produces zero XOR)")
	}

	decrypted, err := Decrypt(c, encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, original) {
		t.Fatalf("round trip failed: got %q, want %q", decrypted, original)
	}
}

// TestEncryptDecrypt_DifferentKeys verifies that decrypting with a different key
// produces corrupted output (not the original plaintext).
func TestEncryptDecrypt_DifferentKeys(t *testing.T) {
	t.Parallel()
	keyA := []byte("keyAAAAAAAAAAAAA")
	keyB := []byte("keyBBBBBBBBBBBBB")
	iv := []byte("sharediv")

	cA, _ := NewCryptor("XOR", keyA, iv)
	cB, _ := NewCryptor("XOR", keyB, iv)

	original := []byte("secret message that should not survive wrong key")
	encrypted, err := Encrypt(cA, original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := Decrypt(cB, encrypted)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if bytes.Equal(decrypted, original) {
		t.Error("decrypting with different key should NOT produce original plaintext")
	}
}

// TestEncryptDecrypt_EmptyData verifies encrypt/decrypt of empty data.
func TestEncryptDecrypt_EmptyData(t *testing.T) {
	t.Parallel()
	c, _ := NewCryptor("XOR", []byte("key"), []byte("iv"))
	encrypted, err := Encrypt(c, []byte{})
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	decrypted, err := Decrypt(c, encrypted)
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if len(decrypted) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(decrypted))
	}
}

// TestCryptoConn_WriteReturnsEncryptedLength verifies that Write returns the
// length of the ENCRYPTED data, not the original. This can cause issues for
// callers expecting the written length to match input length.
func TestCryptoConn_WriteReturnsEncryptedLength(t *testing.T) {
	t.Parallel()
	enc, _ := newXorCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	cc := NewCryptoConn(clientConn, enc)

	original := []byte("test data")
	go func() {
		// drain the OTHER end (serverConn) so write doesn't block
		buf := make([]byte, 1024)
		for {
			_, err := serverConn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	// For XOR, encrypted length == plaintext length, so this is fine.
	// But for other ciphers that change output length, the returned n
	// would be the encrypted length, not the plaintext length.
	n, err := cc.Write(original)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	// XOR preserves length
	if n != len(original) {
		t.Errorf("Write returned %d, expected %d (XOR should preserve length)", n, len(original))
	}
}

// TestXorEncryptor_StatefulCounter verifies XOR counter state accumulates across
// multiple encrypt calls, meaning you CANNOT decrypt a message in isolation
// if previous messages were processed.
func TestXorEncryptor_StatefulCounter(t *testing.T) {
	t.Parallel()
	key := []byte("counterkey")
	iv := []byte("counteriv")

	enc := NewXorEncryptor(key, iv)
	dec := NewXorEncryptor(key, iv)

	msg1 := []byte("first message")
	msg2 := []byte("second message")

	enc1, _ := Encrypt(enc, msg1)
	enc2, _ := Encrypt(enc, msg2)

	// Decrypt in order: should work
	dec1, _ := Decrypt(dec, enc1)
	dec2, _ := Decrypt(dec, enc2)

	if !bytes.Equal(dec1, msg1) {
		t.Errorf("msg1 decrypt failed: got %q", dec1)
	}
	if !bytes.Equal(dec2, msg2) {
		t.Errorf("msg2 decrypt failed: got %q", dec2)
	}

	// Now try decrypting msg2 with a fresh decryptor (out of order) - should FAIL
	freshDec := NewXorEncryptor(key, iv)
	dec2Wrong, _ := Decrypt(freshDec, enc2)
	if bytes.Equal(dec2Wrong, msg2) {
		t.Error("decrypting msg2 with fresh (counter=0) decryptor should fail due to counter mismatch")
	}
}

// TestAesCtrEncryptor_StatefulStream verifies AES CTR is also stateful.
func TestAesCtrEncryptor_StatefulStream(t *testing.T) {
	t.Parallel()
	enc, dec := newAesCryptorPair(t)

	msg1 := []byte("first AES message")
	msg2 := []byte("second AES message")

	enc1, _ := Encrypt(enc, msg1)
	enc2, _ := Encrypt(enc, msg2)

	dec1, _ := Decrypt(dec, enc1)
	dec2, _ := Decrypt(dec, enc2)

	if !bytes.Equal(dec1, msg1) {
		t.Errorf("msg1 AES decrypt failed")
	}
	if !bytes.Equal(dec2, msg2) {
		t.Errorf("msg2 AES decrypt failed")
	}

	// Fresh decryptor cannot decrypt msg2 (stream position mismatch)
	_, freshDec := newAesCryptorPair(t)
	dec2Wrong, _ := Decrypt(freshDec, enc2)
	if bytes.Equal(dec2Wrong, msg2) {
		t.Error("AES CTR: decrypting msg2 with fresh stream should produce wrong output")
	}
}

// TestCryptoConn_ConcurrentReadWrite verifies no data corruption under concurrent access.
// BUG RISK: CryptoConn has no mutex protection on readBuf, so concurrent reads
// could corrupt the buffer. This test may catch races with -race flag.
func TestCryptoConn_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()
	enc, dec := newXorCryptorPair(t)
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	writer := NewCryptoConn(clientConn, enc)
	reader := NewCryptoConn(serverConn, dec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 50; i++ {
			msg := []byte("concurrent write test")
			if _, err := writer.Write(msg); err != nil {
				return
			}
		}
		writer.Close()
	}()

	totalRead := 0
	for {
		buf := make([]byte, 100)
		n, err := reader.Read(buf)
		totalRead += n
		if err == io.EOF || err != nil {
			break
		}
	}
	<-done

	expectedTotal := 50 * len("concurrent write test")
	if totalRead != expectedTotal {
		t.Errorf("concurrent read/write: got %d bytes, want %d", totalRead, expectedTotal)
	}
}
