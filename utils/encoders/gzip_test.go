package encoders

import (
	"bytes"
	"crypto/rand"
	insecureRand "math/rand"
	"testing"
)

func TestGzip(t *testing.T) {
	sample := randomData()

	gzip := new(Gzip)
	output, _ := gzip.Encode(sample)
	data, err := gzip.Decode(output)
	if err != nil {
		t.Errorf("gzip decode returned an error %v", err)
	}
	if !bytes.Equal(data, sample) {
		t.Logf("sample = %#v", sample)
		t.Logf("output = %#v", output)
		t.Logf("  data = %#v", data)
		t.Errorf("sample does not match returned\n%#v != %#v", sample, data)
	}
}

func randomDataRandomSize(maxSize int) []byte {
	buf := make([]byte, insecureRand.Intn(maxSize))
	rand.Read(buf)
	return buf
}

func TestGzipGunzip(t *testing.T) {
	for i := 0; i < 100; i++ {
		data := randomDataRandomSize(8192)
		gzipData, _ := GzipBuf(data)
		gunzipData := GunzipBuf(gzipData)
		if !bytes.Equal(data, gunzipData) {
			t.Fatalf("Data does not match")
		}
	}
}
