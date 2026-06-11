package core

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/parser"
	cryptostream "github.com/chainreactors/malice-network/server/internal/stream"
)

type testPacketParser struct {
	marshalData []byte
}

func (p testPacketParser) ReadHeader(io.ReadWriteCloser) (uint32, uint32, error) {
	return 0, 0, nil
}

func (p testPacketParser) Parse([]byte) (*implantpb.Spites, error) {
	return &implantpb.Spites{}, nil
}

func (p testPacketParser) Marshal(*implantpb.Spites, uint32) ([]byte, error) {
	return p.marshalData, nil
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}

type testAddr string

func (a testAddr) Network() string { return "tcp" }
func (a testAddr) String() string  { return string(a) }

type testConnRWC struct {
	io.ReadWriteCloser
}

func (c testConnRWC) RemoteAddr() net.Addr { return testAddr("127.0.0.1:0") }

func TestConnectionReceiveLoopFailureMarksConnectionDead(t *testing.T) {
	conn := &Connection{
		SessionID: "session-a",
		C:         make(chan *clientpb.SpiteRequest, 1),
		Sender:    make(chan *implantpb.Spites, 1),
		cache:     parser.NewSpitesBuf(),
	}
	conn.alive.Store(true)

	errCh := GoGuarded("connection-recv:test", conn.runReceiveLoop, conn.runtimeErrorHandler("receive loop"))
	conn.C <- nil

	err, ok := waitGuardedResult(t, errCh)
	if !ok || err == nil {
		t.Fatal("expected guarded receive loop to return an error")
	}
	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected PanicError, got %T", err)
	}
	if conn.IsAlive() {
		t.Fatal("expected connection to be marked dead")
	}
	if conn.LastError() == nil {
		t.Fatal("expected connection last error to be recorded")
	}
}

func TestConnectionSendReturnsWriteError(t *testing.T) {
	want := errors.New("write failed")
	conn := &Connection{
		SessionID: "session-send",
		RawID:     7,
		Sender:    make(chan *implantpb.Spites, 1),
		Parser: &parser.MessageParser{
			Implant:      "test",
			PacketParser: testPacketParser{marshalData: []byte{1, 2, 3}},
		},
	}
	conn.Sender <- &implantpb.Spites{}

	streamConn := &cryptostream.Conn{
		ReadWriteCloser: testConnRWC{
			ReadWriteCloser: cryptostream.WrapReadWriteCloser(bytes.NewReader(nil), errorWriter{err: want}, nil),
		},
		Parser: conn.Parser,
	}

	err := conn.Send(context.Background(), streamConn)
	if !errors.Is(err, want) {
		t.Fatalf("Send error = %v, want %v", err, want)
	}
}

func TestConnectionsRemoveDeletesConnection(t *testing.T) {
	pool := &connections{connections: &sync.Map{}}
	conn := &Connection{SessionID: "session-remove"}
	conn.alive.Store(true)
	pool.Add(conn)

	pool.Remove(conn.SessionID)

	if got := pool.Get(conn.SessionID); got != nil {
		t.Fatalf("expected connection to be deleted, got %#v", got)
	}
	if conn.IsAlive() {
		t.Fatal("expected connection to be marked dead")
	}
	if !errors.Is(conn.LastError(), ErrConnectionRemoved) {
		t.Fatalf("last error = %v, want %v", conn.LastError(), ErrConnectionRemoved)
	}
}
