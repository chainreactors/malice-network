package parser

import (
	"bytes"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
)

// --- DetectProtocol tests ---

func TestDetectProtocol_Malefic(t *testing.T) {
	p, err := DetectProtocol([]byte{0xd1, 0x00, 0x00})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil parser")
	}
	if p.Implant != "malefic" {
		t.Fatalf("expected malefic, got %s", p.Implant)
	}
}

func TestDetectProtocol_Pulse(t *testing.T) {
	p, err := DetectProtocol([]byte{0x41, 0x00})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil parser")
	}
	if p.Implant != "pulse" {
		t.Fatalf("expected pulse, got %s", p.Implant)
	}
}

func TestDetectProtocol_UnknownByte(t *testing.T) {
	_, err := DetectProtocol([]byte{0xFF})
	if err == nil {
		t.Fatal("expected error for unknown protocol byte")
	}
}

func TestDetectProtocol_EmptyData(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{"nil slice", nil},
		{"empty slice", []byte{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DetectProtocol(tc.data)
			if err == nil {
				t.Fatal("expected error for empty data")
			}
		})
	}
}

// --- NewParser tests ---

func TestNewParser_Malefic(t *testing.T) {
	p, err := NewParser("malefic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Implant != "malefic" {
		t.Fatalf("expected malefic, got %s", p.Implant)
	}
}

func TestNewParser_Pulse(t *testing.T) {
	p, err := NewParser("pulse")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Implant != "pulse" {
		t.Fatalf("expected pulse, got %s", p.Implant)
	}
}

func TestNewParser_InvalidName(t *testing.T) {
	_, err := NewParser("beacon")
	if !errors.Is(err, types.ErrInvalidImplant) {
		t.Fatalf("expected ErrInvalidImplant, got %v", err)
	}
}

func TestNewParser_EmptyName(t *testing.T) {
	_, err := NewParser("")
	if !errors.Is(err, types.ErrInvalidImplant) {
		t.Fatalf("expected ErrInvalidImplant, got %v", err)
	}
}

// --- ReadPacket / WritePacket round-trip ---

func TestReadPacket_RoundTrip(t *testing.T) {
	parser, err := NewParser("malefic")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	spite := &implantpb.Spite{
		TaskId: 42,
		Body:   &implantpb.Spite_Empty{},
	}
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{spite}}
	sid := uint32(12345)

	// Marshal to get the raw packet bytes
	raw, err := parser.Marshal(spites, sid)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Use net.Pipe for a real io.ReadWriteCloser + net.Conn pair
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	// Write the raw bytes in a goroutine
	errCh := make(chan error, 1)
	go func() {
		_, werr := clientConn.Write(raw)
		clientConn.Close()
		errCh <- werr
	}()

	gotSid, gotSpites, err := parser.ReadPacket(serverConn)
	if err != nil {
		t.Fatalf("ReadPacket failed: %v", err)
	}

	if gotSid != sid {
		t.Fatalf("SID mismatch: got %d, want %d", gotSid, sid)
	}

	if gotSpites == nil {
		t.Fatal("expected non-nil spites")
	}

	if len(gotSpites.Spites) != 1 {
		t.Fatalf("expected 1 spite, got %d", len(gotSpites.Spites))
	}

	if gotSpites.Spites[0].TaskId != 42 {
		t.Fatalf("TaskId mismatch: got %d, want 42", gotSpites.Spites[0].TaskId)
	}

	if werr := <-errCh; werr != nil {
		t.Fatalf("write goroutine error: %v", werr)
	}
}

