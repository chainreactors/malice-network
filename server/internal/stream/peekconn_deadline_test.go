package cryptostream

import (
	"encoding/binary"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
)

func TestWrapPeekConnTimesOutIncompleteInitialPacket(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})
	cryptor, err := NewCryptor(consts.CryptorRAW, nil, nil)
	if err != nil {
		t.Fatalf("new raw cryptor: %v", err)
	}

	started := time.Now()
	_, err = wrapPeekConn(serverConn, []Cryptor{cryptor}, consts.ImplantMalefic, 0, 20*time.Millisecond)
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("WrapPeekConn error = %v, want network timeout", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("initial packet timeout took %s, want under 1s", elapsed)
	}
}

func TestWrapPeekConnClearsInitialPacketDeadline(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		_ = serverConn.Close()
		_ = clientConn.Close()
	})
	header := make([]byte, malefic.HeaderLength)
	header[0] = malefic.DefaultStartDelimiter
	binary.LittleEndian.PutUint32(header[1:5], 7)
	binary.LittleEndian.PutUint32(header[5:9], 0)
	go func() { _, _ = clientConn.Write(header) }()

	cryptor, err := NewCryptor(consts.CryptorRAW, nil, nil)
	if err != nil {
		t.Fatalf("new raw cryptor: %v", err)
	}
	conn, err := wrapPeekConn(serverConn, []Cryptor{cryptor}, consts.ImplantMalefic, 0, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("WrapPeekConn failed: %v", err)
	}
	initial := make([]byte, len(header))
	if _, err := conn.Read(initial); err != nil {
		t.Fatalf("drain initial packet: %v", err)
	}

	time.Sleep(40 * time.Millisecond)
	readResult := make(chan error, 1)
	go func() {
		buf := make([]byte, 1)
		_, readErr := conn.Read(buf)
		if readErr == nil && buf[0] != 0x42 {
			readErr = errors.New("unexpected byte after initial packet")
		}
		readResult <- readErr
	}()
	go func() { _, _ = clientConn.Write([]byte{0x42}) }()
	select {
	case err := <-readResult:
		if err != nil {
			t.Fatalf("read after initial deadline: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("read stayed blocked after initial deadline should have been cleared")
	}
}
