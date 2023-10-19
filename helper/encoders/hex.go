package encoders

import "encoding/hex"

// Hex Encoder
type Hex struct{}

// Encode - Hex Encode
func (e Hex) Encode(data []byte) ([]byte, error) {
	return []byte(hex.EncodeToString(data)), nil
}

// Decode - Hex Decode
func (e Hex) Decode(data []byte) ([]byte, error) {
	return hex.DecodeString(string(data))
}