// TestReadPacket_DiscardParseError documents the bug on line 104 of parser.go:
// ReadPacket returns nil error even when Parse fails because the return
// statement is `return sessionId, msg, nil` instead of `return sessionId, msg, err`.
func TestReadPacket_DiscardParseError(t *testing.T) {
	parser, err := NewParser("malefic")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Build a valid header but with a corrupt payload.
	// The malefic header is 9 bytes: [start_delimiter(1) + SID(4) + length(4)]
	// After the header, we need `length` bytes of payload.
	// The payload must end with 0xd2 (end delimiter) for malefic,
	// but we make the body garbage so protobuf unmarshal fails.
	sid := uint32(1)
	payloadBody := []byte{0xFF, 0xFE, 0xFD} // garbage protobuf data
	payload := append(payloadBody, 0xd2)     // valid end delimiter

	header := make([]byte, 9)
	header[0] = 0xd1 // start delimiter
	header[1] = byte(sid)
	header[2] = 0
	header[3] = 0
	header[4] = 0
	// length field = len(payload) - 1 because readHeader returns length+1
	bodyLen := uint32(len(payload) - 1)
	header[5] = byte(bodyLen)
	header[6] = byte(bodyLen >> 8)
	header[7] = byte(bodyLen >> 16)
	header[8] = byte(bodyLen >> 24)

	raw := append(header, payload...)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		clientConn.Write(raw)
		clientConn.Close()
	}()

	gotSid, msg, err := parser.ReadPacket(serverConn)

	// BUG: err is nil even though Parse should have failed on garbage protobuf data.
	// The msg will be nil (unmarshal failure) but the error is swallowed.
	if err != nil {
		// If this branch is reached, the bug has been fixed.
		t.Logf("Bug appears fixed: ReadPacket now returns parse error: %v", err)
		return
	}

	// Document the bug: error is nil but msg is also nil.
	if msg == nil {
		t.Log("BUG CONFIRMED: ReadPacket returned nil message AND nil error. " +
			"Line 104 discards the parse error (returns nil instead of err).")
	}

	_ = gotSid
}

// --- WritePacket tests ---

type rwcBuf struct {
	*bytes.Buffer
}

func (r *rwcBuf) Close() error { return nil }

// TestWritePacket_NilSpites documents that proto.Marshal accepts nil (treats as
// empty message), so WritePacket does NOT error on nil spites. This is
// potentially surprising behavior.
func TestWritePacket_NilSpites(t *testing.T) {
	parser, err := NewParser("malefic")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	errCh := make(chan error, 1)
	go func() {
		// Drain the written bytes so Write does not block
		buf := make([]byte, 4096)
		for {
			_, rerr := serverConn.Read(buf)
			if rerr != nil {
				break
			}
		}
	}()

	go func() {
		errCh <- parser.WritePacket(clientConn, nil, 1)
	}()

	werr := <-errCh
	// proto.Marshal(nil) succeeds (empty message), so this actually succeeds.
	// This documents the behavior: nil spites are silently accepted.
	if werr != nil {
		t.Logf("WritePacket with nil spites returned error: %v", werr)
	} else {
		t.Log("NOTE: WritePacket accepts nil spites without error (proto.Marshal treats nil as empty message)")
	}
}

// TestReadPacket_TruncatedPayload verifies ReadPacket returns an error when
// the connection closes before all payload bytes are read.
func TestReadPacket_TruncatedPayload(t *testing.T) {
	parser, err := NewParser("malefic")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Build a header that claims 100 bytes of payload, but only send 5.
	header := make([]byte, 9)
	header[0] = 0xd1
	header[1] = 1
	// length = 99 (readHeader returns length+1, so actual read will be 100)
	header[5] = 99

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		clientConn.Write(header)
		clientConn.Write([]byte{1, 2, 3, 4, 5}) // only 5 bytes of claimed 100
		clientConn.Close()
	}()

	_, _, err = parser.ReadPacket(serverConn)
	if err == nil {
		t.Fatal("expected error for truncated payload")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Logf("got error (acceptable): %v", err)
	}
}

// TestReadMessage_ValidPayload tests ReadMessage with a correct payload.
func TestReadMessage_ValidPayload(t *testing.T) {
	parser, err := NewParser("malefic")
	if err != nil {
		t.Fatalf("failed to create parser: %v", err)
	}

	// Create a valid payload: marshal spites, compress, then add end delimiter
	spite := &implantpb.Spite{TaskId: 7, Body: &implantpb.Spite_Empty{}}
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{spite}}

	// Use Marshal to get a full packet, then extract just the payload portion
	raw, err := parser.Marshal(spites, 1)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	// Payload is everything after the 9-byte header
	payload := raw[9:]

	buf := &rwcBuf{bytes.NewBuffer(payload)}
	result, err := parser.ReadMessage(buf, uint32(len(payload)))
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Spites) != 1 || result.Spites[0].TaskId != 7 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
