package encoders

import "io/fs"

const (

	// EncoderModulus - The modulus used to calculate the encoder ID from a C2 request nonce
	// *** IMPORTANT *** ENCODER IDs MUST BE LESS THAN THE MODULUS
	EncoderModulus = uint64(65537)
	MaxN           = uint64(9999999)

	// These were chosen at random other than the "No Encoder" ID (0)
	Base32EncoderID  = uint64(65)
	Base58EncoderID  = uint64(43)
	Base64EncoderID  = uint64(131)
	EnglishEncoderID = uint64(31)
	GzipEncoderID    = uint64(49)
	HexEncoderID     = uint64(92)
	PNGEncoderID     = uint64(22)
	NoEncoderID      = uint64(0)
)

// Encoder - Can losslessly encode arbitrary binary data
type Encoder interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

// EncoderFS - Generic interface to read wasm encoders from a filesystem
type EncoderFS interface {
	Open(name string) (fs.File, error)
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}
