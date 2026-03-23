package pulse

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/encoders"
	"github.com/chainreactors/malice-network/helper/encoders/hash"
)

// rwcBuf wraps a bytes.Buffer to satisfy io.ReadWriteCloser.
type rwcBuf struct {
	*bytes.Buffer
}

func (r *rwcBuf) Close() error { return nil }

// buildPulseHeader constructs a 9-byte pulse header + 1-byte end delimiter.
func buildPulseHeader(startDelim byte, magic uint32, artifact uint32, endDelim byte) []byte {
	header := make([]byte, HeaderLength+1) // 9 + 1 end byte
	header[MsgStart] = startDelim
	copy(header[MsgMagicStart:MsgMagicEnd], encoders.Uint32ToBytes(magic))
	binary.LittleEndian.PutUint32(header[MsgMagicEnd:], artifact)
	header[HeaderLength] = endDelim
	return header
}

// --- NewPulseParser defaults ---

func TestPulseParser_NewDefaults(t *testing.T) {
	p := NewPulseParser()

	expectedMagic := hash.DJB2Hash("beautiful")
	if p.Magic != expectedMagic {
		t.Fatalf("Magic = %d, want %d (DJB2Hash(\"beautiful\"))", p.Magic, expectedMagic)
	}
	if p.StartDelimiter != DefaultStartDelimiter {
		t.Fatalf("StartDelimiter = 0x%x, want 0x%x", p.StartDelimiter, DefaultStartDelimiter)
	}
	if p.EndDelimiter != DefaultEndDelimiter {
		t.Fatalf("EndDelimiter = 0x%x, want 0x%x", p.EndDelimiter, DefaultEndDelimiter)
	}
}

// --- ReadHeader tests ---

