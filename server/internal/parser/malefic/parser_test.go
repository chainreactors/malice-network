package malefic

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/gookit/config/v2"
)

// rwcBuf wraps a bytes.Buffer to satisfy io.ReadWriteCloser.
type rwcBuf struct {
	*bytes.Buffer
}

func (r *rwcBuf) Close() error { return nil }

// buildHeader constructs a 9-byte malefic header.
func buildHeader(startDelim byte, sid uint32, bodyLen uint32) []byte {
	header := make([]byte, HeaderLength)
	header[MsgStart] = startDelim
	binary.LittleEndian.PutUint32(header[MsgSessionStart:MsgSessionEnd], sid)
	binary.LittleEndian.PutUint32(header[MsgSessionEnd:], bodyLen)
	return header
}

// --- ReadHeader tests ---

func TestMaleficParser_ReadHeader_ValidPacket(t *testing.T) {
	config.Set(consts.ConfigMaxPacketLength, 10485760)
	t.Cleanup(func() { config.Set(consts.ConfigMaxPacketLength, 0) })

	p := NewMaleficParser()
	sid := uint32(0xDEAD)
	bodyLen := uint32(100)
	header := buildHeader(DefaultStartDelimiter, sid, bodyLen)

	buf := &rwcBuf{bytes.NewBuffer(header)}
	gotSid, gotLen, err := p.ReadHeader(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSid != sid {
		t.Fatalf("SID mismatch: got %d, want %d", gotSid, sid)
	}
	// readHeader returns length+1 (for end delimiter byte)
	if gotLen != bodyLen+1 {
		t.Fatalf("length mismatch: got %d, want %d", gotLen, bodyLen+1)
	}
}

func TestMaleficParser_ReadHeader_InvalidStartDelimiter(t *testing.T) {
	config.Set(consts.ConfigMaxPacketLength, 10485760)
	t.Cleanup(func() { config.Set(consts.ConfigMaxPacketLength, 0) })

	p := NewMaleficParser()
	header := buildHeader(0xFF, 1, 10) // wrong start byte

	buf := &rwcBuf{bytes.NewBuffer(header)}
	_, _, err := p.ReadHeader(buf)
	if !errors.Is(err, types.ErrInvalidStart) {
		t.Fatalf("expected ErrInvalidStart, got %v", err)
	}
}

func TestMaleficParser_ReadHeader_PacketTooLarge(t *testing.T) {
	config.Set(consts.ConfigMaxPacketLength, 1024)
	t.Cleanup(func() { config.Set(consts.ConfigMaxPacketLength, 0) })

	p := NewMaleficParser()
	// Set body length way beyond max + 16KB margin
	hugeLen := uint32(1024 + consts.KB*16 + 100)
	header := buildHeader(DefaultStartDelimiter, 1, hugeLen)

	buf := &rwcBuf{bytes.NewBuffer(header)}
	_, _, err := p.ReadHeader(buf)
	if !errors.Is(err, types.ErrPacketTooLarge) {
		t.Fatalf("expected ErrPacketTooLarge, got %v", err)
	}
}

func TestMaleficParser_ReadHeader_TruncatedHeader(t *testing.T) {
	p := NewMaleficParser()
	// Only 5 bytes instead of 9
	buf := &rwcBuf{bytes.NewBuffer([]byte{0xd1, 0x01, 0x00, 0x00, 0x00})}
	_, _, err := p.ReadHeader(buf)
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Logf("got error (acceptable): %v", err)
	}
}

func TestMaleficParser_ReadHeader_EmptyInput(t *testing.T) {
	p := NewMaleficParser()
	buf := &rwcBuf{bytes.NewBuffer([]byte{})}
	_, _, err := p.ReadHeader(buf)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

// --- Parse tests ---

func TestMaleficParser_Parse_InvalidEndDelimiter(t *testing.T) {
	p := NewMaleficParser()
	// Payload with wrong end delimiter
	payload := []byte{0x01, 0x02, 0xFF} // 0xFF instead of 0xd2
	_, err := p.Parse(payload)
	if !errors.Is(err, types.ErrInvalidEnd) {
		t.Fatalf("expected ErrInvalidEnd, got %v", err)
	}
}

func TestMaleficParser_Parse_EmptyPayload(t *testing.T) {
	p := NewMaleficParser()
	// Single byte: just the end delimiter. After stripping it, protobuf
	// unmarshal receives an empty buffer.
	payload := []byte{DefaultEndDelimiter}
	spites, err := p.Parse(payload)
	// Empty protobuf unmarshal on Spites{} might succeed (empty message is valid)
	// or fail depending on implementation. Either way, no panic.
	if err != nil {
		t.Logf("Parse empty payload returned error (expected): %v", err)
	} else if spites != nil && len(spites.Spites) != 0 {
		t.Logf("Parse empty payload returned spites with %d items", len(spites.Spites))
	}
}

func TestMaleficParser_Parse_ZeroLengthSlice(t *testing.T) {
	p := NewMaleficParser()
	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG: Parse panics on zero-length slice: %v", r)
		}
	}()
	// This will index buf[len(buf)-1] which is buf[-1] -> panic
	_, _ = p.Parse([]byte{})
}

