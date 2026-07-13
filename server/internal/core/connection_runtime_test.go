package core

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/parser"
	"github.com/chainreactors/malice-network/server/internal/parser/malefic"
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

type errorReader struct {
	err error
}

func (r errorReader) Read([]byte) (int, error) {
	return 0, r.err
}

type testAddr string

func (a testAddr) Network() string { return "tcp" }
func (a testAddr) String() string  { return string(a) }

type testConnRWC struct {
	io.ReadWriteCloser
}

func (c testConnRWC) RemoteAddr() net.Addr { return testAddr("127.0.0.1:0") }

type deadlineRecordingRWC struct {
	io.ReadWriteCloser
	mu             sync.Mutex
	deadlines      []time.Time
	deadlineErrors []error
}

func (c *deadlineRecordingRWC) RemoteAddr() net.Addr { return testAddr("127.0.0.1:0") }

func (c *deadlineRecordingRWC) SetReadDeadline(deadline time.Time) error {
	c.mu.Lock()
	call := len(c.deadlines)
	c.deadlines = append(c.deadlines, deadline)
	defer c.mu.Unlock()
	if call < len(c.deadlineErrors) {
		return c.deadlineErrors[call]
	}
	return nil
}

func (c *deadlineRecordingRWC) readDeadlines() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Time(nil), c.deadlines...)
}

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

func TestConnectionBuildResponseBoundsPayloadRead(t *testing.T) {
	rwc := &deadlineRecordingRWC{
		ReadWriteCloser: cryptostream.WrapReadWriteCloser(bytes.NewReader([]byte{1, 2}), io.Discard, nil),
	}
	packetParser := testPacketParser{}
	streamConn := &cryptostream.Conn{
		ReadWriteCloser: rwc,
		Parser: &parser.MessageParser{
			Implant:      "test",
			PacketParser: packetParser,
		},
	}
	connection := &Connection{
		SessionID:  "payload-deadline",
		PipelineID: "missing-forward",
		Parser:     streamConn.Parser,
	}

	before := time.Now()
	if err := connection.buildResponse(streamConn, 2); err != nil {
		t.Fatalf("buildResponse failed: %v", err)
	}
	deadlines := rwc.readDeadlines()
	if len(deadlines) != 2 {
		t.Fatalf("read deadline calls = %d, want 2", len(deadlines))
	}
	if deadlines[0].Before(before.Add(payloadReadBaseTimeout)) {
		t.Fatalf("payload read deadline = %v, want at least base timeout", deadlines[0])
	}
	if !deadlines[1].IsZero() {
		t.Fatalf("cleared read deadline = %v, want zero", deadlines[1])
	}
}

func TestPayloadReadTimeoutScalesWithDeclaredLength(t *testing.T) {
	if got := payloadReadTimeout(1); got != payloadReadBaseTimeout+time.Second {
		t.Fatalf("timeout for 1 byte = %v", got)
	}
	if got := payloadReadTimeout(payloadReadMinBytesPerSecond * 2); got != payloadReadBaseTimeout+2*time.Second {
		t.Fatalf("timeout for two transfer units = %v", got)
	}
}

