package encoders

import (
	"encoding/base32"
)

// Base32 Encoder
type Base32 struct{}

// Missing chars: s, i, l, o
const base32Alphabet = "ab1c2d3e4f5g6h7j8k9m0npqrtuvwxyz"

var sliverBase32 = base32.NewEncoding(base32Alphabet).WithPadding(base32.NoPadding)

// Encode - Base32 Encode
func (e Base32) Encode(data []byte) ([]byte, error) {
	return []byte(sliverBase32.EncodeToString(data)), nil
}

// Decode - Base32 Decode
func (e Base32) Decode(data []byte) ([]byte, error) {
	return sliverBase32.DecodeString(string(data))
}