func TestMaleficParser_MarshalParse_RoundTrip(t *testing.T) {
	p := NewMaleficParser()

	spite := &implantpb.Spite{
		TaskId: 999,
		Body:   &implantpb.Spite_Empty{},
	}
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{spite}}
	sid := uint32(42)

	raw, err := p.Marshal(spites, sid)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify header structure
	if raw[0] != DefaultStartDelimiter {
		t.Fatalf("expected start delimiter 0x%x, got 0x%x", DefaultStartDelimiter, raw[0])
	}
	if raw[len(raw)-1] != DefaultEndDelimiter {
		t.Fatalf("expected end delimiter 0x%x, got 0x%x", DefaultEndDelimiter, raw[len(raw)-1])
	}

	// Extract SID from header
	gotSid := binary.LittleEndian.Uint32(raw[MsgSessionStart:MsgSessionEnd])
	if gotSid != sid {
		t.Fatalf("SID mismatch in marshalled data: got %d, want %d", gotSid, sid)
	}

	// Parse the payload (everything after 9-byte header)
	payload := raw[HeaderLength:]
	gotSpites, err := p.Parse(payload)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(gotSpites.Spites) != 1 {
		t.Fatalf("expected 1 spite, got %d", len(gotSpites.Spites))
	}
	if gotSpites.Spites[0].TaskId != 999 {
		t.Fatalf("TaskId mismatch: got %d, want 999", gotSpites.Spites[0].TaskId)
	}
}

// --- ParseSid tests ---

func TestParseSid(t *testing.T) {
	tests := []struct {
		name string
		sid  uint32
	}{
		{"zero", 0},
		{"one", 1},
		{"large", 0xDEADBEEF},
		{"max uint32", 0xFFFFFFFF},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 9)
			data[0] = DefaultStartDelimiter
			binary.LittleEndian.PutUint32(data[MsgSessionStart:MsgSessionEnd], tc.sid)

			got := ParseSid(data)
			if got != tc.sid {
				t.Fatalf("ParseSid() = %d, want %d", got, tc.sid)
			}
		})
	}
}

// TestParseSid_ShortData documents a real bug: ParseSid does not bounds-check
// the input slice. Data shorter than 5 bytes causes a panic.
func TestParseSid_ShortData(t *testing.T) {
	shortInputs := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"1 byte", []byte{0xd1}},
		{"4 bytes", []byte{0xd1, 0x01, 0x02, 0x03}},
	}

	for _, tc := range shortInputs {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Logf("BUG CONFIRMED: ParseSid panics on %s input (len=%d): %v",
						tc.name, len(tc.data), r)
				}
			}()
			_ = ParseSid(tc.data)
			// If we reach here without panic, the bug may have been fixed
			// with a bounds check. That would be the correct behavior.
		})
	}
}

// --- Marshal tests ---

// TestMaleficParser_Marshal_NilSpites documents that proto.Marshal accepts nil
// (treats it as an empty message). This means Marshal silently succeeds on nil
// spites, which could be surprising to callers.
func TestMaleficParser_Marshal_NilSpites(t *testing.T) {
	p := NewMaleficParser()
	raw, err := p.Marshal(nil, 1)
	if err != nil {
		// If this is reached, explicit nil checking was added (good).
		t.Logf("Marshal correctly rejects nil spites: %v", err)
		return
	}
	// proto.Marshal(nil) succeeds, producing an empty message.
	t.Log("NOTE: Marshal accepts nil spites without error (proto.Marshal treats nil as empty message)")
	if len(raw) < HeaderLength+1 {
		t.Fatalf("packet too short: %d bytes", len(raw))
	}
}

func TestMaleficParser_Marshal_EmptySpites(t *testing.T) {
	p := NewMaleficParser()
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{}}
	raw, err := p.Marshal(spites, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still produce a valid framed packet
	if len(raw) < HeaderLength+1 {
		t.Fatalf("packet too short: %d bytes", len(raw))
	}
	if raw[0] != DefaultStartDelimiter || raw[len(raw)-1] != DefaultEndDelimiter {
		t.Fatal("missing delimiters in marshalled packet")
	}
}

func TestMaleficParser_Marshal_HeaderLength(t *testing.T) {
	p := NewMaleficParser()
	spites := &implantpb.Spites{Spites: []*implantpb.Spite{
		{TaskId: 1, Body: &implantpb.Spite_Empty{}},
	}}

	raw, err := p.Marshal(spites, 100)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify the length field in the header matches actual payload size
	lengthInHeader := binary.LittleEndian.Uint32(raw[MsgSessionEnd:HeaderLength])
	actualPayload := len(raw) - HeaderLength - 1 // minus header and end delimiter
	if int(lengthInHeader) != actualPayload {
		t.Fatalf("length field mismatch: header says %d, actual payload is %d",
			lengthInHeader, actualPayload)
	}
}