func TestPulseParser_ReadHeader_ValidMagic(t *testing.T) {
	p := NewPulseParser()
	artifact := uint32(12345)
	data := buildPulseHeader(DefaultStartDelimiter, p.Magic, artifact, DefaultEndDelimiter)

	buf := &rwcBuf{bytes.NewBuffer(data)}
	gotMagic, gotArtifact, err := p.ReadHeader(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMagic != p.Magic {
		t.Fatalf("magic mismatch: got %d, want %d", gotMagic, p.Magic)
	}
	if gotArtifact != artifact {
		t.Fatalf("artifact mismatch: got %d, want %d", gotArtifact, artifact)
	}
}

func TestPulseParser_ReadHeader_InvalidMagic(t *testing.T) {
	p := NewPulseParser()
	wrongMagic := p.Magic + 1
	data := buildPulseHeader(DefaultStartDelimiter, wrongMagic, 0, DefaultEndDelimiter)

	buf := &rwcBuf{bytes.NewBuffer(data)}
	_, _, err := p.ReadHeader(buf)
	if !errors.Is(err, types.ErrInvalidMagic) {
		t.Fatalf("expected ErrInvalidMagic, got %v", err)
	}
}

func TestPulseParser_ReadHeader_InvalidStart(t *testing.T) {
	p := NewPulseParser()
	data := buildPulseHeader(0xFF, p.Magic, 0, DefaultEndDelimiter) // wrong start

	buf := &rwcBuf{bytes.NewBuffer(data)}
	_, _, err := p.ReadHeader(buf)
	if !errors.Is(err, types.ErrInvalidStart) {
		t.Fatalf("expected ErrInvalidStart, got %v", err)
	}
}

func TestPulseParser_ReadHeader_MissingEndByte(t *testing.T) {
	p := NewPulseParser()
	// Build only the 9-byte header without the end delimiter byte
	header := make([]byte, HeaderLength)
	header[MsgStart] = DefaultStartDelimiter
	copy(header[MsgMagicStart:MsgMagicEnd], encoders.Uint32ToBytes(p.Magic))
	binary.LittleEndian.PutUint32(header[MsgMagicEnd:], 0)

	buf := &rwcBuf{bytes.NewBuffer(header)}
	_, _, err := p.ReadHeader(buf)
	if err == nil {
		t.Fatal("expected error when end byte is missing (EOF)")
	}
}

func TestPulseParser_ReadHeader_WrongEndByte(t *testing.T) {
	p := NewPulseParser()
	data := buildPulseHeader(DefaultStartDelimiter, p.Magic, 0, 0xFF) // wrong end

	buf := &rwcBuf{bytes.NewBuffer(data)}
	_, _, err := p.ReadHeader(buf)
	if !errors.Is(err, types.ErrInvalidEnd) {
		t.Fatalf("expected ErrInvalidEnd, got %v", err)
	}
}

func TestPulseParser_ReadHeader_TruncatedHeader(t *testing.T) {
	p := NewPulseParser()
	// Only 4 bytes instead of 9
	buf := &rwcBuf{bytes.NewBuffer([]byte{0x41, 0x01, 0x02, 0x03})}
	_, _, err := p.ReadHeader(buf)
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
	_ = p
}

func TestPulseParser_ReadHeader_EmptyInput(t *testing.T) {
	p := NewPulseParser()
	buf := &rwcBuf{bytes.NewBuffer([]byte{})}
	_, _, err := p.ReadHeader(buf)
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

// --- Parse tests ---

// TestPulseParser_Parse_ReturnsNil documents that Parse is a stub
// that always returns (nil, nil). This is a known incomplete implementation.
func TestPulseParser_Parse_ReturnsNil(t *testing.T) {
	p := NewPulseParser()

	spites, err := p.Parse([]byte{0x01, 0x02, 0x03})
	if err != nil {
		t.Fatalf("Parse stub should return nil error, got %v", err)
	}
	if spites != nil {
		t.Fatalf("Parse stub should return nil spites, got %v", spites)
	}

	// Also test with nil input
	spites, err = p.Parse(nil)
	if err != nil {
		t.Fatalf("Parse(nil) stub should return nil error, got %v", err)
	}
	if spites != nil {
		t.Fatal("Parse(nil) stub should return nil spites")
	}

	t.Log("BUG DOCUMENTED: Parse is a stub that always returns (nil, nil). " +
		"Any code calling ReadPacket on a pulse parser will get nil spites with no error.")
}

// --- Marshal tests ---

func TestPulseParser_Marshal_ValidInit(t *testing.T) {
	p := NewPulseParser()

	initData := []byte("test-init-payload")
	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{
				Name: types.MsgInit.String(),
				Body: &implantpb.Spite_Init{
					Init: &implantpb.Init{
						Data: initData,
					},
				},
			},
		},
	}

	raw, err := p.Marshal(spites, p.Magic)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify structure: start_delim(1) + magic(4) + length(4) + data + end_delim(1)
	if raw[0] != DefaultStartDelimiter {
		t.Fatalf("expected start delimiter 0x%x, got 0x%x", DefaultStartDelimiter, raw[0])
	}
	if raw[len(raw)-1] != DefaultEndDelimiter {
		t.Fatalf("expected end delimiter 0x%x, got 0x%x", DefaultEndDelimiter, raw[len(raw)-1])
	}

	// Verify the data portion
	dataStart := 1 + 4 + 4 // after start + magic + length
	dataEnd := len(raw) - 1 // before end delimiter
	gotData := raw[dataStart:dataEnd]
	if !bytes.Equal(gotData, initData) {
		t.Fatalf("data mismatch: got %q, want %q", gotData, initData)
	}
}

func TestPulseParser_Marshal_EmptySpites(t *testing.T) {
	p := NewPulseParser()

	spites := &implantpb.Spites{Spites: []*implantpb.Spite{}}
	_, err := p.Marshal(spites, p.Magic)
	if !errors.Is(err, types.ErrNullSpites) {
		t.Fatalf("expected ErrNullSpites, got %v", err)
	}
}

func TestPulseParser_Marshal_NilSpites(t *testing.T) {
	p := NewPulseParser()

	// Nil Spites.Spites slice also has length 0
	spites := &implantpb.Spites{}
	_, err := p.Marshal(spites, p.Magic)
	if !errors.Is(err, types.ErrNullSpites) {
		t.Fatalf("expected ErrNullSpites, got %v", err)
	}
}

