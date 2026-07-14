package malefic

import (
	"bytes"
	"strings"
	"testing"
)

func TestMaleficParserReadHeader_RejectsMaxDeclaredLength(t *testing.T) {
	p := NewMaleficParser()
	p.MaxPacketLength = 1024
	header := buildHeader(DefaultStartDelimiter, 0x10203040, ^uint32(0))

	_, _, err := p.ReadHeader(&rwcBuf{Buffer: bytes.NewBuffer(header)})
	if err == nil {
		t.Fatal("ReadHeader accepted a declared body length of math.MaxUint32; want a boundary error before uint32 wraparound")
	}
	if !strings.Contains(err.Error(), "overflows framed uint32 length") {
		t.Fatalf("ReadHeader error = %q, want framed length overflow context", err)
	}
}
