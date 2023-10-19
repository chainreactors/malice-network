package encoders

import (
	"bytes"
	"compress/gzip"
	"sync"
)

// Gzip - Gzip compression encoder
type Gzip struct{}

var gzipWriterPools = &sync.Pool{}

func init() {
	gzipWriterPools = &sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			return w
		},
	}
}

// GzipBuf - Gzip a buffer
func GzipBuf(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzipWriterPools.Get().(*gzip.Writer)
	gzipWriter.Reset(&buf)
	gzipWriter.Write(data)
	gzipWriter.Close()
	gzipWriterPools.Put(gzipWriter)
	return buf.Bytes(), nil
}

// GzipBufBestCompression - Gzip a buffer using the best compression setting
func GzipBufBestCompression(data []byte) []byte {
	gzipWriter, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
	var buf bytes.Buffer
	gzipWriter.Reset(&buf)
	gzipWriter.Write(data)
	gzipWriter.Close()
	return buf.Bytes()
}

// GunzipBuf - Gunzip a buffer
func GunzipBuf(data []byte) []byte {
	zip, _ := gzip.NewReader(bytes.NewBuffer(data))
	var buf bytes.Buffer
	buf.ReadFrom(zip)
	return buf.Bytes()
}

// Encode - Compress data with gzip
func (g Gzip) Encode(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzipWriterPools.Get().(*gzip.Writer)
	gzipWriter.Reset(&buf)
	gzipWriter.Write(data)
	gzipWriter.Close()
	gzipWriterPools.Put(gzipWriter)
	return buf.Bytes(), nil
}

// Decode - Uncompressed data with gzip
func (g Gzip) Decode(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(reader)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
