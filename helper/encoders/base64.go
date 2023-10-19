package encoders

import "encoding/base64"

// Base64 Encoder
type Base64 struct{}

var base64Alphabet = "a0b2c5def6hijklmnopqr_st-uvwxyzA1B3C4DEFGHIJKLM7NO9PQR8ST+UVWXYZ"
var sliverBase64 = base64.NewEncoding(base64Alphabet).WithPadding(base64.NoPadding)

// Encode - Base64 Encode
func (e Base64) Encode(data []byte) ([]byte, error) {
	return []byte(sliverBase64.EncodeToString(data)), nil
}

// Decode - Base64 Decode
func (e Base64) Decode(data []byte) ([]byte, error) {
	return sliverBase64.DecodeString(string(data))
}