func TestConnectionBuildResponseReportsDeadlineFailures(t *testing.T) {
	setErr := errors.New("set deadline failed")
	readErr := errors.New("read failed")
	clearErr := errors.New("clear deadline failed")
	tests := []struct {
		name           string
		reader         io.Reader
		deadlineErrors []error
		wantErrors     []error
	}{
		{
			name:           "set deadline",
			reader:         bytes.NewReader([]byte{1, 2}),
			deadlineErrors: []error{setErr},
			wantErrors:     []error{setErr},
		},
		{
			name:           "clear deadline",
			reader:         bytes.NewReader([]byte{1, 2}),
			deadlineErrors: []error{nil, clearErr},
			wantErrors:     []error{clearErr},
		},
		{
			name:           "read and clear deadline",
			reader:         errorReader{err: readErr},
			deadlineErrors: []error{nil, clearErr},
			wantErrors:     []error{readErr, clearErr},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rwc := &deadlineRecordingRWC{
				ReadWriteCloser: cryptostream.WrapReadWriteCloser(tt.reader, io.Discard, nil),
				deadlineErrors:  tt.deadlineErrors,
			}
			streamConn := &cryptostream.Conn{
				ReadWriteCloser: rwc,
				Parser: &parser.MessageParser{
					Implant:      "test",
					PacketParser: testPacketParser{},
				},
			}
			connection := &Connection{
				SessionID:  "payload-deadline-error",
				PipelineID: "missing-forward",
				Parser:     streamConn.Parser,
			}

			err := connection.buildResponse(streamConn, 2)
			for _, wantErr := range tt.wantErrors {
				if !errors.Is(err, wantErr) {
					t.Fatalf("buildResponse error = %v, want error %v", err, wantErr)
				}
			}
		})
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

func TestConnectionsStaleEOFDoesNotDropNewerConnection(t *testing.T) {
	oldConnections := Connections
	t.Cleanup(func() {
		Connections = oldConnections
	})

	for i := 0; i < 1000; i++ {
		Connections = &connections{connections: &sync.Map{}}
		sessionID := fmt.Sprintf("session-stale-eof-%d", i)
		base := time.Unix(1, 0)

		stale := &Connection{
			SessionID:   sessionID,
			LastMessage: base,
			C:           make(chan *clientpb.SpiteRequest, 1),
			Sender:      make(chan *implantpb.Spites, 1),
		}
		stale.alive.Store(true)

		newer := &Connection{
			SessionID:   sessionID,
			LastMessage: base.Add(time.Millisecond),
			C:           make(chan *clientpb.SpiteRequest, 1),
			Sender:      make(chan *implantpb.Spites, 1),
		}
		newer.alive.Store(true)

		Connections.Add(newer)
		Connections.Add(stale)

		_ = stale.closeWithError(io.EOF)

		if got := Connections.Get(sessionID); got != newer {
			t.Fatalf("iteration %d: stale EOF removed newer connection, got %#v want %#v", i, got, newer)
		}
		if err := Connections.Push(sessionID, &clientpb.SpiteRequest{}); err != nil {
			t.Fatalf("iteration %d: push to newer connection failed after stale EOF: %v", i, err)
		}
	}
}

func TestGetOrReuseConnectionDoesNotReuseAcrossPipelines(t *testing.T) {
	oldConnections := Connections
	t.Cleanup(func() {
		Connections = oldConnections
	})
	Connections = &connections{connections: &sync.Map{}}

	rawID := uint32(0x01020304)
	first := newMaleficProbeConn(t, rawID)
	firstConn, err := GetOrReuseConnection(first, "listener-a:http-a", nil)
	if err != nil {
		t.Fatalf("first GetOrReuseConnection failed: %v", err)
	}
	t.Cleanup(func() {
		_ = firstConn.closeWithError(ErrConnectionRemoved)
	})

	second := newMaleficProbeConn(t, rawID)
	secondConn, err := GetOrReuseConnection(second, "listener-b:http-b", nil)
	if err != nil {
		t.Fatalf("second GetOrReuseConnection failed: %v", err)
	}
	t.Cleanup(func() {
		_ = secondConn.closeWithError(ErrConnectionRemoved)
	})

	if secondConn == firstConn {
		t.Fatalf("connection for raw session %d was reused across different pipelines: got pipeline %q, want %q",
			rawID, secondConn.PipelineID, "listener-b:http-b")
	}
	if secondConn.PipelineID != "listener-b:http-b" {
		t.Fatalf("connection pipeline = %q, want %q", secondConn.PipelineID, "listener-b:http-b")
	}
}

func newMaleficProbeConn(t testing.TB, rawID uint32) *cryptostream.Conn {
	t.Helper()

	header := make([]byte, malefic.HeaderLength)
	header[0] = malefic.DefaultStartDelimiter
	binary.LittleEndian.PutUint32(header[1:5], rawID)
	binary.LittleEndian.PutUint32(header[5:9], 0)

	cryptor, err := cryptostream.NewCryptor(consts.CryptorRAW, nil, nil)
	if err != nil {
		t.Fatalf("new raw cryptor: %v", err)
	}
	conn, err := cryptostream.WrapPeekConn(
		cryptostream.WrapReadWriteCloser(bytes.NewReader(header), io.Discard, nil),
		[]cryptostream.Cryptor{cryptor},
		consts.ImplantMalefic,
		0,
	)
	if err != nil {
		t.Fatalf("wrap probe connection: %v", err)
	}
	return conn
}