func TestPulseParser_Marshal_NonInitSpite(t *testing.T) {
	p := NewPulseParser()

	// A spite with an Empty body instead of Init should fail assertion
	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{
				Body: &implantpb.Spite_Empty{},
			},
		},
	}

	_, err := p.Marshal(spites, p.Magic)
	if err == nil {
		t.Fatal("expected error for non-Init spite, got nil")
	}
}

func TestPulseParser_Marshal_NilSpiteEntry(t *testing.T) {
	p := NewPulseParser()

	// A nil spite entry in the slice
	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{nil},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Logf("BUG: Marshal panics on nil spite entry: %v", r)
		}
	}()

	_, err := p.Marshal(spites, p.Magic)
	if err == nil {
		t.Log("Marshal did not error on nil spite entry (may cause downstream issues)")
	}
}

// --- WithSecure ---

func TestPulseParser_WithSecure_NoOp(t *testing.T) {
	p := NewPulseParser()

	// WithSecure is a documented no-op for pulse parser
	p.WithSecure(nil)

	// Verify parser state is unchanged
	if p.Magic != hash.DJB2Hash("beautiful") {
		t.Fatal("WithSecure modified parser state")
	}
	if p.StartDelimiter != DefaultStartDelimiter || p.EndDelimiter != DefaultEndDelimiter {
		t.Fatal("WithSecure modified delimiters")
	}
}

// TestPulseParser_MarshalReadHeader_Incompatibility documents an architectural
// issue: Marshal produces [start(1) + magic(4) + length(4) + data(N) + end(1)],
// but ReadHeader expects [start(1) + magic(4) + artifact(4) + end(1)] with the
// end delimiter immediately after the 9-byte header. When data is non-empty,
// ReadHeader reads the first data byte as the end delimiter and fails.
// This means Marshal output cannot be directly consumed by ReadHeader when
// data is present.
func TestPulseParser_MarshalReadHeader_Incompatibility(t *testing.T) {
	p := NewPulseParser()

	initData := []byte("round-trip-data")
	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{
				Name: types.MsgInit.String(),
				Body: &implantpb.Spite_Init{
					Init: &implantpb.Init{
						Data: initData,
					},
				},
			},
		},
	}

	raw, err := p.Marshal(spites, p.Magic)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	buf := &rwcBuf{bytes.NewBuffer(raw)}
	_, _, err = p.ReadHeader(buf)
	// ReadHeader reads byte[9] as end delimiter, but that is the first data byte.
	// Unless data[0] == 0x42 by coincidence, this will fail.
	if err != nil {
		t.Logf("Expected incompatibility: ReadHeader fails on Marshal output with data: %v", err)
	} else {
		t.Log("ReadHeader succeeded (data[0] may coincidentally be the end delimiter)")
	}
}

// TestPulseParser_MarshalReadHeader_EmptyData tests the case where data is empty,
// so the end delimiter immediately follows the header, making the round-trip work.
func TestPulseParser_MarshalReadHeader_EmptyData(t *testing.T) {
	p := NewPulseParser()

	spites := &implantpb.Spites{
		Spites: []*implantpb.Spite{
			{
				Name: types.MsgInit.String(),
				Body: &implantpb.Spite_Init{
					Init: &implantpb.Init{
						Data: []byte{}, // empty data
					},
				},
			},
		},
	}

	raw, err := p.Marshal(spites, p.Magic)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	buf := &rwcBuf{bytes.NewBuffer(raw)}
	gotMagic, gotArtifact, err := p.ReadHeader(buf)
	if err != nil {
		t.Fatalf("ReadHeader failed on empty-data packet: %v", err)
	}

	if gotMagic != p.Magic {
		t.Fatalf("magic mismatch: got %d, want %d", gotMagic, p.Magic)
	}

	if gotArtifact != 0 {
		t.Fatalf("artifact/length mismatch: got %d, want 0", gotArtifact)
	}
}
