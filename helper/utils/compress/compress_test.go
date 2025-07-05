package compress

import (
	"fmt"
	"testing"
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
