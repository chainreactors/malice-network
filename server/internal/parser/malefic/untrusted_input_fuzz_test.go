package malefic

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/chainreactors/IoM-go/types"
	"github.com/golang/snappy"
)

const fuzzMaxPayloadLength = 64 << 10

func FuzzMaleficParserReadHeader(f *testing.F) {
	seeds := [][]byte{
		nil,
		{DefaultStartDelimiter},
		buildHeader(DefaultStartDelimiter, 1, 0),
		buildHeader(DefaultStartDelimiter, 0x10203040, 1024),
		buildHeader(DefaultStartDelimiter, 0x10203040, ^uint32(0)-1),
		buildHeader(DefaultStartDelimiter, 0x10203040, ^uint32(0)),
		buildHeader(0xff, 1, 16),
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		// ReadHeader consumes exactly nine bytes. Capping the corpus input avoids
		// spending fuzz resources on irrelevant trailing data.
		if len(data) > HeaderLength+16 {
			return
		}

		p := NewMaleficParser()
		p.MaxPacketLength = 1024
		sid, length, err := p.ReadHeader(&rwcBuf{Buffer: bytes.NewBuffer(data)})

		if len(data) < HeaderLength {
			if err == nil {
				t.Fatal("truncated header was accepted")
			}
			return
		}
		if data[MsgStart] != DefaultStartDelimiter {
			if !errors.Is(err, types.ErrInvalidStart) {
				t.Fatalf("invalid delimiter error = %v, want ErrInvalidStart", err)
			}
			return
		}
		// A complete header may be rejected by current or future boundary checks.
		// If it is accepted, its decoded fields must still be internally consistent.
		if err != nil {
			return
		}

		wantSID := binary.LittleEndian.Uint32(data[MsgSessionStart:MsgSessionEnd])
		wantLength := binary.LittleEndian.Uint32(data[MsgSessionEnd:HeaderLength]) + 1
		if sid != wantSID || length != wantLength {
			t.Fatalf("ReadHeader() = (%d, %d), want (%d, %d)", sid, length, wantSID, wantLength)
		}
	})
}

func FuzzMaleficParserParse(f *testing.F) {
	p := NewMaleficParser()
	f.Add([]byte(nil))
	f.Add([]byte{DefaultEndDelimiter})
	f.Add([]byte{0x00, DefaultEndDelimiter})
	f.Add([]byte{0xff, 0xff, DefaultEndDelimiter})

	f.Fuzz(func(t *testing.T, payload []byte) {
		if len(payload) > fuzzMaxPayloadLength {
			return
		}

		// A tiny Snappy frame can declare a very large decoded size. Check the
		// metadata before calling production Parse so this fuzz target cannot
		// exhaust the host while still exercising malformed and bounded frames.
		if len(payload) > 0 && payload[len(payload)-1] == DefaultEndDelimiter {
			decodedLength, err := snappy.DecodedLen(payload[:len(payload)-1])
			if err == nil {
				if uint64(decodedLength) > maxDecodedPayloadSize {
					_, parseErr := p.Parse(payload)
					if !errors.Is(parseErr, ErrDecodedPayloadTooLarge) {
						t.Fatalf("oversized decoded payload error = %v, want ErrDecodedPayloadTooLarge", parseErr)
					}
					return
				}
				if decodedLength > fuzzMaxPayloadLength {
					return
				}
			}
		}

		_, _ = p.Parse(payload)
	})
}
