package compress

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/snappy"
)

// TestSnappyCompression tests the compress and decompress functionality using Snappy.
func TestSnappyCompression(t *testing.T) {
	// Original data to compress
	originalData := []byte("Hello, Snappy compression testing in Golang!")

	// Step 1: Compress the data
	compressedData, err := Compress(originalData)
	if err != nil {
		t.Fatalf("Failed to compress data: %v", err)
	}
	fmt.Println(compressedData)
	// Ensure that the compressed data is not nil or empty
	if len(compressedData) == 0 {
		t.Fatal("Compressed data is empty")
	}

	// Step 2: Decompress the data
	decompressedData, err := Decompress(compressedData)
	if err != nil {
		t.Fatalf("Failed to decompress data: %v", err)
	}

	// Step 3: Compare original data with decompressed data
	if string(decompressedData) != string(originalData) {
		t.Errorf("Decompressed data doesn't match original data.\nExpected: %s\nGot: %s", originalData, decompressedData)
	}
}

func TestValidateSnappyBlock(t *testing.T) {
	for _, original := range [][]byte{
		nil,
		[]byte("small payload"),
		bytes.Repeat([]byte("compressible"), 1024),
	} {
		compressed := snappy.Encode(nil, original)
		decodedLength, err := snappy.DecodedLen(compressed)
		if err != nil {
			t.Fatalf("decoded length: %v", err)
		}
		if err := ValidateSnappyBlock(compressed, decodedLength); err != nil {
			t.Fatalf("ValidateSnappyBlock rejected valid block: %v", err)
		}
	}
}

func TestValidateSnappyBlockRejectsHeaderOnlyLargeBlock(t *testing.T) {
	var header [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(header[:], 2<<30)
	if err := ValidateSnappyBlock(header[:n], 2<<30); !errors.Is(err, snappy.ErrCorrupt) {
		t.Fatalf("ValidateSnappyBlock error = %v, want snappy.ErrCorrupt", err)
	}
}

func FuzzValidateSnappyBlockMatchesDecoder(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		snappy.Encode(nil, []byte("valid")),
		snappy.Encode(nil, bytes.Repeat([]byte("copy"), 1024)),
		{0x05},
		{0x01, 0x01, 0x00},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, compressed []byte) {
		decodedLength, err := snappy.DecodedLen(compressed)
		if err != nil || decodedLength > 1<<20 {
			return
		}
		validateErr := ValidateSnappyBlock(compressed, decodedLength)
		_, decodeErr := snappy.Decode(nil, compressed)
		if (validateErr == nil) != (decodeErr == nil) {
			t.Fatalf("validator/decode mismatch: validate=%v decode=%v data=%x", validateErr, decodeErr, compressed)
		}
	})
}
