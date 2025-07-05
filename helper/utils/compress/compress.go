package compress

import "github.com/golang/snappy"

// Compress compresses the input data using Snappy and returns the compressed data or an error.
func Compress(data []byte) ([]byte, error) {
	// Use snappy compressor to compress the input data
	compressed := snappy.Encode(nil, data)
	return compressed, nil
}

// Decompress decompresses the input Snappy-compressed data and returns the original data or an error.
func Decompress(compressedData []byte) ([]byte, error) {
	// Use snappy decompressor to decompress the input data
	decompressed, err := snappy.Decode(nil, compressedData)
	if err != nil {
		return nil, err
	}
	return decompressed, nil
}
